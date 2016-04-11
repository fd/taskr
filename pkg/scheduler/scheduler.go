package scheduler

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/limbo-services/core/runtime/limbo"
	"github.com/limbo-services/core/runtime/router"
	"github.com/limbo-services/proc"
	"github.com/limbo-services/trace"
	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	"google.golang.org/cloud/datastore"
	"google.golang.org/cloud/pubsub"
	"google.golang.org/grpc"

	"github.com/fd/taskr/pkg/api"
)

const bucketDuration = 10 * time.Second

var errConflict = errors.New("conflict")

func New(projectID, schedulerName string, apiAddr string, numWorkers int) proc.Runner {
	return func(ctx context.Context) <-chan error {
		out := make(chan error)
		go func() {
			defer close(out)

			if numWorkers <= 0 {
				numWorkers = runtime.NumCPU()
			}
			if apiAddr == "" {
				apiAddr = ":80"
			}

			ctx = datastore.WithNamespace(ctx, schedulerName)

			s := scheduler{
				projectID:     projectID,
				schedulerName: schedulerName,
				messages:      make(chan *pubsub.Message),
				clockInterval: time.Second * 10,
			}

			err := s.init(ctx)
			if err != nil {
				out <- err
				return
			}

			log.Printf("running: %q %q", projectID, schedulerName)

			var runners = []proc.Runner{
				s.pull,
				s.clock,
				proc.ServeHTTP(apiAddr, s.makeAPIServer(ctx)),
			}

			for i := 0; i < numWorkers; i++ {
				runners = append(runners, s.worker)
			}

			for err := range proc.Run(ctx, runners...) {
				out <- err
			}
		}()
		return out
	}
}

type scheduler struct {
	projectID     string
	schedulerName string
	messages      chan *pubsub.Message
	clockInterval time.Duration

	datastore    *datastore.Client
	pubsub       *pubsub.Client
	topic        *pubsub.TopicHandle
	subscription *pubsub.SubscriptionHandle
}

type Task struct {
	ID                  string
	Name                string    `datastore:",noindex"`
	QueueName           string    `datastore:",noindex"`
	WaitUntil           time.Time `datastore:",noindex"`
	Payload             []byte    `datastore:",noindex"`
	PendingDependencies []string
	Dependencies        []string
	DependencyNames     []string `datastore:",noindex"`

	Created   time.Time `datastore:",noindex"`
	Updated   time.Time `datastore:",noindex"`
	Scheduled time.Time `datastore:",noindex"`
	Completed time.Time `datastore:",noindex"`

	WaitUntilBucket string `datastore:",noindex"`
	CreateToken     string `datastore:",noindex"`
	ScheduleToken   string `datastore:",noindex"`
	CompleteToken   string `datastore:",noindex"`
}

type WaitingTask struct {
	ID        string
	WaitUntil time.Time
}

type WaitingBucket struct {
	WaitUntil     time.Time
	Completed     bool
	CompleteToken string
}

type Clock struct {
	Now    time.Time
	TickID string
}

func (s *scheduler) init(ctx context.Context) error {

	pubsubClient, err := pubsub.NewClient(ctx, s.projectID)
	if err != nil {
		return err
	}

	datastoreClient, err := datastore.NewClient(ctx, s.projectID)
	if err != nil {
		return err
	}

	s.pubsub = pubsubClient
	s.datastore = datastoreClient

	for {
		topic := pubsubClient.Topic(s.schedulerName)

		ok, err := topic.Exists(ctx)
		if err != nil {
			return err
		}
		if ok {
			s.topic = topic
			break
		}

		_, err = pubsubClient.NewTopic(ctx, s.schedulerName)
		if e, ok := err.(*googleapi.Error); ok && e.Code == http.StatusConflict {
			continue
		}
		if err != nil {
			return err
		}
	}

	for {
		sub := pubsubClient.Subscription(s.schedulerName)

		ok, err := sub.Exists(ctx)
		if err != nil {
			return err
		}
		if ok {
			s.subscription = sub
			break
		}

		_, err = pubsubClient.NewSubscription(ctx, s.schedulerName, s.topic, 0, nil)
		if e, ok := err.(*googleapi.Error); ok && e.Code == http.StatusConflict {
			continue
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *scheduler) makeAPIServer(parentCtx context.Context) *http.Server {
	var (
		routes     = &router.Router{}
		httpServer = &http.Server{}
		grpcServer = grpc.NewServer()
	)

	httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, ctx := trace.New(parentCtx, "HTTP", trace.WithPanicGuard)
		defer span.Close()

		span.Metadata["http/version"] = r.Proto
		span.Metadata["http/method"] = r.Method
		span.Metadata["http/host"] = r.Host
		span.Metadata["http/path"] = r.RequestURI
		if ref := r.Referer(); ref != "" {
			span.Metadata["http/referer"] = ref
		}
		if ua := r.UserAgent(); ua != "" {
			span.Metadata["http/user_agent"] = ua
		}
		if r.ContentLength > 0 {
			span.Metadata["http/content_length"] = strconv.FormatInt(r.ContentLength, 10)
		}
		if len(r.TransferEncoding) > 0 {
			span.Metadata["http/transfer_encoding"] = strings.Join(r.TransferEncoding, " ")
		}

		w.Header().Set("X-Trace", span.TraceID())

		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Expose-Headers", "X-Trace")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "3600")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}

		err := routes.ServeHTTP(ctx, w, r)
		if err != nil {
			span.Error(err)
			// http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	api.RegisterTasksServer(grpcServer, &taskServer{s},
		limbo.WithTracer(),
		limbo.WithGateway(routes))

	return httpServer
}

func (s *scheduler) clock(ctx context.Context) <-chan error {
	out := make(chan error)
	go func() {
		defer close(out)

		var now time.Time

		ticker := time.NewTicker(100 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case newTime := <-ticker.C:
				newTime = newTime.UTC().Truncate(s.clockInterval)
				if newTime.After(now) {
					now = newTime
					s.clockTick(ctx, now)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (s *scheduler) clockTick(ctx context.Context, now time.Time) {
	tickID := uuid.New()

	_, err := s.datastore.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var clock Clock
		var key = datastore.NewKey(ctx, "Clock", "clock", 0, nil)

		err := tx.Get(key, &clock)
		if err == datastore.ErrNoSuchEntity {
			err = nil
		}
		if err != nil {
			return err
		}

		if !clock.Now.Before(now) {
			return errConflict
		}

		clock.Now = now
		clock.TickID = tickID

		_, err = tx.Put(key, &clock)
		if err != nil {
			return err
		}

		return nil
	})
	if err == errConflict {
		return
	}
	if err != nil {
		log.Printf("error: %s", err)
		return
	}

	_, err = s.topic.Publish(ctx, &pubsub.Message{
		Data: []byte(fmt.Sprintf("tick %s\n", tickID)),
	})
	if err != nil {
		log.Printf("error: %s", err)
		return
	}
}

func (s *scheduler) pull(ctx context.Context) <-chan error {
	out := make(chan error)
	go func() {
		defer close(out)

		iter, err := s.subscription.Pull(ctx,
			pubsub.MaxPrefetch(100),
			pubsub.MaxExtension(10*time.Minute))
		if err != nil {
			out <- err
			return
		}
		defer iter.Stop()

		go func() {
			<-ctx.Done()
			iter.Stop()
		}()

		for {
			msg, err := iter.Next()
			if err == io.EOF {
				return
			}
			if err != nil {
				out <- err
				return
			}

			select {
			case s.messages <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (s *scheduler) worker(ctx context.Context) <-chan error {
	out := make(chan error)
	go func() {
		defer close(out)

		for {
			select {
			case msg := <-s.messages:
				s.processMessage(ctx, msg)
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (s *scheduler) processMessage(ctx context.Context, msg *pubsub.Message) {
	defer msg.Done(false)

	var (
		op       = string(msg.Data)
		taskID   string
		tickID   string
		bucketID string
		token    string
	)

	log.Printf("op: %s", op)

	if n, e := fmt.Sscanf(op, "evaluate %s", &taskID); e == nil && n == 1 {
		err := s.evaluateTask(ctx, taskID)
		if err != nil {
			log.Printf("error: %s", err)
		} else {
			msg.Done(true)
		}
	}

	if n, e := fmt.Sscanf(op, "schedule %s %s", &taskID, &token); e == nil && n == 2 {
		err := s.scheduleTask(ctx, taskID, token)
		if err != nil {
			log.Printf("error: %s", err)
		} else {
			msg.Done(true)
		}
	}

	if n, e := fmt.Sscanf(op, "complete %s", &taskID); e == nil && n == 1 {
		err := s.completeTask(ctx, taskID, msg.ID)
		if err != nil {
			log.Printf("error: %s", err)
		} else {
			msg.Done(true)
		}
	}

	if n, e := fmt.Sscanf(op, "tick %s", &tickID); e == nil && n == 1 {
		err := s.scheduleBuckets(ctx, tickID)
		if err != nil {
			log.Printf("error: %s", err)
		} else {
			msg.Done(true)
		}
	}

	if strings.HasPrefix(op, "schedule-bucket ") {
		bucketID = strings.TrimPrefix(op, "schedule-bucket ")
		bucketID = strings.TrimSpace(bucketID)
		err := s.scheduleBucket(ctx, bucketID, msg.ID)
		if err != nil {
			log.Printf("error: %s", err)
		} else {
			msg.Done(true)
		}
	}

}

func (s *scheduler) createTask(ctx context.Context, task *Task, token, procID string) error {
	_, err := s.datastore.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		task.ID = makeID(task.Name)

		var current Task
		var key = taskKey(ctx, task.ID)

		err := tx.Get(key, &current)
		if err == nil {
			if current.CreateToken == token {
				return nil
			}
			return errConflict
		}
		if err != datastore.ErrNoSuchEntity {
			return err
		}

		now := time.Now().UTC()

		current.Name = task.Name
		current.WaitUntil = task.WaitUntil
		current.Payload = task.Payload
		current.DependencyNames = task.DependencyNames
		current.Created = now
		current.Updated = now
		current.CreateToken = token
		current.QueueName = task.QueueName

		current.ID = makeID(task.Name)
		current.DependencyNames = uniqueStrings(task.DependencyNames)
		for _, dep := range current.DependencyNames {
			depID := makeID(dep)
			current.Dependencies = append(current.Dependencies, depID)
			current.PendingDependencies = append(current.PendingDependencies, depID)
		}
		sort.Strings(current.Dependencies)
		sort.Strings(current.PendingDependencies)

		var waitUntilBucket *datastore.Key
		if !current.WaitUntil.IsZero() {
			current.WaitUntil = current.WaitUntil.UTC()
			waitUntilBucket = waitingBucketKey(ctx, procID, current.WaitUntil)
			current.WaitUntilBucket = waitUntilBucket.Name()
		}

		if !current.WaitUntil.IsZero() {
			var (
				bucket WaitingBucket
			)

			err = tx.Get(waitUntilBucket, &bucket)
			if err == datastore.ErrNoSuchEntity {
				bucket.WaitUntil = current.WaitUntil
				_, err = tx.Put(waitUntilBucket, &bucket)
			}
			if err != nil {
				return err
			}

			if bucket.Completed {
				waitUntilBucket = nil
				current.WaitUntilBucket = ""
			}
		}

		if waitUntilBucket != nil {
			waitingTask := WaitingTask{ID: current.ID, WaitUntil: current.WaitUntil}
			waitingTaskKey := waitingTaskKey(ctx, current.ID, procID, current.WaitUntil)
			_, err = tx.Put(waitingTaskKey, &waitingTask)
			if err != nil {
				return err
			}
		}

		_, err = tx.Put(key, &current)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	_, err = s.topic.Publish(ctx, &pubsub.Message{
		Data: []byte(fmt.Sprintf("evaluate %s\n", task.ID)),
	})
	if err != nil {
		return err
	}

	return err
}

func (s *scheduler) rescheduleTask(ctx context.Context, task *Task, procID string) error {
	_, err := s.datastore.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var current Task
		var key = taskKey(ctx, task.ID)

		err := tx.Get(key, &current)
		if err != nil {
			return err
		}

		if current.CompleteToken != "" {
			return errConflict
		}

		now := time.Now().UTC()

		current.WaitUntil = task.WaitUntil
		current.DependencyNames = append(current.DependencyNames, task.DependencyNames...)
		current.Updated = now
		current.ScheduleToken = ""
		current.Scheduled = time.Time{}

		current.DependencyNames = uniqueStrings(task.DependencyNames)
		for _, dep := range current.DependencyNames {
			depID := makeID(dep)
			current.Dependencies = append(current.Dependencies, depID)
			current.PendingDependencies = append(current.PendingDependencies, depID)
		}
		sort.Strings(current.Dependencies)
		sort.Strings(current.PendingDependencies)

		var waitUntilBucket *datastore.Key
		if !current.WaitUntil.IsZero() {
			current.WaitUntil = current.WaitUntil.UTC()
			waitUntilBucket = waitingBucketKey(ctx, procID, current.WaitUntil)
			current.WaitUntilBucket = waitUntilBucket.Name()
		}

		if !current.WaitUntil.IsZero() {
			var (
				bucket WaitingBucket
			)

			err = tx.Get(waitUntilBucket, &bucket)
			if err == datastore.ErrNoSuchEntity {
				bucket.WaitUntil = current.WaitUntil
				_, err = tx.Put(waitUntilBucket, &bucket)
			}
			if err != nil {
				return err
			}

			if bucket.Completed {
				waitUntilBucket = nil
				current.WaitUntilBucket = ""
			}
		}

		if waitUntilBucket != nil {
			waitingTask := WaitingTask{ID: current.ID, WaitUntil: current.WaitUntil}
			waitingTaskKey := waitingTaskKey(ctx, current.ID, procID, current.WaitUntil)
			_, err = tx.Put(waitingTaskKey, &waitingTask)
			if err != nil {
				return err
			}
		}

		_, err = tx.Put(key, &current)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	_, err = s.topic.Publish(ctx, &pubsub.Message{
		Data: []byte(fmt.Sprintf("evaluate %s\n", task.ID)),
	})
	if err != nil {
		return err
	}

	return err
}

func (s *scheduler) evaluateTask(ctx context.Context, taskID string) error {
	var key = taskKey(ctx, taskID)
	var ready bool

	_, err := s.datastore.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		ready = false
		var current Task

		err := tx.Get(key, &current)
		if err != nil {
			return err
		}

		depIDs := current.PendingDependencies
		if len(depIDs) > 10 {
			depIDs = depIDs[:10]
			current.PendingDependencies = current.PendingDependencies[10:]
		} else {
			current.PendingDependencies = nil
		}
		if len(depIDs) > 0 {
			depIDs = uniqueStrings(depIDs)
			var depKeys = make([]*datastore.Key, 0, len(depIDs))
			var deps = make([]*Task, len(depIDs))

			for _, dep := range depIDs {
				depKeys = append(depKeys, taskKey(ctx, dep))
			}

			err = tx.GetMulti(depKeys, deps)
			if err != nil {
				return err
			}

			for _, dep := range deps {
				if dep == nil {
					continue
				}
				if dep.Completed.IsZero() {
					current.PendingDependencies = append(current.PendingDependencies, dep.ID)
				}
			}

			sort.Strings(current.PendingDependencies)
		}

		if current.WaitUntilBucket != "" {
			var bucket WaitingBucket
			var bucketKey = datastore.NewKey(ctx, "WaitingBucket", current.WaitUntilBucket, 0, nil)

			err = tx.Get(bucketKey, &bucket)
			if err != nil {
				return err
			}

			if bucket.Completed {
				current.WaitUntilBucket = ""
			}
		}

		_, err = tx.Put(key, &current)
		if err != nil {
			return err
		}

		if len(current.PendingDependencies) == 0 &&
			current.WaitUntilBucket == "" &&
			current.CompleteToken == "" {
			ready = true
		}

		return nil
	})
	if err != nil {
		return err
	}

	if ready {
		_, err = s.topic.Publish(ctx, &pubsub.Message{
			Data: []byte(fmt.Sprintf("schedule %s %s\n", taskID, uuid.New())),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *scheduler) scheduleTask(ctx context.Context, taskID, token string) error {
	var key = taskKey(ctx, taskID)
	var current Task

	_, err := s.datastore.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		current = Task{}

		err := tx.Get(key, &current)
		if err != nil {
			return err
		}

		if current.ScheduleToken != "" {
			if current.ScheduleToken != token {
				return errConflict
			}
			if current.ScheduleToken == token {
				return nil
			}
		}

		current.ScheduleToken = token
		current.Scheduled = time.Now().UTC()

		_, err = tx.Put(key, &current)
		if err != nil {
			return err
		}

		return nil
	})
	if err == errConflict {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = s.pubsub.Topic(current.QueueName).Publish(ctx, &pubsub.Message{
		Attributes: map[string]string{
			"task-id": current.ID,
			"token":   current.ScheduleToken,
		},
		Data: current.Payload,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *scheduler) completeTask(ctx context.Context, taskID, token string) error {
	var key = taskKey(ctx, taskID)
	var current Task

	_, err := s.datastore.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		current = Task{}

		err := tx.Get(key, &current)
		if err != nil {
			return err
		}

		if current.CompleteToken != "" {
			if current.CompleteToken != token {
				return errConflict
			}
			if current.CompleteToken == token {
				return nil
			}
		}

		current.CompleteToken = token
		current.Completed = time.Now().UTC()

		_, err = tx.Put(key, &current)
		if err != nil {
			return err
		}

		return nil
	})
	if err == errConflict {
		return nil
	}
	if err != nil {
		return err
	}

	query := datastore.NewQuery("Task").
		KeysOnly().
		Filter("Dependencies =", taskID)
	var pubsubMessages []*pubsub.Message

	iter := s.datastore.Run(ctx, query)
	for {
		key, err := iter.Next(nil)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return err
		}

		pubsubMessages = append(pubsubMessages, &pubsub.Message{
			Data: []byte(fmt.Sprintf("evaluate %s\n", key.Name())),
		})

		if len(pubsubMessages) >= 25 {
			_, err := s.topic.Publish(ctx, pubsubMessages...)
			if err != nil {
				return err
			}

			pubsubMessages = pubsubMessages[:0]
		}
	}

	if len(pubsubMessages) > 0 {
		_, err := s.topic.Publish(ctx, pubsubMessages...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *scheduler) scheduleBuckets(ctx context.Context, tickID string) error {
	var (
		clock Clock
		key   = datastore.NewKey(ctx, "Clock", "clock", 0, nil)
	)

	_, err := s.datastore.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		err := tx.Get(key, &clock)
		if err != nil {
			return err
		}

		if clock.TickID != tickID {
			return errConflict
		}

		return nil
	})
	if err == errConflict {
		return nil
	}
	if err != nil {
		return err
	}

	query := datastore.NewQuery("WaitingBucket").
		KeysOnly().
		Filter("WaitUntil <", clock.Now)
	var pubsubMessages []*pubsub.Message

	iter := s.datastore.Run(ctx, query)
	for {
		key, err := iter.Next(nil)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return err
		}

		pubsubMessages = append(pubsubMessages, &pubsub.Message{
			Data: []byte(fmt.Sprintf("schedule-bucket %s\n", key.Name())),
		})

		if len(pubsubMessages) >= 25 {
			_, err := s.topic.Publish(ctx, pubsubMessages...)
			if err != nil {
				return err
			}

			pubsubMessages = pubsubMessages[:0]
		}
	}

	if len(pubsubMessages) > 0 {
		_, err := s.topic.Publish(ctx, pubsubMessages...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *scheduler) scheduleBucket(ctx context.Context, bucketID, token string) error {
	var key = datastore.NewKey(ctx, "WaitingBucket", bucketID, 0, nil)

	_, err := s.datastore.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var bucket WaitingBucket

		err := tx.Get(key, &bucket)
		if err != nil {
			return err
		}

		if bucket.CompleteToken != "" {
			if bucket.CompleteToken == token {
				return nil
			}
			if bucket.CompleteToken != token {
				return errConflict
			}
		}

		bucket.Completed = true
		bucket.CompleteToken = token

		_, err = tx.Put(key, &bucket)
		if err != nil {
			return err
		}

		return nil
	})
	if err == errConflict {
		return nil
	}
	if err != nil {
		return err
	}

	query := datastore.NewQuery("WaitingTask").Ancestor(key).KeysOnly()
	var pubsubMessages []*pubsub.Message

	iter := s.datastore.Run(ctx, query)
	for {
		key, err := iter.Next(nil)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return err
		}

		pubsubMessages = append(pubsubMessages, &pubsub.Message{
			Data: []byte(fmt.Sprintf("evaluate %s\n", key.Name())),
		})

		if len(pubsubMessages) >= 25 {
			_, err := s.topic.Publish(ctx, pubsubMessages...)
			if err != nil {
				return err
			}

			pubsubMessages = pubsubMessages[:0]
		}
	}

	if len(pubsubMessages) > 0 {
		_, err := s.topic.Publish(ctx, pubsubMessages...)
		if err != nil {
			return err
		}
	}

	return nil
}

func taskKey(ctx context.Context, taskID string) *datastore.Key {
	return datastore.NewKey(ctx, "Task", taskID, 0, nil)
}

func waitingBucketKey(ctx context.Context, procID string, at time.Time) *datastore.Key {
	bucket := procID + "/" + at.UTC().Truncate(bucketDuration).String()
	return datastore.NewKey(ctx, "WaitingBucket", bucket, 0, nil)
}

func waitingTaskKey(ctx context.Context, taskID, procID string, at time.Time) *datastore.Key {
	return datastore.NewKey(ctx, "WaitingTask", taskID, 0, waitingBucketKey(ctx, procID, at))
}

func makeID(s string) string {
	h := sha1.New()
	io.WriteString(h, s)
	return hex.EncodeToString(h.Sum(nil))
}

func uniqueStrings(s []string) []string {
	if len(s) == 0 {
		return nil
	}

	sort.Strings(s)
	last := ""
	o := s[:0]
	for _, x := range s {
		if x != last {
			last = x
			o = append(o, x)
		}
	}

	if len(s) == 0 {
		return nil
	}

	return o
}

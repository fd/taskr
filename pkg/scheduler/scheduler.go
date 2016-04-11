package scheduler

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sort"
	"time"

	"github.com/limbo-services/proc"
	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	"google.golang.org/cloud/datastore"
	"google.golang.org/cloud/pubsub"
)

const bucketDuration = 10 * time.Second

var errConflict = errors.New("conflict")

func New(projectID, schedulerName string, numWorkers int) proc.Runner {
	return func(ctx context.Context) <-chan error {
		out := make(chan error)
		go func() {
			defer close(out)

			if numWorkers <= 0 {
				numWorkers = runtime.NumCPU()
			}

			ctx = datastore.WithNamespace(ctx, schedulerName)

			s := scheduler{
				projectID:     projectID,
				schedulerName: schedulerName,
				messages:      make(chan *pubsub.Message),
			}

			err := s.init(ctx)
			if err != nil {
				out <- err
				return
			}

			log.Printf("running: %q %q", projectID, schedulerName)

			var runners = []proc.Runner{
				s.pull,
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

	datastore    *datastore.Client
	pubsub       *pubsub.Client
	topic        *pubsub.TopicHandle
	subscription *pubsub.SubscriptionHandle
}

type Task struct {
	ID                  string
	QueueName           string    `datastore:",noindex"`
	WaitUntil           time.Time `datastore:",noindex"`
	Payload             []byte    `datastore:",noindex"`
	PendingDependencies []string
	Dependencies        []string

	Created   time.Time `datastore:",noindex"`
	Updated   time.Time `datastore:",noindex"`
	Scheduled time.Time `datastore:",noindex"`
	Completed time.Time `datastore:",noindex"`

	WaitUntilBucket *datastore.Key `datastore:",noindex"`
	CreateToken     string         `datastore:",noindex"`
	ScheduleToken   string         `datastore:",noindex"`
	CompleteToken   string         `datastore:",noindex"`
}

type WaitingTask struct {
	ID        string
	WaitUntil time.Time
}

type WaitingBucket struct {
	WaitUntil time.Time
	Completed bool
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
		op     = string(msg.Data)
		taskID string
		token  string
	)

	if n, e := fmt.Sscanf(op, "evaluate %s", &taskID); e == nil && n == 1 {
		err := s.evaluateTask(ctx, taskID)
		if err != nil {
			log.Printf("error: %s", err)
		} else {
			msg.Done(true)
		}
	}

	if n, e := fmt.Sscanf(op, "schedule %s %s", &taskID, &token); e == nil && n == 1 {
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

}

func (s *scheduler) createTask(ctx context.Context, task *Task, token, procID string) error {
	var key = taskKey(ctx, task.ID)

	_, err := s.datastore.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var current Task

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

		current.ID = task.ID
		current.WaitUntil = task.WaitUntil
		current.Payload = task.Payload
		current.Dependencies = task.Dependencies
		current.Created = now
		current.Updated = now
		current.CreateToken = token

		if len(current.Dependencies) == 0 {
			current.Dependencies = nil
		}
		current.PendingDependencies = current.Dependencies

		sort.Strings(current.Dependencies)
		sort.Strings(current.PendingDependencies)

		if !current.WaitUntil.IsZero() {
			current.WaitUntil = current.WaitUntil.UTC()
			current.WaitUntilBucket = waitingBucketKey(ctx, procID, current.WaitUntil)
		}

		if !current.WaitUntil.IsZero() {
			var (
				bucket WaitingBucket
			)

			err = tx.Get(current.WaitUntilBucket, &bucket)
			if err == datastore.ErrNoSuchEntity {
				bucket.WaitUntil = current.WaitUntil
				_, err = tx.Put(current.WaitUntilBucket, &bucket)
			}
			if err != nil {
				return err
			}

			if bucket.Completed {
				current.WaitUntilBucket = nil
			}
		}

		if current.WaitUntilBucket != nil {
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
		}
		if len(depIDs) > 0 {
			var depKeys = make([]*datastore.Key, 0, len(depIDs))
			var deps []*Task

			for _, dep := range depIDs {
				depKeys = append(depKeys, taskKey(ctx, dep))
			}

			err = tx.GetMulti(depKeys, &deps)
			if err != nil {
				return err
			}

			for _, dep := range deps {
				if dep.Completed.IsZero() {
					current.PendingDependencies = append(current.PendingDependencies, dep.ID)
				}
			}

			sort.Strings(current.PendingDependencies)
		}

		if current.WaitUntilBucket != nil {
			var bucket WaitingBucket

			err = tx.Get(current.WaitUntilBucket, &bucket)
			if err != nil {
				return err
			}

			if bucket.Completed {
				current.WaitUntilBucket = nil
			}
		}

		_, err = tx.Put(key, &current)
		if err != nil {
			return err
		}

		if len(current.PendingDependencies) == 0 && current.WaitUntilBucket == nil {
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

		if current.ScheduleToken != token {
			return errConflict
		}
		if current.ScheduleToken == token {
			return nil
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

		if current.CompleteToken != token {
			return errConflict
		}
		if current.CompleteToken == token {
			return nil
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

package taskr

import (
	"io"
	"net/http"
	"time"

	"github.com/limbo-services/proc"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	"google.golang.org/cloud/pubsub"
)

type Worker struct {
	taskrAddr        string
	projectID        string
	topicName        string
	subscriptionName string
	client           *Client
	pubsub           *pubsub.Client
	topic            *pubsub.TopicHandle
	subscription     *pubsub.SubscriptionHandle
	messages         chan *pubsub.Message
}

func NewWorker(taskrAddr, projectID, topic, subscription string) proc.Runner {
	return func(ctx context.Context) <-chan error {
		out := make(chan error)
		go func() {
			defer close(out)

			w := Worker{
				taskrAddr:        taskrAddr,
				projectID:        projectID,
				topicName:        topic,
				subscriptionName: subscription,
				messages:         make(chan *pubsub.Message),
			}

			err := w.init(ctx)
			if err != nil {
				out <- err
				return
			}
		}()
		return out
	}
}

func (w *Worker) init(ctx context.Context) error {

	pubsubClient, err := pubsub.NewClient(ctx, w.projectID)
	if err != nil {
		return err
	}

	client := NewClient(w.taskrAddr)
	if err != nil {
		return err
	}

	for {
		if w.topicName == "" {
			break
		}
		topic := pubsubClient.Topic(w.topicName)

		ok, err := topic.Exists(ctx)
		if err != nil {
			return err
		}
		if ok {
			w.topic = topic
			break
		}

		_, err = pubsubClient.NewTopic(ctx, w.topicName)
		if e, ok := err.(*googleapi.Error); ok && e.Code == http.StatusConflict {
			continue
		}
		if err != nil {
			return err
		}
	}

	for {
		if w.topicName == "" {
			break
		}
		sub := pubsubClient.Subscription(w.subscriptionName)

		ok, err := sub.Exists(ctx)
		if err != nil {
			return err
		}
		if ok {
			w.subscription = sub
			break
		}

		_, err = pubsubClient.NewSubscription(ctx, w.subscriptionName, w.topic, 0, nil)
		if e, ok := err.(*googleapi.Error); ok && e.Code == http.StatusConflict {
			continue
		}
		if err != nil {
			return err
		}
	}

	w.client = client
	w.pubsub = pubsubClient
	return nil
}

func (w *Worker) pull(ctx context.Context) <-chan error {
	out := make(chan error)
	go func() {
		defer close(out)

		iter, err := s.subscription.Pull(ctx,
			pubsub.MaxPrefetch(10),
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
			case w.messages <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (w *Worker) worker(ctx context.Context) <-chan error {
	out := make(chan error)
	go func() {
		defer close(out)

		for {
			select {
			case msg := <-w.messages:
				w.processMessage(ctx, msg)
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (w *Worker) processMessage(ctx context.Context, msg *pubsub.Message) <-chan error {
	defer msg.Done(false)
}

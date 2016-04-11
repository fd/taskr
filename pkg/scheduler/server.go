package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	"google.golang.org/cloud/pubsub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/fd/taskr/pkg/api"
)

type taskServer struct {
	*scheduler
}

func (s *taskServer) Create(ctx context.Context, opts *api.CreateOptions) (*api.Task, error) {
	procID := "proc-0"

	task := Task{
		Name:            opts.ID,
		Payload:         opts.Payload,
		DependencyNames: opts.Dependencies,
		QueueName:       opts.WorkerQueue,
	}

	if task.Name == "" {
		task.Name = uuid.New()
	}
	if opts.WaitUntil != 0 {
		task.WaitUntil = time.Unix(opts.WaitUntil, 0).UTC()
	}
	if task.QueueName == "" {
		return nil, errf(codes.InvalidArgument, "worker_queue must be provided")
	}
	if len(task.Payload) == 0 {
		return nil, errf(codes.InvalidArgument, "payload must be provided")
	}

	err := s.createTask(ctx, &task, opts.Token, procID)
	if err == errConflict {
		return nil, errf(codes.AlreadyExists, "already exists")
	}
	if err != nil {
		log.Printf("error: %s", err)
		return nil, errf(codes.Unavailable, "retry")
	}

	return &api.Task{InternalID: task.ID}, nil
}

func (s *taskServer) Complete(ctx context.Context, opts *api.CompleteOptions) (*api.Task, error) {
	if opts.InternalID == "" {
		return nil, errf(codes.InvalidArgument, "internal_id must be provided")
	}

	_, err := s.topic.Publish(ctx, &pubsub.Message{
		Data: []byte(fmt.Sprintf("complete %s\n", opts.InternalID)),
	})
	if err != nil {
		log.Printf("error: %s", err)
		return nil, errf(codes.Internal, "internal server error")
	}

	return &api.Task{InternalID: opts.InternalID}, nil
}

func (s *taskServer) Reschedule(ctx context.Context, opts *api.RescheduleOptions) (*api.Task, error) {
	procID := "proc-0"

	task := Task{
		ID:              opts.InternalID,
		DependencyNames: opts.Dependencies,
	}

	if opts.WaitUntil != 0 {
		task.WaitUntil = time.Unix(opts.WaitUntil, 0).UTC()
	}
	if opts.InternalID == "" {
		return nil, errf(codes.InvalidArgument, "internal_id must be provided")
	}

	err := s.rescheduleTask(ctx, &task, procID)
	if err == errConflict {
		return nil, errf(codes.AlreadyExists, "already completed")
	}
	if err != nil {
		log.Printf("error: %s", err)
		return nil, errf(codes.Unavailable, "retry")
	}

	return &api.Task{InternalID: task.ID}, nil
}

var errf = grpc.Errorf

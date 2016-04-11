package main

import (
	"log"
	"os"
	"syscall"

	"golang.org/x/net/context"

	"github.com/fd/taskr/pkg/scheduler"
	"github.com/limbo-services/proc"
)

func main() {

	errs := proc.Run(context.Background(),
		proc.TerminateOnSignal(os.Interrupt, syscall.SIGTERM),
		scheduler.New(os.Getenv("PROJECT_ID"), os.Getenv("SCHEDULER_NAME"), 20))

	var exitCode int
	for err := range errs {
		log.Printf("error: %s", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

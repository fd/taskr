package main

import (
	"log"
	"net"
	"os"
	"strings"
	"syscall"

	"golang.org/x/net/context"

	"github.com/fd/taskr/pkg/scheduler"
	"github.com/limbo-services/proc"
	"github.com/limbo-services/trace"
	"github.com/limbo-services/trace/dev"
)

func main() {
	trace.DefaultHandler = dev.NewHandler(nil)

	if prefix := findGCPPrefix(); prefix != "" {
		os.Setenv("PROJECT_ID", os.Getenv(prefix+"_ENV_CLOUDSDK_CORE_PROJECT"))
		os.Setenv("SCHEDULER_NAME", "taskr")
		os.Setenv("DATASTORE_EMULATOR_HOST", net.JoinHostPort(
			os.Getenv(prefix+"_PORT_8490_TCP_ADDR"),
			os.Getenv(prefix+"_PORT_8490_TCP_PORT")))
		os.Setenv("PUBSUB_EMULATOR_HOST", net.JoinHostPort(
			os.Getenv(prefix+"_PORT_8283_TCP_ADDR"),
			os.Getenv(prefix+"_PORT_8283_TCP_PORT")))

		log.Printf("env:")
		log.Printf("  PROJECT_ID              = %q", os.Getenv("PROJECT_ID"))
		log.Printf("  SCHEDULER_NAME          = %q", os.Getenv("SCHEDULER_NAME"))
		log.Printf("  DATASTORE_EMULATOR_HOST = %q", os.Getenv("DATASTORE_EMULATOR_HOST"))
		log.Printf("  PUBSUB_EMULATOR_HOST    = %q", os.Getenv("PUBSUB_EMULATOR_HOST"))
	}

	errs := proc.Run(context.Background(),
		proc.TerminateOnSignal(os.Interrupt, syscall.SIGTERM),
		scheduler.New(os.Getenv("PROJECT_ID"), os.Getenv("SCHEDULER_NAME"), os.Getenv("BIND"), 20))

	var exitCode int
	for err := range errs {
		log.Printf("error: %s", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func findGCPPrefix() string {
	for _, v := range os.Environ() {
		idx := strings.IndexByte(v, '=')
		if idx < 0 {
			continue
		}

		v = v[:idx]

		if strings.HasSuffix(v, "_ENV_CLOUDSDK_CORE_PROJECT") {
			return strings.TrimSuffix(v, "_ENV_CLOUDSDK_CORE_PROJECT")
		}
	}
	return ""
}

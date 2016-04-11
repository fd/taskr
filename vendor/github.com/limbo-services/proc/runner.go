package proc

import (
	"os"
	"os/signal"
	"reflect"

	"golang.org/x/net/context"
)

type Runner func(ctx context.Context) <-chan error

func Run(parentCtx context.Context, runners ...Runner) <-chan error {
	out := make(chan error)
	go func() {
		defer close(out)

		var (
			cases   = make([]reflect.SelectCase, len(runners))
			pending = len(runners)
		)

		ctx, cancel := context.WithCancel(parentCtx)
		defer cancel()

		for i, runner := range runners {
			errChan := runner(ctx)
			cases[i] = reflect.SelectCase{
				Chan: reflect.ValueOf(errChan),
				Dir:  reflect.SelectRecv,
			}
		}

		for pending > 0 {
			chosen, recv, recvOK := reflect.Select(cases)
			// log.Printf("chosen=%v, recv=%v, recvOK=%v", chosen, recv, recvOK)

			if recv.IsValid() && !recv.IsNil() {
				// error received
				err, _ := recv.Interface().(error)
				if err != nil {
					out <- err
				}
			}

			if !recvOK {
				// chanel was closed
				cancel()
				cases[chosen].Chan = reflect.Value{}
				pending--
			}
		}
	}()
	return out
}

func TerminateOnSignal(signals ...os.Signal) Runner {
	return func(ctx context.Context) <-chan error {
		out := make(chan error)
		go func() {
			defer close(out)

			c := make(chan os.Signal)
			defer close(c)

			go signal.Notify(c, signals...)
			defer signal.Stop(c)

			select {
			case <-c:
			case <-ctx.Done():
			}
		}()
		return out
	}
}

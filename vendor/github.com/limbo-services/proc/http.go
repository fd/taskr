package proc

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/net/context"
)

func ServeHTTP(addr string, server *http.Server) Runner {
	return func(ctx context.Context) <-chan error {
		out := make(chan error)
		go func() {
			defer close(out)

			var wg = &sync.WaitGroup{}
			server.Handler = httpWaitGroupHandler(wg, server.Handler)

			l, err := net.Listen("tcp", addr)
			if err != nil {
				out <- err
				return
			}
			defer l.Close()

			go func() {
				<-ctx.Done()
				l.Close()
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				err := server.Serve(l)
				if err != nil {
					if !strings.Contains(err.Error(), "use of closed network connection") {
						out <- err
					}
				}
			}()

			wg.Wait()
		}()
		return out
	}
}

func httpWaitGroupHandler(wg *sync.WaitGroup, h http.Handler) http.HandlerFunc {
	if h == nil {
		h = http.DefaultServeMux
	}

	return func(w http.ResponseWriter, r *http.Request) {
		wg.Add(1)
		defer wg.Done()

		h.ServeHTTP(w, r)
	}
}

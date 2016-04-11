package router

import (
	"net/http"

	"golang.org/x/net/context"
)

func whenMethodIs(m string, h Handler) Handler {
	if m == "" || m == "*" {
		return h
	}

	return HandlerFunc(func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		if req.Method != m {
			return Pass
		}

		return h.ServeHTTP(ctx, rw, req)
	})
}

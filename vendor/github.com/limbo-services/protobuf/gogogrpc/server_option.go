package gogogrpc

import (
	"google.golang.org/grpc"
)

type ServerOptions struct {
	descWrappers []func(*grpc.ServiceDesc, interface{})
	srvWrappers  []func(srv interface{}) interface{}
}

type ServerOption func(options *ServerOptions)

// ApplyServerOptions options to a ServiceDesc and service
func ApplyServerOptions(desc *grpc.ServiceDesc, srv interface{}, options []ServerOption) (*grpc.ServiceDesc, interface{}) {
	desc = CloneServiceDesc(desc)

	var opts ServerOptions

	for _, option := range options {
		option(&opts)
	}

	for i := len(opts.srvWrappers) - 1; i >= 0; i-- {
		srv = opts.srvWrappers[i](srv)
	}

	for i := len(opts.descWrappers) - 1; i >= 0; i-- {
		opts.descWrappers[i](desc, srv)
	}

	return desc, srv
}

func WithServiceDescWrapper(f func(*grpc.ServiceDesc, interface{})) ServerOption {
	return func(options *ServerOptions) {
		options.descWrappers = append(options.descWrappers, f)
	}
}

func WithServiceWrapper(f func(srv interface{}) interface{}) ServerOption {
	return func(options *ServerOptions) {
		options.srvWrappers = append(options.srvWrappers, f)
	}
}

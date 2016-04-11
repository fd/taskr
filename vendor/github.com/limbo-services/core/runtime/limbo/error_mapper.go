package limbo

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/limbo-services/protobuf/gogogrpc"
)

type ErrorMapper interface {
	HandleError(err error) error
}

func WithErrorMapper(mapper ErrorMapper) gogogrpc.ServerOption {
	return gogogrpc.WithServiceDescWrapper(func(desc *grpc.ServiceDesc, _ interface{}) {

		for i, m := range desc.Methods {
			desc.Methods[i] = wrapMethodWithErrorMapper(desc, m, mapper)
		}

		for i, s := range desc.Streams {
			desc.Streams[i] = wrapStreamWithErrorMapper(desc, s, mapper)
		}

	})
}

func wrapMethodWithErrorMapper(srv *grpc.ServiceDesc, desc grpc.MethodDesc, mapper ErrorMapper) grpc.MethodDesc {
	h := desc.Handler
	desc.Handler = func(srv interface{}, ctx context.Context, dec func(interface{}) error) (out interface{}, err error) {
		res, err := h(srv, ctx, dec)
		if err != nil {
			if e := mapper.HandleError(err); e != nil {
				err = e
			}
			return nil, err
		}
		return res, nil
	}
	return desc
}

func wrapStreamWithErrorMapper(srv *grpc.ServiceDesc, desc grpc.StreamDesc, mapper ErrorMapper) grpc.StreamDesc {
	h := desc.Handler
	desc.Handler = func(srv interface{}, stream grpc.ServerStream) (err error) {
		err = h(srv, stream)
		if err != nil {
			if e := mapper.HandleError(err); e != nil {
				err = e
			}
			return err
		}
		return nil
	}
	return desc
}

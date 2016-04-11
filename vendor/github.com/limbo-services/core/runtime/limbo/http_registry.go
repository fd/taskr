package limbo

import (
	"net/http"
	"reflect"

	"github.com/limbo-services/core/runtime/router"
	"github.com/limbo-services/protobuf/gogogrpc"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var gatewayDescriptions = map[reflect.Type]*GatewayDesc{}

func RegisterGatewayDesc(desc *GatewayDesc) {
	rt := reflect.TypeOf(desc.HandlerType)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	gatewayDescriptions[rt] = desc
}

func lookupGatewayDesc(typ interface{}) *GatewayDesc {
	rt := reflect.TypeOf(typ)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	return gatewayDescriptions[rt]
}

type GatewayDesc struct {
	ServiceName string
	HandlerType interface{}
	Routes      []RouteDesc
}

type RouteDesc struct {
	Method  string
	Pattern string
	Handler gatewayHandlerFunc
}

type gatewayHandlerFunc func(desc *grpc.ServiceDesc, srv interface{}, ctx context.Context, rw http.ResponseWriter, req *http.Request) error

func WithGateway(r *router.Router) gogogrpc.ServerOption {
	return gogogrpc.WithServiceDescWrapper(func(desc *grpc.ServiceDesc, srv interface{}) {
		gatewayDesc := lookupGatewayDesc(desc.HandlerType)
		if gatewayDesc == nil {
			return
		}

		// ensure the ServiceDesc doesn't get modified by later options
		desc = gogogrpc.CloneServiceDesc(desc)

		for _, route := range gatewayDesc.Routes {
			r.Addf(route.Method, route.Pattern, wrapServiceWithGateway(desc, srv, route.Handler))
		}
	})
}

func wrapServiceWithGateway(desc *grpc.ServiceDesc, srv interface{}, f gatewayHandlerFunc) router.HandlerFunc {
	return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		return f(desc, srv, ctx, rw, req)
	}
}

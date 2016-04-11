package limbo

import (
	"fmt"
	"reflect"
	"sort"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/limbo-services/protobuf/gogogrpc"
)

var authDescriptions = map[reflect.Type]*ServiceAuthDesc{}
var registeredScopes = map[reflect.Type]map[reflect.Type][]string{}

func RegisterServiceAuthDesc(desc *ServiceAuthDesc) {
	rt := reflect.TypeOf(desc.HandlerType)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	authDescriptions[rt] = desc

	for _, desc := range authDescriptions {
		for _, method := range desc.Methods {
			if method.Authorize == nil {
				continue
			}

			callerRT := reflect.TypeOf(method.CallerType)
			contextRT := reflect.TypeOf(method.ContextType)

			for callerRT != nil && callerRT.Kind() == reflect.Ptr {
				callerRT = callerRT.Elem()
			}
			for contextRT != nil && contextRT.Kind() == reflect.Ptr {
				contextRT = contextRT.Elem()
			}

			m := registeredScopes[callerRT]
			if m == nil {
				m = map[reflect.Type][]string{}
				registeredScopes[callerRT] = m
			}

			n := m[contextRT]
			n = append(n, method.Scope)

			sort.Strings(n)
			o := n[:0]
			last := ""
			for _, s := range n {
				if last != s {
					last = s
					o = append(o, s)
				}
			}

			m[contextRT] = o
		}
		for _, stream := range desc.Streams {
			if stream.Authorize == nil {
				continue
			}

			callerRT := reflect.TypeOf(stream.CallerType)
			contextRT := reflect.TypeOf(stream.ContextType)

			for callerRT != nil && callerRT.Kind() == reflect.Ptr {
				callerRT = callerRT.Elem()
			}
			for contextRT != nil && contextRT.Kind() == reflect.Ptr {
				contextRT = contextRT.Elem()
			}

			m := registeredScopes[callerRT]
			if m == nil {
				m = map[reflect.Type][]string{}
				registeredScopes[callerRT] = m
			}

			n := m[contextRT]
			n = append(n, stream.Scope)

			sort.Strings(n)
			o := n[:0]
			last := ""
			for _, s := range n {
				if last != s {
					last = s
					o = append(o, s)
				}
			}

			m[contextRT] = o
		}
	}
}

func LookupAuthScopes(caller, context interface{}) []string {
	callerRT := reflect.TypeOf(caller)
	contextRT := reflect.TypeOf(context)

	for callerRT != nil && callerRT.Kind() == reflect.Ptr {
		callerRT = callerRT.Elem()
	}
	for contextRT != nil && contextRT.Kind() == reflect.Ptr {
		contextRT = contextRT.Elem()
	}

	m := registeredScopes[callerRT]
	if m == nil {
		return nil
	}

	n := m[contextRT]
	return n
}

func lookupServiceAuthDesc(typ interface{}) *ServiceAuthDesc {
	rt := reflect.TypeOf(typ)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	return authDescriptions[rt]
}

type ServiceAuthDesc struct {
	ServiceName     string
	HandlerType     interface{}
	AuthHandlerType interface{}
	Methods         []MethodAuthDesc
	Streams         []StreamAuthDesc
}

type MethodAuthDesc struct {
	MethodName  string
	CallerType  interface{}
	ContextType interface{}

	GetCaller  func(msg interface{}) interface{}
	SetCaller  func(msg interface{}, caller interface{})
	GetContext func(msg interface{}) interface{}

	Strategies   []string
	Authenticate func(handler interface{}, ctx context.Context, staregies []string, caller interface{}) error

	Scope     string
	Authorize func(handler interface{}, ctx context.Context, scope string, caller, context interface{}) error
}

type StreamAuthDesc struct {
	StreamName  string
	CallerType  interface{}
	ContextType interface{}

	GetCaller  func(msg interface{}) interface{}
	SetCaller  func(msg interface{}, caller interface{})
	GetContext func(msg interface{}) interface{}

	Strategies   []string
	Authenticate func(handler interface{}, ctx context.Context, staregies []string, caller interface{}) error

	Scope     string
	Authorize func(handler interface{}, ctx context.Context, scope string, caller, context interface{}) error
}

func WithAuthGuard(handler interface{}) gogogrpc.ServerOption {
	return gogogrpc.WithServiceDescWrapper(func(desc *grpc.ServiceDesc, _ interface{}) {
		authDesc := lookupServiceAuthDesc(desc.HandlerType)
		if authDesc == nil {
			return
		}

		requiredInterface := reflect.TypeOf(authDesc.AuthHandlerType)
		for requiredInterface.Kind() == reflect.Ptr {
			requiredInterface = requiredInterface.Elem()
		}

		handlerRV := reflect.ValueOf(handler)
		handlerRT := handlerRV.Type()
		if !handlerRT.Implements(requiredInterface) {
			panic(fmt.Sprintf("WithAuthGuard(handler %s) should implement %s", handlerRT.Name(), requiredInterface.Name()))
		}

		for i, methodDesc := range desc.Methods {
			methodAuthDesc := authDesc.Methods[i]

			if methodAuthDesc.Authorize != nil || methodAuthDesc.Authenticate != nil {
				wrapMethodWithAuth(methodAuthDesc, &methodDesc, handler)
				desc.Methods[i] = methodDesc
			}
		}

		for i, streamDesc := range desc.Streams {
			streamAuthDesc := authDesc.Streams[i]

			if streamAuthDesc.Authorize != nil || streamAuthDesc.Authenticate != nil {
				wrapStreamWithAuth(streamAuthDesc, &streamDesc, handler)
				desc.Streams[i] = streamDesc
			}
		}
	})
}

func wrapMethodWithAuth(authDesc MethodAuthDesc, desc *grpc.MethodDesc, handler interface{}) {
	h := desc.Handler
	desc.Handler = func(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
		return h(srv, ctx, func(msg interface{}) error {
			if err := dec(msg); err != nil {
				return err
			}

			var (
				caller  = authDesc.GetCaller(msg)
				context interface{}
				err     error
			)

			if authDesc.Authenticate != nil {
				err = authDesc.Authenticate(handler, ctx, authDesc.Strategies, caller)
				if err != nil {
					return err
				}
				authDesc.SetCaller(msg, caller)
			}

			if authDesc.Authorize != nil {
				if authDesc.GetContext != nil {
					context = authDesc.GetContext(msg)
				}
				err = authDesc.Authorize(handler, ctx, authDesc.Scope, caller, context)
				if err != nil {
					return err
				}
			}

			return nil
		})
	}
}

func wrapStreamWithAuth(authDesc StreamAuthDesc, desc *grpc.StreamDesc, handler interface{}) {
	h := desc.Handler
	desc.Handler = func(srv interface{}, stream grpc.ServerStream) error {
		stream = &authServerStream{stream, authDesc, handler}
		return h(srv, stream)
	}
}

type authServerStream struct {
	grpc.ServerStream
	authDesc StreamAuthDesc
	handler  interface{}
}

func (ss *authServerStream) RecvMsg(msg interface{}) error {
	if err := ss.ServerStream.RecvMsg(msg); err != nil {
		return err
	}

	var (
		authDesc = ss.authDesc
		caller   = authDesc.GetCaller(msg)
		context  interface{}
		err      error
	)

	if authDesc.Authenticate != nil {
		err = authDesc.Authenticate(ss.handler, ss.Context(), authDesc.Strategies, caller)
		if err != nil {
			return err
		}
		authDesc.SetCaller(msg, caller)
	}

	if authDesc.Authorize != nil {
		if authDesc.GetContext != nil {
			context = authDesc.GetContext(msg)
		}
		err = authDesc.Authorize(ss.handler, ss.Context(), authDesc.Scope, caller, context)
		if err != nil {
			return err
		}
	}

	return nil
}

// var _ = ServiceAuthDesc{
// 	Methods: []MethodAuthDesc{
// 		{
// 			Authenticate: _auth_MySrv_MyMethod,
// 		},
// 	},
// }
//
// func _auth_MySrv_MyMethod(handler interface{}, ctx context.Context, staregies []string) (interface{}, error) {
// 	caller, err := handler.(MyAuthHandler).AuthenticateAsCaller(ctx, staregies)
// 	return caller, err
// }

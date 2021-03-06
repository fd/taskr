// Code generated by protoc-gen-gogo.
// source: github.com/fd/taskr/pkg/api/api.proto
// DO NOT EDIT!

/*
	Package api is a generated protocol buffer package.

	It is generated from these files:
		github.com/fd/taskr/pkg/api/api.proto

	It has these top-level messages:
		Task
		CreateOptions
		CompleteOptions
		RescheduleOptions
*/
package api

import proto "github.com/limbo-services/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/limbo-services/core/runtime/limbo"
import _ "github.com/limbo-services/protobuf/gogoproto"

import encoding_json "encoding/json"

import (
	gogogrpc "github.com/limbo-services/protobuf/gogogrpc"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

import github_com_limbo_services_core_runtime_limbo "github.com/limbo-services/core/runtime/limbo"

import golang_org_x_net_context "golang.org/x/net/context"
import google_golang_org_grpc_codes "google.golang.org/grpc/codes"
import google_golang_org_grpc "google.golang.org/grpc"
import net_http "net/http"
import github_com_juju_errors "github.com/juju/errors"
import github_com_limbo_services_core_runtime_router "github.com/limbo-services/core/runtime/router"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Task struct {
	InternalID string `protobuf:"bytes,1,opt,name=internal_id,proto3" json:"internal_id,omitempty"`
}

func (m *Task) Reset()         { *m = Task{} }
func (m *Task) String() string { return proto.CompactTextString(m) }
func (*Task) ProtoMessage()    {}

type CreateOptions struct {
	ID           string                   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Dependencies []string                 `protobuf:"bytes,2,rep,name=dependencies" json:"dependencies,omitempty"`
	WaitUntil    int64                    `protobuf:"varint,3,opt,name=wait_until,proto3" json:"wait_until,omitempty"`
	Payload      encoding_json.RawMessage `protobuf:"bytes,4,opt,name=payload,proto3,casttype=encoding/json.RawMessage" json:"payload,omitempty"`
	Token        string                   `protobuf:"bytes,5,opt,name=token,proto3" json:"token,omitempty"`
	WorkerQueue  string                   `protobuf:"bytes,6,opt,name=worker_queue,proto3" json:"worker_queue,omitempty"`
}

func (m *CreateOptions) Reset()         { *m = CreateOptions{} }
func (m *CreateOptions) String() string { return proto.CompactTextString(m) }
func (*CreateOptions) ProtoMessage()    {}

type CompleteOptions struct {
	InternalID string `protobuf:"bytes,1,opt,name=internal_id,proto3" json:"internal_id,omitempty"`
}

func (m *CompleteOptions) Reset()         { *m = CompleteOptions{} }
func (m *CompleteOptions) String() string { return proto.CompactTextString(m) }
func (*CompleteOptions) ProtoMessage()    {}

type RescheduleOptions struct {
	InternalID   string   `protobuf:"bytes,1,opt,name=internal_id,proto3" json:"internal_id,omitempty"`
	Dependencies []string `protobuf:"bytes,2,rep,name=dependencies" json:"dependencies,omitempty"`
	WaitUntil    int64    `protobuf:"varint,3,opt,name=wait_until,proto3" json:"wait_until,omitempty"`
}

func (m *RescheduleOptions) Reset()         { *m = RescheduleOptions{} }
func (m *RescheduleOptions) String() string { return proto.CompactTextString(m) }
func (*RescheduleOptions) ProtoMessage()    {}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn
var _ gogogrpc.ServerOption

// Client API for Tasks service

type TasksClient interface {
	Create(ctx context.Context, in *CreateOptions, opts ...grpc.CallOption) (*Task, error)
	Complete(ctx context.Context, in *CompleteOptions, opts ...grpc.CallOption) (*Task, error)
	Reschedule(ctx context.Context, in *RescheduleOptions, opts ...grpc.CallOption) (*Task, error)
}

type tasksClient struct {
	cc *grpc.ClientConn
}

func NewTasksClient(cc *grpc.ClientConn) TasksClient {
	return &tasksClient{cc}
}

func (c *tasksClient) Create(ctx context.Context, in *CreateOptions, opts ...grpc.CallOption) (*Task, error) {
	out := new(Task)
	err := grpc.Invoke(ctx, "/taskr.api.Tasks/Create", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tasksClient) Complete(ctx context.Context, in *CompleteOptions, opts ...grpc.CallOption) (*Task, error) {
	out := new(Task)
	err := grpc.Invoke(ctx, "/taskr.api.Tasks/Complete", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tasksClient) Reschedule(ctx context.Context, in *RescheduleOptions, opts ...grpc.CallOption) (*Task, error) {
	out := new(Task)
	err := grpc.Invoke(ctx, "/taskr.api.Tasks/Reschedule", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Tasks service

type TasksServer interface {
	Create(context.Context, *CreateOptions) (*Task, error)
	Complete(context.Context, *CompleteOptions) (*Task, error)
	Reschedule(context.Context, *RescheduleOptions) (*Task, error)
}

func RegisterTasksServer(s *grpc.Server, srv TasksServer, options ...gogogrpc.ServerOption) {
	s.RegisterService(gogogrpc.ApplyServerOptions(&_Tasks_serviceDesc, srv, options))
}

func _Tasks_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(CreateOptions)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(TasksServer).Create(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Tasks_Complete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(CompleteOptions)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(TasksServer).Complete(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Tasks_Reschedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(RescheduleOptions)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(TasksServer).Reschedule(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

var _Tasks_serviceDesc = grpc.ServiceDesc{
	ServiceName: "taskr.api.Tasks",
	HandlerType: (*TasksServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Create",
			Handler:    _Tasks_Create_Handler,
		},
		{
			MethodName: "Complete",
			Handler:    _Tasks_Complete_Handler,
		},
		{
			MethodName: "Reschedule",
			Handler:    _Tasks_Reschedule_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

var jsonSchemaDefscd21e75484a2275e5e2503bf125ee570df74b477 = []github_com_limbo_services_core_runtime_limbo.SchemaDefinition{
	{
		Name:         "taskr.api.CompleteOptions",
		Dependencies: []string{},
		Definition: []byte(`{
			"properties": {
				"internalId": {
					"type": "string"
				}
			},
			"title": "CompleteOptions",
			"type": "object"
		}`),
	},
	{
		Name:         "taskr.api.CreateOptions",
		Dependencies: []string{},
		Definition: []byte(`{
			"properties": {
				"dependencies": {
					"items": {
						"type": "string"
					},
					"type": "array"
				},
				"id": {
					"type": "string"
				},
				"payload": {
					"format": "base64",
					"type": "string"
				},
				"token": {
					"type": "string"
				},
				"waitUntil": {
					"format": "int64",
					"type": "integer"
				},
				"workerQueue": {
					"type": "string"
				}
			},
			"title": "CreateOptions",
			"type": "object"
		}`),
	},
	{
		Name:         "taskr.api.RescheduleOptions",
		Dependencies: []string{},
		Definition: []byte(`{
			"properties": {
				"dependencies": {
					"items": {
						"type": "string"
					},
					"type": "array"
				},
				"internalId": {
					"type": "string"
				},
				"waitUntil": {
					"format": "int64",
					"type": "integer"
				}
			},
			"title": "RescheduleOptions",
			"type": "object"
		}`),
	},
	{
		Name:         "taskr.api.Task",
		Dependencies: []string{},
		Definition: []byte(`{
			"properties": {
				"internalId": {
					"type": "string"
				}
			},
			"title": "Task",
			"type": "object"
		}`),
	},
}
var swaggerDefscd21e75484a2275e5e2503bf125ee570df74b477 = []github_com_limbo_services_core_runtime_limbo.SwaggerOperation{
	{
		Pattern: "/v1/tasks",
		Method:  "POST",
		Dependencies: []string{
			"taskr.api.CreateOptions",
			"taskr.api.Task",
		},
		Definition: []byte(`{
			"description": "",
			"parameters": [
				{
					"in": "body",
					"name": "parameters",
					"schema": {
						"$ref": "taskr.api.CreateOptions"
					}
				}
			],
			"responses": {
				"200": {
					"description": "",
					"schema": {
						"$ref": "taskr.api.Task"
					}
				}
			},
			"tags": [
				"Tasks"
			]
		}`),
	},
	{
		Pattern: "/v1/tasks/{internal_id}/complete",
		Method:  "POST",
		Dependencies: []string{
			"taskr.api.CompleteOptions",
			"taskr.api.Task",
		},
		Definition: []byte(`{
			"description": "",
			"parameters": [
				{
					"in": "path",
					"maxItems": 1,
					"minItems": 1,
					"name": "internal_id",
					"required": true,
					"type": "string"
				},
				{
					"in": "body",
					"name": "parameters",
					"schema": {
						"$ref": "taskr.api.CompleteOptions"
					}
				}
			],
			"responses": {
				"200": {
					"description": "",
					"schema": {
						"$ref": "taskr.api.Task"
					}
				}
			},
			"tags": [
				"Tasks"
			]
		}`),
	},
	{
		Pattern: "/v1/tasks/{internal_id}/reschedule",
		Method:  "POST",
		Dependencies: []string{
			"taskr.api.RescheduleOptions",
			"taskr.api.Task",
		},
		Definition: []byte(`{
			"description": "",
			"parameters": [
				{
					"in": "path",
					"maxItems": 1,
					"minItems": 1,
					"name": "internal_id",
					"required": true,
					"type": "string"
				},
				{
					"in": "body",
					"name": "parameters",
					"schema": {
						"$ref": "taskr.api.RescheduleOptions"
					}
				}
			],
			"responses": {
				"200": {
					"description": "",
					"schema": {
						"$ref": "taskr.api.Task"
					}
				}
			},
			"tags": [
				"Tasks"
			]
		}`),
	},
}

func (m *Task) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *Task) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.InternalID) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintApi(data, i, uint64(len(m.InternalID)))
		i += copy(data[i:], m.InternalID)
	}
	return i, nil
}

func (m *CreateOptions) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *CreateOptions) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.ID) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintApi(data, i, uint64(len(m.ID)))
		i += copy(data[i:], m.ID)
	}
	if len(m.Dependencies) > 0 {
		for _, s := range m.Dependencies {
			data[i] = 0x12
			i++
			l = len(s)
			for l >= 1<<7 {
				data[i] = uint8(uint64(l)&0x7f | 0x80)
				l >>= 7
				i++
			}
			data[i] = uint8(l)
			i++
			i += copy(data[i:], s)
		}
	}
	if m.WaitUntil != 0 {
		data[i] = 0x18
		i++
		i = encodeVarintApi(data, i, uint64(m.WaitUntil))
	}
	if m.Payload != nil {
		if len(m.Payload) > 0 {
			data[i] = 0x22
			i++
			i = encodeVarintApi(data, i, uint64(len(m.Payload)))
			i += copy(data[i:], m.Payload)
		}
	}
	if len(m.Token) > 0 {
		data[i] = 0x2a
		i++
		i = encodeVarintApi(data, i, uint64(len(m.Token)))
		i += copy(data[i:], m.Token)
	}
	if len(m.WorkerQueue) > 0 {
		data[i] = 0x32
		i++
		i = encodeVarintApi(data, i, uint64(len(m.WorkerQueue)))
		i += copy(data[i:], m.WorkerQueue)
	}
	return i, nil
}

func (m *CompleteOptions) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *CompleteOptions) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.InternalID) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintApi(data, i, uint64(len(m.InternalID)))
		i += copy(data[i:], m.InternalID)
	}
	return i, nil
}

func (m *RescheduleOptions) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *RescheduleOptions) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.InternalID) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintApi(data, i, uint64(len(m.InternalID)))
		i += copy(data[i:], m.InternalID)
	}
	if len(m.Dependencies) > 0 {
		for _, s := range m.Dependencies {
			data[i] = 0x12
			i++
			l = len(s)
			for l >= 1<<7 {
				data[i] = uint8(uint64(l)&0x7f | 0x80)
				l >>= 7
				i++
			}
			data[i] = uint8(l)
			i++
			i += copy(data[i:], s)
		}
	}
	if m.WaitUntil != 0 {
		data[i] = 0x18
		i++
		i = encodeVarintApi(data, i, uint64(m.WaitUntil))
	}
	return i, nil
}

func encodeFixed64Api(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Api(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintApi(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (m *Task) Size() (n int) {
	var l int
	_ = l
	l = len(m.InternalID)
	if l > 0 {
		n += 1 + l + sovApi(uint64(l))
	}
	return n
}

func (m *CreateOptions) Size() (n int) {
	var l int
	_ = l
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovApi(uint64(l))
	}
	if len(m.Dependencies) > 0 {
		for _, s := range m.Dependencies {
			l = len(s)
			n += 1 + l + sovApi(uint64(l))
		}
	}
	if m.WaitUntil != 0 {
		n += 1 + sovApi(uint64(m.WaitUntil))
	}
	if m.Payload != nil {
		l = len(m.Payload)
		if l > 0 {
			n += 1 + l + sovApi(uint64(l))
		}
	}
	l = len(m.Token)
	if l > 0 {
		n += 1 + l + sovApi(uint64(l))
	}
	l = len(m.WorkerQueue)
	if l > 0 {
		n += 1 + l + sovApi(uint64(l))
	}
	return n
}

func (m *CompleteOptions) Size() (n int) {
	var l int
	_ = l
	l = len(m.InternalID)
	if l > 0 {
		n += 1 + l + sovApi(uint64(l))
	}
	return n
}

func (m *RescheduleOptions) Size() (n int) {
	var l int
	_ = l
	l = len(m.InternalID)
	if l > 0 {
		n += 1 + l + sovApi(uint64(l))
	}
	if len(m.Dependencies) > 0 {
		for _, s := range m.Dependencies {
			l = len(s)
			n += 1 + l + sovApi(uint64(l))
		}
	}
	if m.WaitUntil != 0 {
		n += 1 + sovApi(uint64(m.WaitUntil))
	}
	return n
}

func sovApi(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozApi(x uint64) (n int) {
	return sovApi(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}

var _Tasks_gatewayDesc = github_com_limbo_services_core_runtime_limbo.GatewayDesc{
	ServiceName: "taskr.api.Tasks",
	HandlerType: ((*TasksServer)(nil)),
	Routes: []github_com_limbo_services_core_runtime_limbo.RouteDesc{
		{
			Method:  "POST",
			Pattern: "/v1/tasks",
			Handler: _http_Tasks_Create,
		},
		{
			Method:  "POST",
			Pattern: "/v1/tasks/{internal_id}/complete",
			Handler: _http_Tasks_Complete,
		},
		{
			Method:  "POST",
			Pattern: "/v1/tasks/{internal_id}/reschedule",
			Handler: _http_Tasks_Reschedule,
		},
	},
}

func _http_Tasks_Create(srvDesc *google_golang_org_grpc.ServiceDesc, srv interface{}, ctx golang_org_x_net_context.Context, rw net_http.ResponseWriter, req *net_http.Request) error {
	if req.Method != "POST" {
		return github_com_juju_errors.MethodNotAllowedf("expected POST request")
	}

	stream, err := github_com_limbo_services_core_runtime_limbo.NewServerStream(ctx, rw, req, false, false, 0, func(x interface{}) error {
		input := x.(*CreateOptions)
		_ = input

		return nil
	})

	desc := &srvDesc.Methods[0]
	output, err := desc.Handler(srv, stream.Context(), stream.RecvMsg)
	if err == nil && output == nil {
		err = google_golang_org_grpc.Errorf(google_golang_org_grpc_codes.Internal, "internal server error")
	}
	if err == nil {
		err = stream.SendMsg(output)
	}
	if err != nil {
		stream.SetError(err)
	}

	return stream.CloseSend()
}

func _http_Tasks_Complete(srvDesc *google_golang_org_grpc.ServiceDesc, srv interface{}, ctx golang_org_x_net_context.Context, rw net_http.ResponseWriter, req *net_http.Request) error {
	if req.Method != "POST" {
		return github_com_juju_errors.MethodNotAllowedf("expected POST request")
	}

	params := github_com_limbo_services_core_runtime_router.P(ctx)
	stream, err := github_com_limbo_services_core_runtime_limbo.NewServerStream(ctx, rw, req, false, false, 0, func(x interface{}) error {
		input := x.(*CompleteOptions)
		_ = input

		// populate {internal_id}
		{
			var msg0 = input
			val := params.Get("internal_id")
			msg0.InternalID = val
		}

		return nil
	})

	desc := &srvDesc.Methods[1]
	output, err := desc.Handler(srv, stream.Context(), stream.RecvMsg)
	if err == nil && output == nil {
		err = google_golang_org_grpc.Errorf(google_golang_org_grpc_codes.Internal, "internal server error")
	}
	if err == nil {
		err = stream.SendMsg(output)
	}
	if err != nil {
		stream.SetError(err)
	}

	return stream.CloseSend()
}

func _http_Tasks_Reschedule(srvDesc *google_golang_org_grpc.ServiceDesc, srv interface{}, ctx golang_org_x_net_context.Context, rw net_http.ResponseWriter, req *net_http.Request) error {
	if req.Method != "POST" {
		return github_com_juju_errors.MethodNotAllowedf("expected POST request")
	}

	params := github_com_limbo_services_core_runtime_router.P(ctx)
	stream, err := github_com_limbo_services_core_runtime_limbo.NewServerStream(ctx, rw, req, false, false, 0, func(x interface{}) error {
		input := x.(*RescheduleOptions)
		_ = input

		// populate {internal_id}
		{
			var msg0 = input
			val := params.Get("internal_id")
			msg0.InternalID = val
		}

		return nil
	})

	desc := &srvDesc.Methods[2]
	output, err := desc.Handler(srv, stream.Context(), stream.RecvMsg)
	if err == nil && output == nil {
		err = google_golang_org_grpc.Errorf(google_golang_org_grpc_codes.Internal, "internal server error")
	}
	if err == nil {
		err = stream.SendMsg(output)
	}
	if err != nil {
		stream.SetError(err)
	}

	return stream.CloseSend()
}

func (m *Task) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowApi
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Task: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Task: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field InternalID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.InternalID = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipApi(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthApi
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *CreateOptions) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowApi
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: CreateOptions: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CreateOptions: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ID = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Dependencies", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Dependencies = append(m.Dependencies, string(data[iNdEx:postIndex]))
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field WaitUntil", wireType)
			}
			m.WaitUntil = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.WaitUntil |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Payload", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthApi
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Payload = append(m.Payload[:0], data[iNdEx:postIndex]...)
			if m.Payload == nil {
				m.Payload = []byte{}
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Token", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Token = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field WorkerQueue", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.WorkerQueue = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipApi(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthApi
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *CompleteOptions) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowApi
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: CompleteOptions: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CompleteOptions: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field InternalID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.InternalID = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipApi(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthApi
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *RescheduleOptions) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowApi
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RescheduleOptions: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RescheduleOptions: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field InternalID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.InternalID = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Dependencies", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Dependencies = append(m.Dependencies, string(data[iNdEx:postIndex]))
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field WaitUntil", wireType)
			}
			m.WaitUntil = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.WaitUntil |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipApi(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthApi
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipApi(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowApi
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowApi
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if data[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowApi
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthApi
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowApi
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := data[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipApi(data[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthApi = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowApi   = fmt.Errorf("proto: integer overflow")
)

func (msg *Task) Validate() error {
	return nil
}

func (msg *CreateOptions) Validate() error {
	return nil
}

func (msg *CompleteOptions) Validate() error {
	return nil
}

func (msg *RescheduleOptions) Validate() error {
	return nil
}

func init() {
	proto.RegisterType((*Task)(nil), "taskr.api.Task")
	proto.RegisterType((*CreateOptions)(nil), "taskr.api.CreateOptions")
	proto.RegisterType((*CompleteOptions)(nil), "taskr.api.CompleteOptions")
	proto.RegisterType((*RescheduleOptions)(nil), "taskr.api.RescheduleOptions")
	github_com_limbo_services_core_runtime_limbo.RegisterSchemaDefinitions(jsonSchemaDefscd21e75484a2275e5e2503bf125ee570df74b477)
	github_com_limbo_services_core_runtime_limbo.RegisterSwaggerOperations(swaggerDefscd21e75484a2275e5e2503bf125ee570df74b477)
	github_com_limbo_services_core_runtime_limbo.RegisterGatewayDesc(&_Tasks_gatewayDesc)
}

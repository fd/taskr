package gogogrpc

import "google.golang.org/grpc"

// CloneServiceDesc make a new copy of a ServiceDesc
func CloneServiceDesc(desc *grpc.ServiceDesc) *grpc.ServiceDesc {
	clone := &grpc.ServiceDesc{}
	*clone = *desc

	if desc.Methods != nil {
		clone.Methods = make([]grpc.MethodDesc, len(clone.Methods))
		copy(clone.Methods, desc.Methods)
	}

	if desc.Streams != nil {
		clone.Streams = make([]grpc.StreamDesc, len(clone.Streams))
		copy(clone.Streams, desc.Streams)
	}

	return clone
}

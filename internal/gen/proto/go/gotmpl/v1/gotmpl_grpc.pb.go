// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: gotmpl/v1/gotmpl.proto

package gotmplv1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// GotmplServiceClient is the client API for GotmplService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GotmplServiceClient interface {
	// Execute executes a go template.
	Execute(ctx context.Context, in *ExecuteRequest, opts ...grpc.CallOption) (*ExecuteResponse, error)
}

type gotmplServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewGotmplServiceClient(cc grpc.ClientConnInterface) GotmplServiceClient {
	return &gotmplServiceClient{cc}
}

func (c *gotmplServiceClient) Execute(ctx context.Context, in *ExecuteRequest, opts ...grpc.CallOption) (*ExecuteResponse, error) {
	out := new(ExecuteResponse)
	err := c.cc.Invoke(ctx, "/gotmpl.v1.GotmplService/Execute", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GotmplServiceServer is the server API for GotmplService service.
// All implementations must embed UnimplementedGotmplServiceServer
// for forward compatibility
type GotmplServiceServer interface {
	// Execute executes a go template.
	Execute(context.Context, *ExecuteRequest) (*ExecuteResponse, error)
	mustEmbedUnimplementedGotmplServiceServer()
}

// UnimplementedGotmplServiceServer must be embedded to have forward compatible implementations.
type UnimplementedGotmplServiceServer struct {
}

func (UnimplementedGotmplServiceServer) Execute(context.Context, *ExecuteRequest) (*ExecuteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Execute not implemented")
}
func (UnimplementedGotmplServiceServer) mustEmbedUnimplementedGotmplServiceServer() {}

// UnsafeGotmplServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GotmplServiceServer will
// result in compilation errors.
type UnsafeGotmplServiceServer interface {
	mustEmbedUnimplementedGotmplServiceServer()
}

func RegisterGotmplServiceServer(s grpc.ServiceRegistrar, srv GotmplServiceServer) {
	s.RegisterService(&GotmplService_ServiceDesc, srv)
}

func _GotmplService_Execute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ExecuteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GotmplServiceServer).Execute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gotmpl.v1.GotmplService/Execute",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GotmplServiceServer).Execute(ctx, req.(*ExecuteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// GotmplService_ServiceDesc is the grpc.ServiceDesc for GotmplService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var GotmplService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gotmpl.v1.GotmplService",
	HandlerType: (*GotmplServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Execute",
			Handler:    _GotmplService_Execute_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "gotmpl/v1/gotmpl.proto",
}

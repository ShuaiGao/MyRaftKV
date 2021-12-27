// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.0
// source: node/node.proto

package rpc

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

// ElectionServiceClient is the client API for ElectionService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ElectionServiceClient interface {
	Election(ctx context.Context, in *ElectionRequest, opts ...grpc.CallOption) (*ElectionReply, error)
	Heart(ctx context.Context, in *HeartRequest, opts ...grpc.CallOption) (*HeartReply, error)
	Append(ctx context.Context, in *AppendEntryRequest, opts ...grpc.CallOption) (*AppendEntryReply, error)
}

type electionServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewElectionServiceClient(cc grpc.ClientConnInterface) ElectionServiceClient {
	return &electionServiceClient{cc}
}

func (c *electionServiceClient) Election(ctx context.Context, in *ElectionRequest, opts ...grpc.CallOption) (*ElectionReply, error) {
	out := new(ElectionReply)
	err := c.cc.Invoke(ctx, "/service.electionService/Election", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *electionServiceClient) Heart(ctx context.Context, in *HeartRequest, opts ...grpc.CallOption) (*HeartReply, error) {
	out := new(HeartReply)
	err := c.cc.Invoke(ctx, "/service.electionService/Heart", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *electionServiceClient) Append(ctx context.Context, in *AppendEntryRequest, opts ...grpc.CallOption) (*AppendEntryReply, error) {
	out := new(AppendEntryReply)
	err := c.cc.Invoke(ctx, "/service.electionService/Append", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ElectionServiceServer is the server API for ElectionService service.
// All implementations must embed UnimplementedElectionServiceServer
// for forward compatibility
type ElectionServiceServer interface {
	Election(context.Context, *ElectionRequest) (*ElectionReply, error)
	Heart(context.Context, *HeartRequest) (*HeartReply, error)
	Append(context.Context, *AppendEntryRequest) (*AppendEntryReply, error)
	mustEmbedUnimplementedElectionServiceServer()
}

// UnimplementedElectionServiceServer must be embedded to have forward compatible implementations.
type UnimplementedElectionServiceServer struct {
}

func (UnimplementedElectionServiceServer) Election(context.Context, *ElectionRequest) (*ElectionReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Election not implemented")
}
func (UnimplementedElectionServiceServer) Heart(context.Context, *HeartRequest) (*HeartReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Heart not implemented")
}
func (UnimplementedElectionServiceServer) Append(context.Context, *AppendEntryRequest) (*AppendEntryReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Append not implemented")
}
func (UnimplementedElectionServiceServer) mustEmbedUnimplementedElectionServiceServer() {}

// UnsafeElectionServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ElectionServiceServer will
// result in compilation errors.
type UnsafeElectionServiceServer interface {
	mustEmbedUnimplementedElectionServiceServer()
}

func RegisterElectionServiceServer(s grpc.ServiceRegistrar, srv ElectionServiceServer) {
	s.RegisterService(&ElectionService_ServiceDesc, srv)
}

func _ElectionService_Election_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ElectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ElectionServiceServer).Election(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.electionService/Election",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ElectionServiceServer).Election(ctx, req.(*ElectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ElectionService_Heart_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HeartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ElectionServiceServer).Heart(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.electionService/Heart",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ElectionServiceServer).Heart(ctx, req.(*HeartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ElectionService_Append_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AppendEntryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ElectionServiceServer).Append(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.electionService/Append",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ElectionServiceServer).Append(ctx, req.(*AppendEntryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ElectionService_ServiceDesc is the grpc.ServiceDesc for ElectionService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ElectionService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "service.electionService",
	HandlerType: (*ElectionServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Election",
			Handler:    _ElectionService_Election_Handler,
		},
		{
			MethodName: "Heart",
			Handler:    _ElectionService_Heart_Handler,
		},
		{
			MethodName: "Append",
			Handler:    _ElectionService_Append_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "node/node.proto",
}

// DBServiceClient is the client API for DBService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DBServiceClient interface {
	DBAppend(ctx context.Context, in *DBAppendLogRequest, opts ...grpc.CallOption) (*DBAppendLogReply, error)
}

type dBServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDBServiceClient(cc grpc.ClientConnInterface) DBServiceClient {
	return &dBServiceClient{cc}
}

func (c *dBServiceClient) DBAppend(ctx context.Context, in *DBAppendLogRequest, opts ...grpc.CallOption) (*DBAppendLogReply, error) {
	out := new(DBAppendLogReply)
	err := c.cc.Invoke(ctx, "/service.DBService/DBAppend", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DBServiceServer is the server API for DBService service.
// All implementations must embed UnimplementedDBServiceServer
// for forward compatibility
type DBServiceServer interface {
	DBAppend(context.Context, *DBAppendLogRequest) (*DBAppendLogReply, error)
	mustEmbedUnimplementedDBServiceServer()
}

// UnimplementedDBServiceServer must be embedded to have forward compatible implementations.
type UnimplementedDBServiceServer struct {
}

func (UnimplementedDBServiceServer) DBAppend(context.Context, *DBAppendLogRequest) (*DBAppendLogReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DBAppend not implemented")
}
func (UnimplementedDBServiceServer) mustEmbedUnimplementedDBServiceServer() {}

// UnsafeDBServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DBServiceServer will
// result in compilation errors.
type UnsafeDBServiceServer interface {
	mustEmbedUnimplementedDBServiceServer()
}

func RegisterDBServiceServer(s grpc.ServiceRegistrar, srv DBServiceServer) {
	s.RegisterService(&DBService_ServiceDesc, srv)
}

func _DBService_DBAppend_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DBAppendLogRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DBServiceServer).DBAppend(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.DBService/DBAppend",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DBServiceServer).DBAppend(ctx, req.(*DBAppendLogRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DBService_ServiceDesc is the grpc.ServiceDesc for DBService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DBService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "service.DBService",
	HandlerType: (*DBServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "DBAppend",
			Handler:    _DBService_DBAppend_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "node/node.proto",
}
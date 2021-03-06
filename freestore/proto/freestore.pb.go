// Code generated by protoc-gen-go.
// source: freestore.proto
// DO NOT EDIT!

/*
Package freestore is a generated protocol buffer package.

It is generated from these files:
	freestore.proto

It has these top-level messages:
	Peer
	KnownPeersRequest
	KnownPeersResponse
	MakeContractRequest
	MakeContractResponse
	ExtendContractRequest
	ExtendContractResponse
	ContractMetadataRequest
	ContractMetadataResponse
	ReadSectorRequest
	ReadSectorResponse
	WriteSectorRequest
	WriteSectorResponse
	ShrinkRequest
	ShrinkResponse
	ReorderRequest
	ReorderResponse
*/
package freestore

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Peer struct {
	Pubkey  []byte `protobuf:"bytes,1,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	Address string `protobuf:"bytes,2,opt,name=address" json:"address,omitempty"`
}

func (m *Peer) Reset()                    { *m = Peer{} }
func (m *Peer) String() string            { return proto.CompactTextString(m) }
func (*Peer) ProtoMessage()               {}
func (*Peer) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Peer) GetPubkey() []byte {
	if m != nil {
		return m.Pubkey
	}
	return nil
}

func (m *Peer) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

type KnownPeersRequest struct {
	Peers []*Peer `protobuf:"bytes,1,rep,name=peers" json:"peers,omitempty"`
}

func (m *KnownPeersRequest) Reset()                    { *m = KnownPeersRequest{} }
func (m *KnownPeersRequest) String() string            { return proto.CompactTextString(m) }
func (*KnownPeersRequest) ProtoMessage()               {}
func (*KnownPeersRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *KnownPeersRequest) GetPeers() []*Peer {
	if m != nil {
		return m.Peers
	}
	return nil
}

type KnownPeersResponse struct {
	Peers []*Peer `protobuf:"bytes,1,rep,name=peers" json:"peers,omitempty"`
}

func (m *KnownPeersResponse) Reset()                    { *m = KnownPeersResponse{} }
func (m *KnownPeersResponse) String() string            { return proto.CompactTextString(m) }
func (*KnownPeersResponse) ProtoMessage()               {}
func (*KnownPeersResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *KnownPeersResponse) GetPeers() []*Peer {
	if m != nil {
		return m.Peers
	}
	return nil
}

type MakeContractRequest struct {
	SectorSize   int32  `protobuf:"varint,1,opt,name=sector_size,json=sectorSize" json:"sector_size,omitempty"`
	ClientPubkey []byte `protobuf:"bytes,2,opt,name=client_pubkey,json=clientPubkey,proto3" json:"client_pubkey,omitempty"`
	DaysNum      int32  `protobuf:"varint,3,opt,name=days_num,json=daysNum" json:"days_num,omitempty"`
}

func (m *MakeContractRequest) Reset()                    { *m = MakeContractRequest{} }
func (m *MakeContractRequest) String() string            { return proto.CompactTextString(m) }
func (*MakeContractRequest) ProtoMessage()               {}
func (*MakeContractRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *MakeContractRequest) GetSectorSize() int32 {
	if m != nil {
		return m.SectorSize
	}
	return 0
}

func (m *MakeContractRequest) GetClientPubkey() []byte {
	if m != nil {
		return m.ClientPubkey
	}
	return nil
}

func (m *MakeContractRequest) GetDaysNum() int32 {
	if m != nil {
		return m.DaysNum
	}
	return 0
}

type MakeContractResponse struct {
	Id []byte `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (m *MakeContractResponse) Reset()                    { *m = MakeContractResponse{} }
func (m *MakeContractResponse) String() string            { return proto.CompactTextString(m) }
func (*MakeContractResponse) ProtoMessage()               {}
func (*MakeContractResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *MakeContractResponse) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

type ExtendContractRequest struct {
	Id      []byte `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	DaysNum int32  `protobuf:"varint,2,opt,name=days_num,json=daysNum" json:"days_num,omitempty"`
}

func (m *ExtendContractRequest) Reset()                    { *m = ExtendContractRequest{} }
func (m *ExtendContractRequest) String() string            { return proto.CompactTextString(m) }
func (*ExtendContractRequest) ProtoMessage()               {}
func (*ExtendContractRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *ExtendContractRequest) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *ExtendContractRequest) GetDaysNum() int32 {
	if m != nil {
		return m.DaysNum
	}
	return 0
}

type ExtendContractResponse struct {
}

func (m *ExtendContractResponse) Reset()                    { *m = ExtendContractResponse{} }
func (m *ExtendContractResponse) String() string            { return proto.CompactTextString(m) }
func (*ExtendContractResponse) ProtoMessage()               {}
func (*ExtendContractResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

type ContractMetadataRequest struct {
	Id []byte `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (m *ContractMetadataRequest) Reset()                    { *m = ContractMetadataRequest{} }
func (m *ContractMetadataRequest) String() string            { return proto.CompactTextString(m) }
func (*ContractMetadataRequest) ProtoMessage()               {}
func (*ContractMetadataRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *ContractMetadataRequest) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

type ContractMetadataResponse struct {
	SectorIds  [][]byte `protobuf:"bytes,1,rep,name=sector_ids,json=sectorIds,proto3" json:"sector_ids,omitempty"`
	DaysLeft   int32    `protobuf:"varint,2,opt,name=days_left,json=daysLeft" json:"days_left,omitempty"`
	SectorSize int32    `protobuf:"varint,3,opt,name=sector_size,json=sectorSize" json:"sector_size,omitempty"`
	Signature  []byte   `protobuf:"bytes,4,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (m *ContractMetadataResponse) Reset()                    { *m = ContractMetadataResponse{} }
func (m *ContractMetadataResponse) String() string            { return proto.CompactTextString(m) }
func (*ContractMetadataResponse) ProtoMessage()               {}
func (*ContractMetadataResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

func (m *ContractMetadataResponse) GetSectorIds() [][]byte {
	if m != nil {
		return m.SectorIds
	}
	return nil
}

func (m *ContractMetadataResponse) GetDaysLeft() int32 {
	if m != nil {
		return m.DaysLeft
	}
	return 0
}

func (m *ContractMetadataResponse) GetSectorSize() int32 {
	if m != nil {
		return m.SectorSize
	}
	return 0
}

func (m *ContractMetadataResponse) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type ReadSectorRequest struct {
	Id          []byte `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Sector      int64  `protobuf:"varint,2,opt,name=sector" json:"sector,omitempty"`
	Offset      int32  `protobuf:"varint,3,opt,name=offset" json:"offset,omitempty"`
	Size        int32  `protobuf:"varint,4,opt,name=size" json:"size,omitempty"`
	ProofNeeded bool   `protobuf:"varint,5,opt,name=proof_needed,json=proofNeeded" json:"proof_needed,omitempty"`
}

func (m *ReadSectorRequest) Reset()                    { *m = ReadSectorRequest{} }
func (m *ReadSectorRequest) String() string            { return proto.CompactTextString(m) }
func (*ReadSectorRequest) ProtoMessage()               {}
func (*ReadSectorRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

func (m *ReadSectorRequest) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *ReadSectorRequest) GetSector() int64 {
	if m != nil {
		return m.Sector
	}
	return 0
}

func (m *ReadSectorRequest) GetOffset() int32 {
	if m != nil {
		return m.Offset
	}
	return 0
}

func (m *ReadSectorRequest) GetSize() int32 {
	if m != nil {
		return m.Size
	}
	return 0
}

func (m *ReadSectorRequest) GetProofNeeded() bool {
	if m != nil {
		return m.ProofNeeded
	}
	return false
}

type ReadSectorResponse struct {
	Data  []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Proof []byte `protobuf:"bytes,2,opt,name=proof,proto3" json:"proof,omitempty"`
}

func (m *ReadSectorResponse) Reset()                    { *m = ReadSectorResponse{} }
func (m *ReadSectorResponse) String() string            { return proto.CompactTextString(m) }
func (*ReadSectorResponse) ProtoMessage()               {}
func (*ReadSectorResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{10} }

func (m *ReadSectorResponse) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *ReadSectorResponse) GetProof() []byte {
	if m != nil {
		return m.Proof
	}
	return nil
}

type WriteSectorRequest struct {
	Id        []byte `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Sector    int64  `protobuf:"varint,2,opt,name=sector" json:"sector,omitempty"`
	Offset    int32  `protobuf:"varint,3,opt,name=offset" json:"offset,omitempty"`
	Length    int32  `protobuf:"varint,4,opt,name=length" json:"length,omitempty"`
	Data      []byte `protobuf:"bytes,5,opt,name=data,proto3" json:"data,omitempty"`
	Signature []byte `protobuf:"bytes,6,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (m *WriteSectorRequest) Reset()                    { *m = WriteSectorRequest{} }
func (m *WriteSectorRequest) String() string            { return proto.CompactTextString(m) }
func (*WriteSectorRequest) ProtoMessage()               {}
func (*WriteSectorRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{11} }

func (m *WriteSectorRequest) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *WriteSectorRequest) GetSector() int64 {
	if m != nil {
		return m.Sector
	}
	return 0
}

func (m *WriteSectorRequest) GetOffset() int32 {
	if m != nil {
		return m.Offset
	}
	return 0
}

func (m *WriteSectorRequest) GetLength() int32 {
	if m != nil {
		return m.Length
	}
	return 0
}

func (m *WriteSectorRequest) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *WriteSectorRequest) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type WriteSectorResponse struct {
}

func (m *WriteSectorResponse) Reset()                    { *m = WriteSectorResponse{} }
func (m *WriteSectorResponse) String() string            { return proto.CompactTextString(m) }
func (*WriteSectorResponse) ProtoMessage()               {}
func (*WriteSectorResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{12} }

type ShrinkRequest struct {
	Id         []byte `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	NumSectors int64  `protobuf:"varint,2,opt,name=num_sectors,json=numSectors" json:"num_sectors,omitempty"`
	Signature  []byte `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (m *ShrinkRequest) Reset()                    { *m = ShrinkRequest{} }
func (m *ShrinkRequest) String() string            { return proto.CompactTextString(m) }
func (*ShrinkRequest) ProtoMessage()               {}
func (*ShrinkRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{13} }

func (m *ShrinkRequest) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *ShrinkRequest) GetNumSectors() int64 {
	if m != nil {
		return m.NumSectors
	}
	return 0
}

func (m *ShrinkRequest) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type ShrinkResponse struct {
}

func (m *ShrinkResponse) Reset()                    { *m = ShrinkResponse{} }
func (m *ShrinkResponse) String() string            { return proto.CompactTextString(m) }
func (*ShrinkResponse) ProtoMessage()               {}
func (*ShrinkResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{14} }

type ReorderRequest struct {
	Id        []byte  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Ordering  []int64 `protobuf:"varint,2,rep,packed,name=ordering" json:"ordering,omitempty"`
	Signature []byte  `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (m *ReorderRequest) Reset()                    { *m = ReorderRequest{} }
func (m *ReorderRequest) String() string            { return proto.CompactTextString(m) }
func (*ReorderRequest) ProtoMessage()               {}
func (*ReorderRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{15} }

func (m *ReorderRequest) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *ReorderRequest) GetOrdering() []int64 {
	if m != nil {
		return m.Ordering
	}
	return nil
}

func (m *ReorderRequest) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type ReorderResponse struct {
}

func (m *ReorderResponse) Reset()                    { *m = ReorderResponse{} }
func (m *ReorderResponse) String() string            { return proto.CompactTextString(m) }
func (*ReorderResponse) ProtoMessage()               {}
func (*ReorderResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{16} }

func init() {
	proto.RegisterType((*Peer)(nil), "freestore.Peer")
	proto.RegisterType((*KnownPeersRequest)(nil), "freestore.KnownPeersRequest")
	proto.RegisterType((*KnownPeersResponse)(nil), "freestore.KnownPeersResponse")
	proto.RegisterType((*MakeContractRequest)(nil), "freestore.MakeContractRequest")
	proto.RegisterType((*MakeContractResponse)(nil), "freestore.MakeContractResponse")
	proto.RegisterType((*ExtendContractRequest)(nil), "freestore.ExtendContractRequest")
	proto.RegisterType((*ExtendContractResponse)(nil), "freestore.ExtendContractResponse")
	proto.RegisterType((*ContractMetadataRequest)(nil), "freestore.ContractMetadataRequest")
	proto.RegisterType((*ContractMetadataResponse)(nil), "freestore.ContractMetadataResponse")
	proto.RegisterType((*ReadSectorRequest)(nil), "freestore.ReadSectorRequest")
	proto.RegisterType((*ReadSectorResponse)(nil), "freestore.ReadSectorResponse")
	proto.RegisterType((*WriteSectorRequest)(nil), "freestore.WriteSectorRequest")
	proto.RegisterType((*WriteSectorResponse)(nil), "freestore.WriteSectorResponse")
	proto.RegisterType((*ShrinkRequest)(nil), "freestore.ShrinkRequest")
	proto.RegisterType((*ShrinkResponse)(nil), "freestore.ShrinkResponse")
	proto.RegisterType((*ReorderRequest)(nil), "freestore.ReorderRequest")
	proto.RegisterType((*ReorderResponse)(nil), "freestore.ReorderResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Freestore service

type FreestoreClient interface {
	KnownPeers(ctx context.Context, in *KnownPeersRequest, opts ...grpc.CallOption) (*KnownPeersResponse, error)
	MakeContract(ctx context.Context, in *MakeContractRequest, opts ...grpc.CallOption) (*MakeContractResponse, error)
	ExtendContract(ctx context.Context, in *ExtendContractRequest, opts ...grpc.CallOption) (*ExtendContractResponse, error)
	ContractMetadata(ctx context.Context, in *ContractMetadataRequest, opts ...grpc.CallOption) (*ContractMetadataResponse, error)
	ReadSector(ctx context.Context, in *ReadSectorRequest, opts ...grpc.CallOption) (*ReadSectorResponse, error)
	Shrink(ctx context.Context, in *ShrinkRequest, opts ...grpc.CallOption) (*ShrinkResponse, error)
	Reorder(ctx context.Context, in *ReorderRequest, opts ...grpc.CallOption) (*ReorderResponse, error)
}

type freestoreClient struct {
	cc *grpc.ClientConn
}

func NewFreestoreClient(cc *grpc.ClientConn) FreestoreClient {
	return &freestoreClient{cc}
}

func (c *freestoreClient) KnownPeers(ctx context.Context, in *KnownPeersRequest, opts ...grpc.CallOption) (*KnownPeersResponse, error) {
	out := new(KnownPeersResponse)
	err := grpc.Invoke(ctx, "/freestore.Freestore/KnownPeers", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *freestoreClient) MakeContract(ctx context.Context, in *MakeContractRequest, opts ...grpc.CallOption) (*MakeContractResponse, error) {
	out := new(MakeContractResponse)
	err := grpc.Invoke(ctx, "/freestore.Freestore/MakeContract", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *freestoreClient) ExtendContract(ctx context.Context, in *ExtendContractRequest, opts ...grpc.CallOption) (*ExtendContractResponse, error) {
	out := new(ExtendContractResponse)
	err := grpc.Invoke(ctx, "/freestore.Freestore/ExtendContract", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *freestoreClient) ContractMetadata(ctx context.Context, in *ContractMetadataRequest, opts ...grpc.CallOption) (*ContractMetadataResponse, error) {
	out := new(ContractMetadataResponse)
	err := grpc.Invoke(ctx, "/freestore.Freestore/ContractMetadata", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *freestoreClient) ReadSector(ctx context.Context, in *ReadSectorRequest, opts ...grpc.CallOption) (*ReadSectorResponse, error) {
	out := new(ReadSectorResponse)
	err := grpc.Invoke(ctx, "/freestore.Freestore/ReadSector", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *freestoreClient) Shrink(ctx context.Context, in *ShrinkRequest, opts ...grpc.CallOption) (*ShrinkResponse, error) {
	out := new(ShrinkResponse)
	err := grpc.Invoke(ctx, "/freestore.Freestore/Shrink", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *freestoreClient) Reorder(ctx context.Context, in *ReorderRequest, opts ...grpc.CallOption) (*ReorderResponse, error) {
	out := new(ReorderResponse)
	err := grpc.Invoke(ctx, "/freestore.Freestore/Reorder", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Freestore service

type FreestoreServer interface {
	KnownPeers(context.Context, *KnownPeersRequest) (*KnownPeersResponse, error)
	MakeContract(context.Context, *MakeContractRequest) (*MakeContractResponse, error)
	ExtendContract(context.Context, *ExtendContractRequest) (*ExtendContractResponse, error)
	ContractMetadata(context.Context, *ContractMetadataRequest) (*ContractMetadataResponse, error)
	ReadSector(context.Context, *ReadSectorRequest) (*ReadSectorResponse, error)
	Shrink(context.Context, *ShrinkRequest) (*ShrinkResponse, error)
	Reorder(context.Context, *ReorderRequest) (*ReorderResponse, error)
}

func RegisterFreestoreServer(s *grpc.Server, srv FreestoreServer) {
	s.RegisterService(&_Freestore_serviceDesc, srv)
}

func _Freestore_KnownPeers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(KnownPeersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FreestoreServer).KnownPeers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/freestore.Freestore/KnownPeers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FreestoreServer).KnownPeers(ctx, req.(*KnownPeersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Freestore_MakeContract_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MakeContractRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FreestoreServer).MakeContract(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/freestore.Freestore/MakeContract",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FreestoreServer).MakeContract(ctx, req.(*MakeContractRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Freestore_ExtendContract_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ExtendContractRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FreestoreServer).ExtendContract(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/freestore.Freestore/ExtendContract",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FreestoreServer).ExtendContract(ctx, req.(*ExtendContractRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Freestore_ContractMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ContractMetadataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FreestoreServer).ContractMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/freestore.Freestore/ContractMetadata",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FreestoreServer).ContractMetadata(ctx, req.(*ContractMetadataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Freestore_ReadSector_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadSectorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FreestoreServer).ReadSector(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/freestore.Freestore/ReadSector",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FreestoreServer).ReadSector(ctx, req.(*ReadSectorRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Freestore_Shrink_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShrinkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FreestoreServer).Shrink(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/freestore.Freestore/Shrink",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FreestoreServer).Shrink(ctx, req.(*ShrinkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Freestore_Reorder_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReorderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FreestoreServer).Reorder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/freestore.Freestore/Reorder",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FreestoreServer).Reorder(ctx, req.(*ReorderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Freestore_serviceDesc = grpc.ServiceDesc{
	ServiceName: "freestore.Freestore",
	HandlerType: (*FreestoreServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "KnownPeers",
			Handler:    _Freestore_KnownPeers_Handler,
		},
		{
			MethodName: "MakeContract",
			Handler:    _Freestore_MakeContract_Handler,
		},
		{
			MethodName: "ExtendContract",
			Handler:    _Freestore_ExtendContract_Handler,
		},
		{
			MethodName: "ContractMetadata",
			Handler:    _Freestore_ContractMetadata_Handler,
		},
		{
			MethodName: "ReadSector",
			Handler:    _Freestore_ReadSector_Handler,
		},
		{
			MethodName: "Shrink",
			Handler:    _Freestore_Shrink_Handler,
		},
		{
			MethodName: "Reorder",
			Handler:    _Freestore_Reorder_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "freestore.proto",
}

func init() { proto.RegisterFile("freestore.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 676 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x55, 0x4f, 0x6f, 0xd3, 0x4e,
	0x10, 0x95, 0xe3, 0x24, 0x6d, 0x26, 0x69, 0xda, 0x6e, 0xdb, 0xfc, 0x5c, 0xff, 0x5a, 0x9a, 0xba,
	0x02, 0x85, 0x4b, 0x0f, 0xe5, 0x82, 0x40, 0x20, 0x04, 0x02, 0xa9, 0x82, 0x96, 0xca, 0x15, 0x42,
	0x02, 0x89, 0xc8, 0xed, 0x4e, 0x5a, 0xab, 0xc9, 0x3a, 0x78, 0xd7, 0x40, 0xfb, 0x11, 0xb8, 0x73,
	0xe5, 0xca, 0xd7, 0x44, 0xd9, 0x9d, 0x24, 0xb6, 0x13, 0x17, 0x0e, 0xdc, 0x3c, 0x7f, 0xdf, 0x9b,
	0x97, 0xd9, 0x09, 0x2c, 0xf7, 0x62, 0x44, 0xa9, 0xa2, 0x18, 0xf7, 0x87, 0x71, 0xa4, 0x22, 0x56,
	0x9b, 0x38, 0xbc, 0x87, 0x50, 0x3e, 0x41, 0x8c, 0x59, 0x0b, 0xaa, 0xc3, 0xe4, 0xec, 0x0a, 0xaf,
	0x1d, 0xab, 0x6d, 0x75, 0x1a, 0x3e, 0x59, 0xcc, 0x81, 0x85, 0x80, 0xf3, 0x18, 0xa5, 0x74, 0x4a,
	0x6d, 0xab, 0x53, 0xf3, 0xc7, 0xa6, 0xf7, 0x08, 0x56, 0x5f, 0x8b, 0xe8, 0xab, 0x18, 0x95, 0x4b,
	0x1f, 0x3f, 0x27, 0x28, 0x15, 0xbb, 0x0b, 0x95, 0xe1, 0xc8, 0x76, 0xac, 0xb6, 0xdd, 0xa9, 0x1f,
	0x2c, 0xef, 0x4f, 0xa1, 0x47, 0x79, 0xbe, 0x89, 0x7a, 0x8f, 0x81, 0xa5, 0x6b, 0xe5, 0x30, 0x12,
	0x12, 0xff, 0xb6, 0xf8, 0x0b, 0xac, 0x1d, 0x05, 0x57, 0xf8, 0x22, 0x12, 0x2a, 0x0e, 0xce, 0xd5,
	0x18, 0x7a, 0x07, 0xea, 0x12, 0xcf, 0x55, 0x14, 0x77, 0x65, 0x78, 0x83, 0x7a, 0x8c, 0x8a, 0x0f,
	0xc6, 0x75, 0x1a, 0xde, 0x20, 0xdb, 0x83, 0xa5, 0xf3, 0x7e, 0x88, 0x42, 0x75, 0x69, 0xd2, 0x92,
	0x9e, 0xb4, 0x61, 0x9c, 0x27, 0x66, 0xde, 0x4d, 0x58, 0xe4, 0xc1, 0xb5, 0xec, 0x8a, 0x64, 0xe0,
	0xd8, 0xba, 0xc5, 0xc2, 0xc8, 0x3e, 0x4e, 0x06, 0xde, 0x3d, 0x58, 0xcf, 0xe2, 0x12, 0xed, 0x26,
	0x94, 0x42, 0x4e, 0xb2, 0x95, 0x42, 0xee, 0x3d, 0x87, 0x8d, 0x97, 0xdf, 0x14, 0x0a, 0x9e, 0x67,
	0x98, 0x4b, 0xcc, 0x60, 0x95, 0xb2, 0x58, 0x0e, 0xb4, 0xf2, 0x3d, 0x0c, 0x9a, 0x77, 0x1f, 0xfe,
	0x1b, 0xfb, 0x8e, 0x50, 0x05, 0x3c, 0x50, 0x41, 0x41, 0x7f, 0xef, 0x87, 0x05, 0xce, 0x6c, 0x2e,
	0xb1, 0xde, 0x06, 0xd2, 0xa6, 0x1b, 0x72, 0xa3, 0x78, 0xc3, 0xaf, 0x19, 0xcf, 0x21, 0x97, 0xec,
	0x7f, 0xa8, 0x69, 0x6e, 0x7d, 0xec, 0x29, 0x22, 0xa7, 0xc9, 0xbe, 0xc1, 0xde, 0x8c, 0xd4, 0xf6,
	0x8c, 0xd4, 0x5b, 0x50, 0x93, 0xe1, 0x85, 0x08, 0x54, 0x12, 0xa3, 0x53, 0xd6, 0x84, 0xa6, 0x0e,
	0xef, 0xbb, 0x05, 0xab, 0x3e, 0x06, 0xfc, 0x54, 0x17, 0x14, 0xa9, 0xd3, 0x82, 0xaa, 0xe9, 0xa8,
	0xe1, 0x6d, 0x9f, 0xac, 0x91, 0x3f, 0xea, 0xf5, 0x24, 0x2a, 0xc2, 0x25, 0x8b, 0x31, 0x28, 0x6b,
	0x36, 0x65, 0xed, 0xd5, 0xdf, 0x6c, 0x17, 0x1a, 0xc3, 0x38, 0x8a, 0x7a, 0x5d, 0x81, 0xc8, 0x91,
	0x3b, 0x95, 0xb6, 0xd5, 0x59, 0xf4, 0xeb, 0xda, 0x77, 0xac, 0x5d, 0xde, 0x53, 0x60, 0x69, 0x2e,
	0xa4, 0x0e, 0x83, 0xf2, 0x48, 0x2d, 0xa2, 0xa3, 0xbf, 0xd9, 0x3a, 0x54, 0x74, 0x21, 0xed, 0x8d,
	0x31, 0xbc, 0x9f, 0x16, 0xb0, 0xf7, 0x71, 0xa8, 0xf0, 0xdf, 0x4e, 0xd3, 0x82, 0x6a, 0x1f, 0xc5,
	0x85, 0xba, 0xa4, 0x79, 0xc8, 0x9a, 0x10, 0xab, 0xa4, 0x88, 0x65, 0xd4, 0xae, 0xe6, 0xd5, 0xde,
	0x80, 0xb5, 0x0c, 0x3f, 0xda, 0xa3, 0x4f, 0xb0, 0x74, 0x7a, 0x19, 0x87, 0xe2, 0xaa, 0x88, 0xf1,
	0x0e, 0xd4, 0x45, 0x32, 0xe8, 0x1a, 0x9e, 0x92, 0x68, 0x83, 0x48, 0x06, 0xa6, 0x91, 0xcc, 0xc2,
	0xda, 0x79, 0xd8, 0x15, 0x68, 0x8e, 0xfb, 0x13, 0xe2, 0x07, 0x68, 0xfa, 0x18, 0xc5, 0x1c, 0x0b,
	0x45, 0x72, 0x61, 0x51, 0xc7, 0x43, 0x71, 0xe1, 0x94, 0xda, 0x76, 0xc7, 0xf6, 0x27, 0xf6, 0x1f,
	0xd0, 0x56, 0x61, 0x79, 0xd2, 0xdb, 0xc0, 0x1d, 0xfc, 0x2a, 0x43, 0xed, 0xd5, 0xf8, 0x80, 0xb0,
	0x43, 0x80, 0xe9, 0xc5, 0x61, 0x5b, 0xa9, 0xd3, 0x32, 0x73, 0xc4, 0xdc, 0xed, 0x82, 0x28, 0xed,
	0xc6, 0x5b, 0x68, 0xa4, 0xef, 0x00, 0xbb, 0x93, 0x4a, 0x9f, 0x73, 0x98, 0xdc, 0x9d, 0xc2, 0x38,
	0x35, 0x7c, 0x07, 0xcd, 0xec, 0x63, 0x67, 0xed, 0x54, 0xc9, 0xdc, 0x5b, 0xe2, 0xee, 0xde, 0x92,
	0x41, 0x6d, 0x3f, 0xc2, 0x4a, 0xfe, 0xf5, 0x33, 0x2f, 0x55, 0x56, 0x70, 0x46, 0xdc, 0xbd, 0x5b,
	0x73, 0xa8, 0xf9, 0x21, 0xc0, 0xf4, 0xd9, 0x64, 0xf4, 0x9c, 0x79, 0xd9, 0x19, 0x3d, 0xe7, 0xbc,
	0xb5, 0x27, 0x50, 0x35, 0x9b, 0xc2, 0x9c, 0x54, 0x62, 0x66, 0x39, 0xdd, 0xcd, 0x39, 0x11, 0x2a,
	0x7f, 0x06, 0x0b, 0xf4, 0xd3, 0xb3, 0xcd, 0x0c, 0x50, 0x7a, 0xd5, 0x5c, 0x77, 0x5e, 0xc8, 0x74,
	0x38, 0xab, 0xea, 0x7f, 0xc5, 0x07, 0xbf, 0x03, 0x00, 0x00, 0xff, 0xff, 0xcc, 0x51, 0x9a, 0xe8,
	0x28, 0x07, 0x00, 0x00,
}

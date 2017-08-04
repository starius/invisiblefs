package server

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fpb "github.com/starius/invisiblefs/freestore/proto"
)

type Pubkey struct {
	// TODO
}

type Server struct {
	peers map[string]Pubkey
}

func NewServer() (*Server, error) {
	server := &Server{
		peers: make(map[string]Pubkey),
	}
	return server, nil
}

func (s *Server) KnownPeers(ctx context.Context, req *fpb.KnownPeersRequest) (*fpb.KnownPeersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "todo")
}

func (s *Server) MakeContract(ctx context.Context, req *fpb.MakeContractRequest) (*fpb.MakeContractResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "todo")
}

func (s *Server) ExtendContract(ctx context.Context, req *fpb.ExtendContractRequest) (*fpb.ExtendContractResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "todo")
}

func (s *Server) ContractMetadata(ctx context.Context, req *fpb.ContractMetadataRequest) (*fpb.ContractMetadataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "todo")
}

func (s *Server) ReadSector(ctx context.Context, req *fpb.ReadSectorRequest) (*fpb.ReadSectorResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "todo")
}

func (s *Server) Shrink(ctx context.Context, req *fpb.ShrinkRequest) (*fpb.ShrinkResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "todo")
}

func (s *Server) Reorder(ctx context.Context, req *fpb.ReorderRequest) (*fpb.ReorderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "todo")
}

package main

import (
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"

	fpb "github.com/starius/invisiblefs/freestore/proto"
	"github.com/starius/invisiblefs/freestore/server"
)

var (
	serverListenAddress = flag.String("server-listen-address", "", "Address to run GRPC server on")
)

func main() {
	flag.Parse()
	grpcServer := grpc.NewServer()
	server, err := server.NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v.", err)
	}
	fpb.RegisterFreestoreServer(grpcServer, server)
	conn, err := net.Listen("tcp", *serverListenAddress)
	if err != nil {
		log.Fatalf("net.Listen: %v.", err)
	}
	grpcServer.Serve(conn)
}

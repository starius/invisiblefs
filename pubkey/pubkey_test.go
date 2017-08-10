package pubkey

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
)

type dummyListener struct {
	c chan net.Conn
}

func (l *dummyListener) Accept() (net.Conn, error) {
	conn, ok := <-l.c
	if !ok {
		return nil, fmt.Errorf("l.c is closed")
	}
	return conn, nil
}

func (l *dummyListener) Close() error {
	return nil
}

func (l *dummyListener) Addr() net.Addr {
	return &net.IPAddr{
		IP:   net.IPv4(1, 2, 3, 4),
		Zone: "-",
	}
}

func TestPubkey(t *testing.T) {
	priv, err := GeneratePriv()
	if err != nil {
		t.Fatal(err)
	}
	cert, err := Cert(priv)
	if err != nil {
		t.Fatal(err)
	}
	serverCreds, err := ServerCreds(priv, cert)
	if err != nil {
		t.Fatal(err)
	}
	clientCreds, err := ClientCreds(cert)
	if err != nil {
		t.Fatal(err)
	}
	c := make(chan net.Conn)
	listener := &dummyListener{c: c}
	grpcServer := grpc.NewServer(grpc.Creds(serverCreds))
	var wg sync.WaitGroup
	var serverErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		serverErr = grpcServer.Serve(listener)
	}()
	cConn, sConn := net.Pipe()
	c <- sConn
	dialer := func(string, time.Duration) (net.Conn, error) {
		return cConn, nil
	}
	cCliConn, err := grpc.Dial(
		"server",
		grpc.WithBlock(),
		grpc.WithDialer(dialer),
		grpc.WithTransportCredentials(clientCreds),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := cCliConn.Close(); err != nil {
		t.Fatal(err)
	}
	grpcServer.GracefulStop()
	close(c)
	wg.Wait()
	if serverErr != nil {
		t.Fatal(serverErr)
	}
}

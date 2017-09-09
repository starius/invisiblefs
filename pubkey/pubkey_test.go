package pubkey

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
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

type helloServer struct{}

func (s *helloServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func TestPubkey(t *testing.T) {
	t.Skip("hangs in Travis")
	priv, err := GeneratePriv(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := Cert(priv, rand.Reader)
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
	pb.RegisterGreeterServer(grpcServer, &helloServer{})
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cCliConn, err := grpc.DialContext(
		ctx,
		"server",
		grpc.WithBlock(),
		grpc.WithDialer(dialer),
		grpc.WithTransportCredentials(clientCreds),
	)
	if err != nil {
		t.Fatal(err)
	}
	grpcClient := pb.NewGreeterClient(cCliConn)
	res, err := grpcClient.SayHello(context.Background(), &pb.HelloRequest{
		Name: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Message != "Hello test" {
		t.Errorf("res.Message = %q", res.Message)
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

func hkdfReader(secret string) io.Reader {
	xof, err := blake2b.NewXOF(blake2b.OutputLengthUnknown, []byte(secret))
	if err != nil {
		panic(err)
	}
	return xof
}

func TestPubkeyReproducibleRandom(t *testing.T) {
	t.Skip("hangs in Travis")
	r1 := hkdfReader("top secret")
	priv1, err := GeneratePriv(r1)
	if err != nil {
		t.Fatal(err)
	}
	cert1, err := Cert(priv1, r1)
	if err != nil {
		t.Fatal(err)
	}
	r2 := hkdfReader("top secret")
	priv2, err := GeneratePriv(r2)
	if err != nil {
		t.Fatal(err)
	}
	cert2, err := Cert(priv2, r2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(priv1, priv2) {
		t.Fatal("private keys generated from the same random inputs are different")
	}
	if !bytes.Equal(cert1, cert2) {
		t.Fatal("certs generated from the same PRNG are different")
	}
}

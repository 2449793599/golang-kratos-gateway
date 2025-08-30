package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	pb "github.com/go-kratos/examples/helloworld/helloworld"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

var (
	httpAddr string
	grpcAddr string
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {

	fmt.Println("收到GRPC请求")

	if in.Name == "error" {
		return nil, context.DeadlineExceeded
	}

	return &pb.HelloReply{Message: in.GetName()}, nil

}

func init() {
	flag.StringVar(&httpAddr, "http.addr", ":8000", "server address, eg: 127.0.0.1:8000")
	flag.StringVar(&grpcAddr, "grpc.addr", ":9000", "server address, eg: 127.0.0.1:9000")
}

func main() {

	flag.Parse()

	s := &server{}

	httpSrv := http.NewServer(
		http.Address(httpAddr),
		http.Middleware(
			recovery.Recovery(), // 恢复中间件：捕获PANIC避免服务崩溃
		),
	)
	grpcSrv := grpc.NewServer(
		grpc.Address(grpcAddr),
		grpc.Middleware(
			recovery.Recovery(), // 恢复中间件：捕获PANIC避免服务崩溃
		),
	)

	// curl http://127.0.0.1:8000/helloworld/header
	httpSrv.HandleFunc("/helloworld/header", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("收到HTTP请求")
		for k, v := range r.Header {
			fmt.Fprintf(w, "%s: %s\n", k, v)
		}
	})

	pb.RegisterGreeterServer(grpcSrv, s)
	pb.RegisterGreeterHTTPServer(httpSrv, s)

	app := kratos.New(
		kratos.Name("Helloworld"),
		kratos.Server(
			httpSrv,
			grpcSrv,
		),
	)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

}

package main

import (
	"context"
	"log"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "gateway/proto"
)

func main() {
	ctx := context.Background()
	mux := runtime.NewServeMux()

	// 注册gRPC网关端点
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	err := pb.RegisterUserServiceHandlerFromEndpoint(
		ctx,
		mux,
		"localhost:50052",
		opts,
	)
	if err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	// 启动HTTP服务
	log.Println("gRPC-Gateway running on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

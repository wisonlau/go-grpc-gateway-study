package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "gateway/proto"
)

func loggingInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Record start time
	start := time.Now()

	// Log request details
	log.Printf("gRPC request started | Method: %s | Request: %+v", method, req)

	// Invoke the actual RPC method
	err := invoker(ctx, method, req, reply, cc, opts...)

	// Calculate duration and log results
	duration := time.Since(start)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			log.Printf("gRPC request failed | Method: %s | Duration: %v | Error code: %d | Error message: %s",
				method, duration, s.Code(), s.Message())
		} else {
			log.Printf("gRPC request failed | Method: %s | Duration: %v | Error: %v",
				method, duration, err)
		}
	} else {
		log.Printf("gRPC request succeeded | Method: %s | Duration: %v | Response: %+v",
			method, duration, reply)
	}

	return err
}

func main() {
	ctx := context.Background()
	mux := runtime.NewServeMux()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(loggingInterceptor),
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

	log.Println("gRPC-Gateway running on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

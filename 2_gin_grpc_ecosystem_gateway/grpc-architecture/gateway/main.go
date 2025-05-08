package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "gateway/proto"
)

func newPrefixHandler(gwMux *runtime.ServeMux, prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// remove the specified prefix
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		gwMux.ServeHTTP(w, r)
	})
}

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
	// 1. simple use grpc-ecosystem/grpc-gateway/v2/runtime
	/*
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
	*/

	// 2. use gin + grpc-ecosystem/grpc-gateway/v2/runtime
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

	// Create Gin router
	router := gin.Default()
	// Add health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	// Route /api/* requests to gRPC-Gateway
	apiPrefix := "/api"
	apiGroup := router.Group(apiPrefix)
	apiGroup.Any("/*any", gin.WrapH(newPrefixHandler(mux, apiPrefix)))
	// Route /api/* requests to Gin
	orderGroup := router.Group("/orders")
	{
		orderGroup.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": "order list"})
		})
		orderGroup.GET("/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"order": c.Param("id")})
		})
	}

	// Start server
	log.Println("HTTP server listening on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

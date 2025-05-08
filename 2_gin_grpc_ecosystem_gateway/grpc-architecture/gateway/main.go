package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "gateway/proto"
)

type gatewayServer struct {
	pb.UnimplementedUserServiceServer
	userClient pb.UserServiceClient
}

func (s *gatewayServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	log.Printf("[Gateway] Processing gRPC request GetUser: %v", req)
	return s.userClient.GetUser(ctx, req)
}

func (s *gatewayServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	log.Printf("[Gateway] Processing gRPC request CreateUser: %v", req)
	return s.userClient.CreateUser(ctx, req)
}

func loggingInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	duration := time.Since(start)

	if err != nil {
		if s, ok := status.FromError(err); ok {
			log.Printf("gRPC call failed | Method: %s | Duration: %v | Code: %d | Message: %s",
				method, duration, s.Code(), s.Message())
		} else {
			log.Printf("gRPC call failed | Method: %s | Duration: %v | Error: %v",
				method, duration, err)
		}
	} else {
		log.Printf("gRPC call succeeded | Method: %s | Duration: %v", method, duration)
	}
	return err
}

func startGRPCServer(ctx context.Context, client pb.UserServiceClient) error {
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		return fmt.Errorf("failed to listen on port 8081: %w", err)
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				start := time.Now()
				defer func() {
					log.Printf("gRPC server processing completed | Method: %s | Duration: %v", info.FullMethod, time.Since(start))
				}()
				return handler(ctx, req)
			},
		),
	)
	pb.RegisterUserServiceServer(s, &gatewayServer{userClient: client})

	go func() {
		<-ctx.Done()
		log.Println("Gracefully shutting down gRPC server...")
		s.GracefulStop()
	}()

	log.Println("gRPC server started on :8081")
	return s.Serve(lis)
}

func newPrefixHandler(gwMux *runtime.ServeMux, prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// remove the specified prefix
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		gwMux.ServeHTTP(w, r)
	})
}

func startHTTPServer(ctx context.Context, gwMux *runtime.ServeMux) error {
	router := gin.Default()
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// API routing group
	apiPrefix := "/api"
	apiGroup := router.Group(apiPrefix)
	apiGroup.Any("/*any", gin.WrapH(newPrefixHandler(gwMux, apiPrefix)))

	// orders group
	orderGroup := router.Group("/orders")
	{
		orderGroup.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": "order list"})
		})
		orderGroup.GET("/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"order": c.Param("id")})
		})
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Println("HTTP server started on :8080")
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server error: %w", err)
	}
	return nil
}

func main() {
	// Initialize context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to user service
	userConn, err := grpc.DialContext(ctx, "localhost:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(loggingInterceptor),
	)
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	defer userConn.Close()

	// Initialize gRPC gateway
	gwMux := runtime.NewServeMux()
	if err := pb.RegisterUserServiceHandlerFromEndpoint(ctx, gwMux, "localhost:50052", []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}); err != nil {
		log.Fatalf("Failed to register gateway handler: %v", err)
	}

	// Dual-protocol server startup
	var wg sync.WaitGroup
	wg.Add(2)
	errChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		if err := startGRPCServer(ctx, pb.NewUserServiceClient(userConn)); err != nil {
			errChan <- fmt.Errorf("gRPC server: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := startHTTPServer(ctx, gwMux); err != nil {
			errChan <- fmt.Errorf("HTTP server: %w", err)
		}
	}()

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received termination signal: %v", sig)
		cancel()
	case err := <-errChan:
		log.Printf("Service error: %v", err)
		cancel()
	}

	// Wait for graceful shutdown
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All services stopped safely")
	case <-time.After(10 * time.Second):
		log.Println("Warning: Service shutdown timeout")
	}
}

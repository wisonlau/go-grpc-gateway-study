package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	pb "user-service/proto"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
}

var logCh = make(chan string, 10000)

func init() {
	go func() {
		for msg := range logCh {
			log.Print(msg)
		}
	}()
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	logCh <- fmt.Sprintf("Received GetUser request for ID: %s", req.UserId)
	return &pb.GetUserResponse{
		Id:    req.UserId,
		Name:  "John Doe",
		Email: "john@example.com",
	}, nil
}

func (s *userServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	logCh <- fmt.Sprintf("Received CreateUser request: %s, %s", req.Name, req.Email)
	return &pb.CreateUserResponse{
		Id:    "123",
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			start := time.Now()
			defer func() {
				logCh <- fmt.Sprintf("[gRPC] %s | Duration: %v", info.FullMethod, time.Since(start))
			}()
			logCh <- fmt.Sprintf("gRPC call: %s", info.FullMethod)
			return handler(ctx, req)
		}),
	)
	pb.RegisterUserServiceServer(s, &userServer{})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		s.GracefulStop()
		close(logCh)
	}()

	logCh <- fmt.Sprintf("User gRPC service started on :50052")
	if err := s.Serve(lis); err != nil {
		logCh <- fmt.Sprintf("failed to serve: %v", err)
	}
	wg.Wait()
}

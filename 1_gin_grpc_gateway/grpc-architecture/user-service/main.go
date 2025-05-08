package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	pb "user-service/proto"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	log.Printf("Received GetUser request for ID: %s", req.UserId)
	return &pb.GetUserResponse{
		Id:    req.UserId,
		Name:  "John Doe",
		Email: "john@example.com",
	}, nil
}

func (s *userServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	log.Printf("Received CreateUser request: %s, %s", req.Name, req.Email)
	return &pb.CreateUserResponse{
		Id:    "123",
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			log.Printf("gRPC call: %s", info.FullMethod)
			return handler(ctx, req)
		}),
	)
	pb.RegisterUserServiceServer(s, &userServer{})

	log.Println("User gRPC service started on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

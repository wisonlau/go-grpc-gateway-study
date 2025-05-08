package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "gateway/proto"
)

var (
	userClient pb.UserServiceClient
	grpcServer *grpc.Server
)

func init() {
	conn, err := grpc.Dial("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	userClient = pb.NewUserServiceClient(conn)
}

type gatewayServer struct {
	pb.UnimplementedUserServiceServer
}

func (s *gatewayServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	log.Printf("[gRPC] Received GetUser request for user ID: %s", req.UserId)
	return userClient.GetUser(ctx, req)
}

func (s *gatewayServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	log.Printf("[gRPC] Received CreateUser request: Name=%s, Email=%s", req.Name, req.Email)
	return userClient.CreateUser(ctx, req)
}

func startGRPCServer(wg *sync.WaitGroup) {
	defer wg.Done()

	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer = grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &gatewayServer{})

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve gRPC: %v", err)
	}
}

func startHTTPServer(wg *sync.WaitGroup) {
	defer wg.Done()

	router := gin.Default()

	router.GET("/user/:id", getUserHandler)
	router.POST("/user", createUserHandler)

	log.Println("HTTP server started on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}

func getUserHandler(c *gin.Context) {
	userID := c.Param("id")

	md := prepareMetadata(c.Request.Header)
	ctx := metadata.NewOutgoingContext(c.Request.Context(), md)

	res, err := userClient.GetUser(ctx, &pb.GetUserRequest{UserId: userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func createUserHandler(c *gin.Context) {
	var req pb.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	md := prepareMetadata(c.Request.Header)
	ctx := metadata.NewOutgoingContext(c.Request.Context(), md)

	res, err := userClient.CreateUser(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func prepareMetadata(header http.Header) metadata.MD {
	md := metadata.New(make(map[string]string))
	for k, v := range header {
		md.Set(strings.ToLower(k), v...)
	}
	return md
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	// 启动 gRPC 服务
	go startGRPCServer(&wg)

	// 启动 HTTP 服务
	go startHTTPServer(&wg)

	wg.Wait()
}

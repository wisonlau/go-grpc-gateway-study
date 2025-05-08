package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "gateway/proto"
)

var userClient pb.UserServiceClient

func init() {
	conn, err := grpc.Dial("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	userClient = pb.NewUserServiceClient(conn)
}

func main() {
	router := gin.Default()

	router.GET("/user/:id", getUserHandler)
	router.POST("/user", createUserHandler)

	log.Println("API Gateway started on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("failed to serve: %v", err)
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

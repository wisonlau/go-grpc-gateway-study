# catalogue
```
grpc-architecture/
├── gateway/
│   ├── go.mod
│   ├── go.sum
│   └── main.go
├── user-service/
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   └── proto/
│       └── user.proto
└── proto/
└── user.proto
```

# generate files
```shell
cd gateway
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/user.proto

cd user-service
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/user.proto
```

### run
```shell
cd user-service
go run main.go

cd gateway
go run main.go
```

### test request
```shell
curl http://localhost:8080/user/123

curl -X POST http://localhost:8080/user \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "email": "alice@example.com"}'

curl http://localhost:8080/health

curl http://localhost:8080/orders

grpcurl -plaintext -proto ./proto/user.proto -d '{"user_id": "123"}' localhost:8081 user.UserService/GetUser
grpcurl -plaintext -proto ./proto/user.proto -d '{"name": "Alice", "email": "alice@example.com"}' localhost:8081 user.UserService/CreateUser
```
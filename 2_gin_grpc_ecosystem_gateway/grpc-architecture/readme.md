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
git clone https://github.com/googleapis/googleapis.git /tmp/googleapis
protoc -I=. -I=/tmp/googleapis \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
    proto/user.proto

# --go_out=. --go_opt=paths=source_relative # generate user.pb.go
# --go-grpc_out=. --go-grpc_opt=paths=source_relative # generate user_grpc.pb.go
# --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative # generate user.pb.gw.go

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

use gin + grpc-ecosystem
```shell
curl http://localhost:8080/health

curl http://localhost:8080/orders

curl http://localhost:8080/api/user/123

curl -X POST http://localhost:8080/api/user \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "email": "alice@example.com"}'

grpcurl -plaintext -proto ./proto/user.proto -d '{"user_id": "123"}' localhost:8081 user.UserService/GetUser
grpcurl -plaintext -proto ./proto/user.proto -d '{"name": "Alice", "email": "alice@example.com"}' localhost:8081 user.UserService/CreateUser
```
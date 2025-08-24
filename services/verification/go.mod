module github.com/lcanady/depin/services/verification

go 1.21

require (
    github.com/google/uuid v1.5.0
    github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.0
    google.golang.org/grpc v1.60.1
    google.golang.org/protobuf v1.32.0
    github.com/sirupsen/logrus v1.9.3
    github.com/spf13/viper v1.18.2
    github.com/go-redis/redis/v8 v8.11.5
    gorm.io/gorm v1.25.5
    gorm.io/driver/postgres v1.5.4
)

require (
    github.com/golang/protobuf v1.5.3 // indirect
    golang.org/x/net v0.19.0 // indirect
    golang.org/x/sys v0.15.0 // indirect
    golang.org/x/text v0.14.0 // indirect
    google.golang.org/genproto/googleapis/api v0.0.0-20231212172506-995d672761c0 // indirect
    google.golang.org/genproto/googleapis/rpc v0.0.0-20231212172506-995d672761c0 // indirect
)
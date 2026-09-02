module github.com/roadrunner-server/redis/v6

go 1.27

toolchain go1.27.1

require (
	github.com/prometheus/client_golang v1.24.1
	github.com/redis/go-redis/extra/redisotel/v9 v9.22.0
	github.com/redis/go-redis/extra/redisprometheus/v9 v9.22.0
	github.com/redis/go-redis/v9 v9.22.0
	github.com/roadrunner-server/api-plugins/v6 v6.0.0-beta.2
	github.com/roadrunner-server/endure/v2 v2.6.2
	github.com/roadrunner-server/errors v1.5.0
	github.com/stretchr/testify v1.12.1
	go.opentelemetry.io/otel/sdk v1.46.0
)

exclude github.com/redis/go-redis/v9 v9.15.0

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.3 // indirect
	github.com/prometheus/common v0.71.0 // indirect
	github.com/prometheus/procfs v0.22.0 // indirect
	github.com/redis/go-redis/extra/rediscmd/v9 v9.22.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.47.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

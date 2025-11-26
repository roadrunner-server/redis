module github.com/roadrunner-server/redis/v5

go 1.25

toolchain go1.25.4

require (
	github.com/prometheus/client_golang v1.23.2
	github.com/redis/go-redis/extra/redisotel/v9 v9.17.0
	github.com/redis/go-redis/extra/redisprometheus/v9 v9.17.0
	github.com/redis/go-redis/v9 v9.17.1
	github.com/roadrunner-server/api/v4 v4.23.0
	github.com/roadrunner-server/endure/v2 v2.6.2
	github.com/roadrunner-server/errors v1.4.1
	go.opentelemetry.io/otel/sdk v1.38.0
	go.uber.org/zap v1.27.1
	golang.org/x/sys v0.38.0
)

exclude github.com/redis/go-redis/v9 v9.15.0

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.4 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/redis/go-redis/extra/rediscmd/v9 v9.17.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

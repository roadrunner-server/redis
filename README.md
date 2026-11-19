# Redis plugin for RoadRunner

Use Redis for [KV storage](https://docs.roadrunner.dev/key-value/redis) or as a Jobs backend. Jobs requires Redis 8.0 or later.

```yaml
version: '3'
redis:
  addrs: ["127.0.0.1:6379"]
  db: 1
jobs:
  pipelines:
    mail:
      driver: redis
      config:
        prefetch: 10
        visibility_timeout: 5m
  consume: [mail]
```

Each consumer group shares one queue across RoadRunner instances. With manual acknowledgment, another instance can recover a job after the visibility timeout. Set this timeout longer than the time a job waits for a worker plus its execution time. Redis keeps delayed jobs until a consumer can deliver them.

With `auto_ack`, RoadRunner acknowledges the job when it prepares the payload for PHP. Jobs that are still waiting for a local queue slot remain in Redis. An explicit retry after auto-ack returns an error. During shutdown, workers can acknowledge or requeue active deliveries until the shutdown deadline.

Use a separate database for KV if the application calls `kv.Clear`. This operation clears the entire Redis database. On Redis Cluster, use database 0 and a separate deployment for KV.

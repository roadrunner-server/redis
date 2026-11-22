# Changelog

## [6.1.0](https://github.com/roadrunner-server/redis/compare/v6.0.0...v6.1.0) (2026-11-22)


### Features

* add a Redis Streams driver for Jobs ([c6f84e5](https://github.com/roadrunner-server/redis/commit/c6f84e573b470eec0f6b635c3686aa1d45453824))
* expose Redis commands over RPC ([#250](https://github.com/roadrunner-server/redis/issues/250)) ([22242ae](https://github.com/roadrunner-server/redis/commit/22242ae7686ec1dffe0d99dc244fa6ad9a59f98c))
* register Redis Jobs pipelines from configuration and RPC ([18e0ba8](https://github.com/roadrunner-server/redis/commit/18e0ba8bfbc7a20934dc2110651b2f64762a2982))
* store and recover Jobs deliveries in Redis Streams ([43d36e9](https://github.com/roadrunner-server/redis/commit/43d36e9e17c5a4c083ffc3dea5585f2fe1216591))


### Bug Fixes

* **jobs:** bound pending scans and recover the Redis queue ([4b1c040](https://github.com/roadrunner-server/redis/commit/4b1c040f125ed0b5c35ccd001caca14633ff3f19))
* **jobs:** preserve deliveries through pause and graceful shutdown ([d23e47c](https://github.com/roadrunner-server/redis/commit/d23e47c7fa061f580574f0b61f53eab347e4fccf))
* **jobs:** release Redis consumers and client metrics on stop ([ebcc540](https://github.com/roadrunner-server/redis/commit/ebcc54009aec8bb90c6dd61dbd408fbd7246cd98))
* **jobs:** retire the previous Redis consumer after its last claim ([8a55f72](https://github.com/roadrunner-server/redis/commit/8a55f72a65b23e91e74fa3517daa3a4be2393ca0))

## Changelog

## Unreleased

- Add a Redis Streams driver for Jobs on Redis 8.0 and later. It supports delayed jobs, worker acknowledgments, and recovery of pending deliveries.

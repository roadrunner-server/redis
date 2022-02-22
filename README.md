RRv2 Redis plugin

![kv-general-info](https://user-images.githubusercontent.com/2461257/128436785-3dadbf0d-13c3-4e0c-859c-4fd9668558c8.png)

### Redis Driver Configuration

Before configuring the Redis driver, please make sure that the Redis Server is
installed and running. You can read more about this [in the documentation](https://redis.io/).

In the simplest case, when a full-fledged cluster or a fault-tolerant system is
not required, we have one connection to the Redis Server. The configuration of
such a connection will look like this.

```yaml
version: "2.7"

kv:
  # User defined name of the storage.
  redis:
    # Required section.
    # Should be "redis" for the redis driver.
    driver: redis

    config:
      # Optional section.
      # By default, one connection will be specified with the
      # "localhost:6379" value.
      addrs:
        - "localhost:6379"

      # Optional section.
      # Default: ""
      username: ""

      # Optional section.
      # Default: ""
      password: ""

      # Optional section.
      # Default: 0
      db: 0

      # Optional section.
      # Default: 0 (equivalent to the default value of 5 seconds)
      dial_timeout: 0

      # Optional section.
      # Default: 0 (equivalent to the default value of 3 retries)
      max_retries: 0

      # Optional section.
      # Default: 0 (equivalent to the default value of 8ms)
      min_retry_backoff: 0

      # Optional section.
      # Default: 0 (equivalent to the default value of 512ms)
      max_retry_backoff: 0

      # Optional section.
      # Default: 0 (equivalent to the default value of 10 connections per CPU).
      pool_size: 0

      # Optional section.
      # Default: 0 (do not use idle connections)
      min_idle_conns: 0

      # Optional section.
      # Default: 0 (do not close aged connections)
      max_conn_age: 0

      # Optional section.
      # Default: 0 (equivalent to the default value of 3s)
      read_timeout: 0

      # Optional section.
      # Default: 0 (equivalent to the value specified in the "read_timeout" section)
      write_timeout: 0

      # Optional section.
      # Default: 0 (equivalent to the value specified in the "read_timeout" + 1s)
      pool_timeout: 0

      # Optional section.
      # Default: 0 (equivalent to the default value of 5m)
      idle_timeout: 0

      # Optional section.
      # Default: 0 (equivalent to the default value of 1m)
      idle_check_freq: 0

      # Optional section.
      # Default: false
      read_only: false
```

Below is a more detailed description of each of the Redis-specific options:

- `addrs` - An array of strings of connections to the Redis Server. Must
  contain at least one value of an existing connection in the format of host or
  IP address and port, separated by a colon (`:`) character.

- `username` - Optional value containing the username credentials of the Redis
  connection. You can omit this field, or specify an empty string if the 
  username of the connection is not specified.

- `password` - Optional value containing the password credentials of the Redis
  connection. You can omit this field, or specify an empty string if the
  password of the connection is not specified.

- `db` - An optional identifier for the database used in this connection to the
  Redis Server. Read more about databases section on the documentation page for
  the description of the [select command](https://redis.io/commands/select).

- `dial_timeout` - Server connection timeout. A value of `0` is equivalent to a 
  timeout of 5 seconds (`5s`). After the specified time has elapsed, if the
  connection has not been established, a connection error will occur.

  Must be in the format of a "numeric value" + "time format suffix", like "`2h`" where 
  suffixes means:
  - `h` - the number of hours. For example `1h` means 1 hour.
  - `m` - the number of minutes. For example `2m` means 2 minutes.
  - `s` - the number of seconds. For example `3s` means 3 seconds.
  - `ms` - the number of milliseconds. For example `4ms` means 4 milliseconds.
  - If no suffix is specified, the value will be interpreted as specified in
    nanoseconds. In most cases, this accuracy is redundant and may not be true. 
    For example `5` means 5 nanoseconds.

  Please note that all time intervals can be suffixed.

- `max_retries` - Maximum number of retries before giving up. Specifying `0` is
  equivalent to the default (`3` attempts). If you need to specify an infinite
  number of connection attempts, specify the value `-1`.

- `min_retry_backoff` - Minimum backoff between each retry. Must be in the format
  of a "numeric value" + "time format suffix". A value of `0` is equivalent to a
  timeout of 8 milliseconds (`8ms`). A value of `-1` disables backoff.

- `max_retry_backoff` - Maximum backoff between each retry. Must be in the format
  of a "numeric value" + "time format suffix". A value of `0` is equivalent to a
  timeout of 512 milliseconds (`512ms`). A value of `-1` disables backoff.

- `pool_size` - Maximum number of RoadRunner socket connections. A value of `0`
  is equivalent to a `10` connections per every CPU. Please note that specifying
  the value corresponds to the number of connections **per core**, so if you
  have 8 cores in your system, then setting the option to 2 you will get 16
  connections.

- `min_idle_conns` - Minimum number of idle connections which is useful when
  establishing new connection is slow. A value of 0 means no such idle
  connections. More details about the problem requiring the presence of this
  option available in the [corresponding issue](https://github.com/go-redis/redis/issues/772).

- `max_conn_age` - Connection age at which client retires (closes) the connection.
  A value of `0` is equivalent to a disabling this option. In this case, aged
  connections will not be closed.

- `read_timeout` - Timeout for socket reads. If reached, commands will fail with
  a timeout instead of blocking. Must be in the format of a "numeric value" + 
  "time format suffix". A value of `0` is equivalent to a timeout of 3 seconds
  (`3s`). A value of `-1` disables timeout.

- `write_timeout` - Timeout for socket writes. If reached, commands will fail
  with a timeout instead of blocking. A value of `0` is equivalent of the value
  specified in the `read_timeout` section. If `read_timeout` value is not 
  specified, a value of 3 seconds (`3s`) will be used.

- `pool_timeout` - Amount of time client waits for connection if all connections
  are busy before returning an error. A value of `0` is equivalent of the value
  specified in the `read_timeout` + `1s`. If `read_timeout` value is not
  specified, a value of 4 seconds (`4s`) will be used.

- `idle_timeout` - Amount of time after which client closes idle connections.
  Must be in the format of a "numeric value" + "time format suffix". A value of
  `0` is equivalent to a timeout of 5 minutes (`5m`). A value of `-1` disables
  idle timeout check.

- `idle_check_freq` - Frequency of idle checks made by idle connections reaper.
  Must be in the format of a "numeric value" + "time format suffix". A value of
  `0` is equivalent to a timeout of 1 minute (`1m`). A value of `-1` disables
  idle connections reaper. Note, that idle connections are still discarded by
  the client if `idle_timeout` is set.

- `read_only` - An optional boolean value that enables or disables read-only
  mode. If `true` value is specified, the writing will be unavailable.
  Note that this option **not allowed** when working with Redis Sentinel.

These are all options available for all Redis connection types.

#### Redis Cluster

In the case that you want to configure a [Redis Cluster](https://redis.io/topics/cluster-tutorial),
then you can specify additional options required only if you are organizing this
type of server.

When creating a cluster, multiple connections are available to you. For example,
you call such a command `redis-cli --cluster create 127.0.0.1:6379 127.0.0.1:6380`,
you should specify the appropriate set of connections. In addition, when
organizing a cluster, two additional options with algorithms for working with
connections will be available to you: `route_by_latency` and `route_randomly`.

```yaml
version: "2.7"

kv:
  redis:
    driver: redis
    config:
      addrs:
        - "127.0.0.1:6379"
        - "127.0.0.1:6380"

      # Optional section.
      # Default: false
      route_by_latency: false

      # Optional section.
      # Default: false
      route_randomly: false
```

Where new options means:

- `route_by_latency` - Allows routing read-only commands to the closest master
  or slave node. If this option is specified, the `read_only` configuration value
  will be automatically set to `true`.

- `route_randomly` - Allows routing read-only commands to the random master or
  slave node. If this option is specified, the `read_only` configuration value
  will be automatically set to `true`.

#### Redis Sentinel

Redis Sentinel provides high availability for Redis. You can find more 
information about [Sentinel on the documentation page](https://redis.io/topics/sentinel).

There are two additional options available for the Sentinel configuration: 
`master_name` and `sentinel_password`.

```yaml
version: "2.7"

kv:
  redis:
    driver: redis

    config:
      # Required section.
      master_name: ""

      # Optional section.
      # Default: "" (no password)
      sentinel_password: ""
```

Where Sentinel's options means:

- `master_name` - The name of the Sentinel's master in string format.

- `sentinel_password` - Sentinel password from "requirepass <password>" 
  (if enabled) in Sentinel configuration.
  
  ## Usage

First, you need to create the RPC connection to the RoadRunner server. You can
specify an address with a connection by hands or use automatic detection if
you run the php code as a [RoadRunner Worker](/php/worker.md).

```php
use Spiral\RoadRunner\Environment;
use Spiral\Goridge\RPC\RPC;

// Manual configuration
$rpc = RPC::create('tcp://127.0.0.1:6001');

// Autodetection
$env = Environment::fromGlobals();
$rpc = RPC::create($env->getRPCAddress());
```

After creating the RPC connection, you should create the
`Spiral\RoadRunner\KeyValue\Factory` object for working with storages of KV
RoadRunner plugin.

The factory object provides two methods for working with the plugin.

- Method `Factory::isAvailable(): bool` returns boolean `true` value if the
  plugin is available and `false` otherwise. Note, that this method will always return an `Exception` because it was removed from the RR RPC since `v2.6.2`, [issue](https://github.com/roadrunner-server/roadrunner/issues/901).  In the releases after `v2.6.2` you can safely remove calls to that method.

- Method `Factory::select(string): CacheInterface` receives the name of the
  storage as the first argument and returns the implementation of the
  [PSR-16](https://www.php-fig.org/psr/psr-16/) `Psr\SimpleCache\CacheInterface`
  for interact with the key-value RoadRunner storage.

```php
use Spiral\Goridge\RPC\RPC;
use Spiral\RoadRunner\KeyValue\Factory;

$factory = new Factory(RPC::create('tcp://127.0.0.1:6001'));

if (!$factory->isAvailable()) {
    throw new \LogicException('The "kv" RoadRunner plugin not available');
}

$storage = $factory->select('storage-name');
// Expected:
//  An instance of Psr\SimpleCache\CacheInterface interface

$storage->set('key', 'value');

echo $storage->get('key');
// Expected:
//  string(5) "string"
```

> The `clear()` method available since [RoadRunner v2.3.1](https://github.com/spiral/roadrunner/releases/tag/v2.3.1).

Apart from this, RoadRunner Key-Value API provides several additional methods:
You can use `getTtl(string): ?\DateTimeInterface` and
`getMultipleTtl(string): iterable<\DateTimeInterface|null>` methods to get
information about the expiration of an item stored in a key-value storage.

> Please note that the `memcached` driver
> [**does not support**](https://github.com/memcached/memcached/issues/239)
> these methods.

```php
$ttl = $factory->select('memory')
    ->getTtl('key');
// Expected:
//  - An instance of \DateTimeInterface if "key" expiration time is available
//  - Or null otherwise

$ttl = $factory->select('memcached')
    ->getTtl('key');
// Expected:
//  Spiral\RoadRunner\KeyValue\Exception\KeyValueException: Storage "memcached"
//  does not support kv.TTL RPC method execution. Please use another driver for
//  the storage if you require this functionality.
```

### Value Serialization

To save and receive data from the key-value store, the data serialization
mechanism is used. This way you can store and receive arbitrary serializable
objects.

```php
$storage->set('test', (object)['key' => 'value']);

$item = $storage->set('test');
// Expected:
//  object(StdClass)#399 (1) {
//    ["key"] => string(5) "value"
//  }
```

To specify your custom serializer, you will need to specify it in the key-value
factory constructor as a second argument, or use the
`Factory::withSerializer(SerializerInterface): self` method.

```php
use Spiral\Goridge\RPC\RPC;
use Spiral\RoadRunner\KeyValue\Factory;

$connection = RPC::create('tcp://127.0.0.1:6001');

$storage = (new Factory($connection))
    ->withSerializer(new CustomSerializer())
    ->select('storage');
```

In the case that you need a specific serializer for a specific value from the
storage, then you can use a similar method `withSerializer()` for a specific
storage.

```php
// Using default serializer
$storage->set('key', 'value');

// Using custom serializer
$storage
    ->withSerializer(new CustomSerializer())
    ->set('key', 'value');
```


#### Igbinary Value Serialization

As you know, the serialization mechanism in PHP is not always productive. To
increase the speed of work, it is recommended to use the
[ignbinary extension](https://github.com/igbinary/igbinary).

- For the Windows OS, you can download it from the
  [PECL website](https://windows.php.net/downloads/pecl/releases/igbinary/).

- In a Linux and MacOS environment, it may be installed with a simple command:
```sh
$ pecl install igbinary
```

More detailed installation instructions are [available here](https://github.com/igbinary/igbinary#installing).

After installing the extension, you just need to install the desired igbinary
serializer in the factory instance.

```php
use Spiral\Goridge\RPC\RPC;
use Spiral\RoadRunner\KeyValue\Factory;
use Spiral\RoadRunner\KeyValue\Serializer\IgbinarySerializer;

$storage = (new Factory(RPC::create('tcp://127.0.0.1:6001')))
    ->withSerializer(new IgbinarySerializer())
    ->select('storage');
//
// Now this $storage is using igbinary serializer.
//
```

#### End-to-End Value Encryption

Some data may contain sensitive information, such as personal data of the user.
In these cases, it is recommended to use data encryption.

To use encryption, you need to install the
[Sodium extension](https://www.php.net/manual/en/book.sodium.php).

Next, you should have an encryption key generated using
[sodium_crypto_box_keypair()](https://www.php.net/manual/en/function.sodium-crypto-box-keypair.php)
function. You can do this using the following command:
```sh
$ php -r "echo sodium_crypto_box_keypair();" > keypair.key
```

> Do not store security keys in a control versioning systems (like GIT)!

After generating the keypair, you can use it to encrypt and decrypt the data.

```php
use Spiral\Goridge\RPC\RPC;
use Spiral\RoadRunner\KeyValue\Factory;
use Spiral\RoadRunner\KeyValue\Serializer\SodiumSerializer;
use Spiral\RoadRunner\KeyValue\Serializer\DefaultSerializer;

$storage = new Factory(RPC::create('tcp://127.0.0.1:6001'));
    ->select('storage');

// Encrypted serializer
$key = file_get_contents(__DIR__ . '/path/to/keypair.key');
$encrypted = new SodiumSerializer($storage->getSerializer(), $key);

// Storing public data
$storage->set('user.login', 'test');

// Storing private data
$storage->withSerializer($encrypted)
    ->set('user.email', 'test@example.com');
```

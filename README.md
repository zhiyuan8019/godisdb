# `GodisDB`

## `GodisDB` : 事件驱动的KV数据库

`GodisDB`是Redis的简略版本，采用与Redis相同的基础架构。数据库整体由事件驱动，其中文件事件由`Epoll`提供多路IO复用。在服务端和客户端通信中使用了`protobuf`作为序列化方式。

**主要特性**：

- **事件驱动架构**：基于`AE事件库`，支持文件事件和时间事件。通过`epoll`实现多路I/O复用，保证高效的并发处理。

- **命令表**：所有的操作命令均保存在一个`map`中，每个命令关联有一个特定的回调函数。

- **S-C通信**：在服务端和客户端之间，采用`protobuf`作为序列化方式，确保数据传输的高效，并保留了扩展性。

- **类型支持**：支持Redis早期版本中的`五`大核心数据类型

## 数据结构对应


```go
String GODIS_STRING  string                              
List   GODIS_LIST    GodisList
Hash   GODIS_HASH    map[string]*GodisObj
Set    GODIS_SET     map[string]*GodisObj
Zset   GODIS_ZSET    GodisSkipList + map[string]float64 
```

## 命令支持

实现了Redis 3.0版本的大部分命令，包括但不限于基础的键值操作、列表操作、有序集合操作等。详情请见[godis/command.go](./godis/command.go)


## 测试

### PING 

```bash
» ping
PONG
```

### Err

```bash
» set 1 2 3
(error) ERR wrong number of arguments for 'set' command
» set key value
OK
» hget key value
(error) WRONGTYPE Operation against a key holding the wrong kind of value
» zadd key1 notfloat value
(error) ERR value is not a valid float
```

### String 

```bash
» ping
PONG
» set key value
OK
» get key
"value"
» set K V
OK
» exists key K
(integer) 2
» del key
(integer) 1
» get key
(nil)
» exists key
(integer) 0
```

### Expire

```bash
» set key value
OK
» expire key 10
(integer) 1
» get key
"value"
» get key
(nil)
```

### List

```bash
» lpush key c b a
(integer) 3
» rpush key d e f
(integer) 6
» lrange key -6 -1
1) " a"
2) " b"
3) " c"
4) " d"
5) " e"
6) " f"
» lset key 0 new
OK
» lrange key -6 -1
1) " new"
2) " b"
3) " c"
4) " d"
5) " e"
6) " f"
» lpop key
"new"
» llen key
(integer) 5
» lindex key 4
"f"
```




### Hash

```bash
» hset key field value
(integer) 1
» hget key field
"value"
» hexists key field
(integer) 1
» hdel key field
(integer) 1
» hlen key
(integer) 0
» hset key f1 v1 f2 v2
(integer) 2
» hgetall key
1) " f2"
2) " v2"
3) " f1"
4) " v1"
```

### Set

```bash
» sadd key 1 2 3 4 5 6
(integer) 6
» scard key
(integer) 6
» sismember key 7
(integer) 0
» smembers key
1) " 1"
2) " 2"
3) " 3"
4) " 4"
5) " 5"
6) " 6"
» srem key 1
(integer) 1
» smembers key
1) " 5"
2) " 6"
3) " 2"
4) " 3"
5) " 4"

```


### Zset

```bash
» zadd key 1 a 2 b 3.1 c
(integer) 3
» zcard key
(integer) 3
» zcount key 1 3
(integer) 2
» zcount key 1 3.1
(integer) 3
» zrange key 0 10
1) " a"
2) " b"
3) " c"
» zrank key b
(integer) 1
» zscore key c
"3.10"
» zrem key a
(integer) 1
```

## 注意

- server基于epoll仅linux可用
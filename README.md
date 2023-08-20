# `GodisDB`

## `GodisDB` : 事件驱动的KV数据库

GodisDB是Redis的简略版本，采用与Redis相同的基础架构。数据库整体由事件驱动，其中文件事件由`Epoll`提供多路IO复用。在服务端和客户端通信中使用了`protobuf`作为序列化方式

**主要特性**：

- **AE事件库**：包括文件事件和时间事件，每个事件均有回调函数，通过`epoll`事件或时间触发。

- **命令表**：通过`map`保存命令对应的回调函数。

- **S-C通信**：通过`protobuf`的`message`编写传输消息数据结构，经序列化后发送。

- **redisObj**：实现了Redis早期版本的全部`5`类对象

## 数据结构对应


```go
String GODIS_STRING string                              
List   GODIS_LIST    GodisList
Hash   GODIS_HASH   map[string]*GodisObj
Set    GODIS_SET  map[string]*GodisObj
Zset   GODIS_ZSETGodisSkipList + map[string]float64 
```

## 功能

- ✅ String

- ✅ List

- ✅ Hash

- ✅ Set

- ✅ Zset

## 组件

- ✅ ae事件库

- ✅ Client

- ✅ Server

- ✅ List

- ✅ SkipList


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
package godis

import (
	"errors"
	"fmt"
	myProto "godisdb/proto"
	"log"
	"strconv"
)

type CommandProc func(c *GodisClient)

type CommandType int

const (
	WRITE_COMMAND CommandType = 0x01
	READ_COMMAND  CommandType = 0x02
	ADMIN_COMMAND CommandType = 0x04
)

type GodisCommand struct {
	name         string
	proc         CommandProc
	arity        int
	mask         CommandType
	microseconds int
	calls        int
	arity_more   bool
}

var CommandTable map[string]GodisCommand

func initCommandTable() {
	CommandTable = map[string]GodisCommand{
		"ping":   {"ping", pingCommand, 1, ADMIN_COMMAND, 0, 0, false},
		"get":    {"get", getCommand, 2, READ_COMMAND, 0, 0, false},
		"set":    {"set", setCommand, 3, WRITE_COMMAND, 0, 0, false},
		"del":    {"del", delCommand, 2, WRITE_COMMAND, 0, 0, true},
		"exists": {"exists", existsCommand, 2, READ_COMMAND, 0, 0, true},
		"expire": {"expire", expireCommand, 3, WRITE_COMMAND, 0, 0, false},

		"lpush":  {"lpush", lpushCommand, 3, WRITE_COMMAND, 0, 0, true},
		"rpush":  {"rpush", rpushCommand, 3, WRITE_COMMAND, 0, 0, true},
		"lpop":   {"lpop", lpopCommand, 2, WRITE_COMMAND, 0, 0, false},
		"rpop":   {"rpop", rpopCommand, 2, WRITE_COMMAND, 0, 0, false},
		"llen":   {"llen", llenCommand, 2, READ_COMMAND, 0, 0, false},
		"lindex": {"lindex", lindexCommand, 3, READ_COMMAND, 0, 0, false},
		"lset":   {"lset", lsetCommand, 4, WRITE_COMMAND, 0, 0, false},
		"lrange": {"lrange", lrangeCommand, 4, READ_COMMAND, 0, 0, false},

		"hset":    {"hset", hsetCommand, 4, WRITE_COMMAND, 0, 0, true}, //need more check count%2
		"hget":    {"hget", hgetCommand, 3, READ_COMMAND, 0, 0, false},
		"hexists": {"hexists", hexistsCommand, 3, READ_COMMAND, 0, 0, false},
		"hdel":    {"hdel", hdelCommand, 3, WRITE_COMMAND, 0, 0, true},
		"hlen":    {"hlen", hlenCommand, 2, READ_COMMAND, 0, 0, false},
		"hgetall": {"hgetall", hgetallCommand, 2, READ_COMMAND, 0, 0, false},

		"sadd":      {"sadd", saddCommand, 3, WRITE_COMMAND, 0, 0, true},
		"scard":     {"scard", scardCommand, 2, READ_COMMAND, 0, 0, false},
		"sismember": {"sismember", sismemberCommand, 3, READ_COMMAND, 0, 0, false},
		"smembers":  {"smembers", sinterCommand, 2, READ_COMMAND, 0, 0, false},
		"srem":      {"srem", sremCommand, 3, WRITE_COMMAND, 0, 0, true},

		"zadd":   {"zadd", zaddCommand, 4, WRITE_COMMAND, 0, 0, true},
		"zcard":  {"zcard", zcardCommand, 2, READ_COMMAND, 0, 0, false},
		"zcount": {"zcount", zcountCommand, 4, READ_COMMAND, 0, 0, false},
		"zrange": {"zrange", zrangeCommand, 4, READ_COMMAND, 0, 0, false},
		"zrank":  {"zrank", zrankCommand, 3, READ_COMMAND, 0, 0, false},
		"zrem":   {"zrem", zremCommand, 3, WRITE_COMMAND, 0, 0, true},
		"zscore": {"zscore", zscoreCommand, 3, READ_COMMAND, 0, 0, false},
	}
}

func checkDel(c *GodisClient, key string) bool {

	if when, ok := c.db.expires[key]; ok {
		now := GetMsTime()
		if when < now {
			delete(c.db.dict, key)
			delete(c.db.expires, key)
			return true
		}
	}
	return false
}

var str_pong string = "PONG"
var str_ok string = "OK"
var str_err_wrongtype string = "WRONGTYPE Operation against a key holding the wrong kind of value"
var str_err_outrange string = "ERR value is not an integer or out of range"
var str_err_nokey string = "ERR no such key"
var str_err_notfloat string = "ERR value is not a valid float"

func genReply(c *GodisClient, re_type ReplyType, s *string, d int, slice []string) {
	switch re_type {
	case RE_NONE:
		c.reply = append(c.reply, myProto.Reply{
			ReplyType: int64(RE_NONE),
		})
	case RE_OK:
		c.reply = append(c.reply, myProto.Reply{
			Args:      []string{*s},
			ReplyType: int64(RE_OK),
		})
	case RE_ERR:
		c.reply = append(c.reply, myProto.Reply{
			Args:      []string{*s},
			ReplyType: int64(RE_ERR),
		})
	case RE_STRING:
		c.reply = append(c.reply, myProto.Reply{
			Args:      []string{*s},
			ReplyType: int64(RE_STRING),
		})
	case RE_INT:
		tmp := fmt.Sprintf("%d", d)
		c.reply = append(c.reply, myProto.Reply{
			Args:      []string{tmp},
			ReplyType: int64(RE_INT),
		})
	case RE_HASH:
		c.reply = append(c.reply, myProto.Reply{
			Args:      slice,
			ReplyType: int64(RE_HASH),
		})
	case RE_LIST:
		c.reply = append(c.reply, myProto.Reply{
			Args:      slice,
			ReplyType: int64(RE_LIST),
		})
	case RE_SET:
		c.reply = append(c.reply, myProto.Reply{
			Args:      slice,
			ReplyType: int64(RE_SET),
		})
	case RE_ZSET:
		c.reply = append(c.reply, myProto.Reply{
			Args:      slice,
			ReplyType: int64(RE_ZSET),
		})
	case RE_FLOAT:
		c.reply = append(c.reply, myProto.Reply{
			Args:      []string{*s},
			ReplyType: int64(RE_FLOAT),
		})
	}
}

func checkArgsCount(c *GodisClient) error {
	more := CommandTable[c.command].arity_more
	arity := CommandTable[c.command].arity
	if (!more && c.arg_count != (arity-1)) || (more && c.arg_count < (arity-1)) {
		s := fmt.Sprintf("ERR wrong number of arguments for '%s' command", c.command)
		genReply(c, RE_ERR, &s, 0, nil)
		return errors.New("Args Count Err")
	}
	return nil
}

func pingCommand(c *GodisClient) {
	genReply(c, RE_OK, &str_pong, 0, nil)
}

func getCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])

	if v, ok := c.db.dict[c.args[0]]; ok {
		if strVal, ok := v.val.(string); ok {
			genReply(c, RE_STRING, &strVal, 0, nil)
		} else {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		}
	} else {
		genReply(c, RE_NONE, nil, 0, nil)
	}
}

func setCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	c.db.dict[c.args[0]] = CreateObj(GODIS_STRING, string(c.args[1]))
	genReply(c, RE_OK, &str_ok, 0, nil)
}

func delCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	var count int = 0
	for _, v := range c.args {
		if _, ok := c.db.dict[v]; ok {
			delete(c.db.dict, v)
			count++
		}
	}
	genReply(c, RE_INT, nil, count, nil)
}

func existsCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	var count int = 0
	for _, v := range c.args {
		checkDel(c, v)
		if _, ok := c.db.dict[v]; ok {
			count++
		}
	}
	genReply(c, RE_INT, nil, count, nil)
}

func expireCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	var count int = 0
	if _, ok := c.db.dict[c.args[0]]; ok {
		ti, err := strconv.Atoi(c.args[1])
		// 10 years
		if err != nil || ti < 0 || ti > 315360000 {
			genReply(c, RE_ERR, &str_err_outrange, 0, nil)
			return
		}
		c.db.expires[c.args[0]] = GetMsTime() + ti*1000
		count++
	}
	genReply(c, RE_INT, nil, count, nil)
}

func hsetCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	// checkArgs do not check "mod 2"
	if c.arg_count%2 != 1 {
		s := fmt.Sprintf("ERR wrong number of arguments for '%s' command", c.command)
		genReply(c, RE_ERR, &s, 0, nil)
		return
	}
	var count int = 0
	if _, ok := c.db.dict[c.args[0]]; !ok {
		c.db.dict[c.args[0]] = CreateObj(GODIS_HASH, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_HASH {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	if hashVal, ok := c.db.dict[c.args[0]].val.(GodisHash); ok {
		for i := 1; i < c.arg_count; i += 2 {
			if _, ok := hashVal[c.args[i]]; !ok {
				hashVal[c.args[i]] = CreateObj(GODIS_STRING, c.args[i+1])
				count++
			} else {
				if _, ok := hashVal[c.args[i]].val.(string); ok {
					hashVal[c.args[i]].val = c.args[i+1]
				} else {
					log.Printf("command %s error\n", c.command)
				}
			}

		}
	} else {
		genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		return
	}
	genReply(c, RE_INT, nil, count, nil)
}

func hgetCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_NONE, nil, 0, nil)
		return
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_HASH {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	if hashVal, ok := c.db.dict[c.args[0]].val.(GodisHash); ok {
		if v, ok := hashVal[c.args[1]]; ok {
			if strVal, ok := v.val.(string); ok {
				genReply(c, RE_STRING, &strVal, 0, nil)
			} else {
				log.Printf("command %s error\n", c.command)
			}
		} else {
			genReply(c, RE_NONE, nil, 0, nil)
			return
		}
	} else {
		genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		return
	}

}

func hexistsCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	var count int = 0
	if _, ok := c.db.dict[c.args[0]]; ok {
		if hashVal, ok := c.db.dict[c.args[0]].val.(GodisHash); ok {
			if _, ok := hashVal[c.args[1]]; ok {
				count++
			}
		}
	}
	genReply(c, RE_INT, nil, count, nil)
}

func hdelCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	var count int = 0
	if _, ok := c.db.dict[c.args[0]]; ok {
		if hashVal, ok := c.db.dict[c.args[0]].val.(GodisHash); ok {
			for i := 1; i < c.arg_count; i++ {
				if _, ok := hashVal[c.args[i]]; ok {
					delete(hashVal, c.args[i])
					count++
				}
			}

		} else {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	genReply(c, RE_INT, nil, count, nil)
}

func hlenCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; ok {
		if hashVal, ok := c.db.dict[c.args[0]].val.(GodisHash); ok {
			genReply(c, RE_INT, nil, len(hashVal), nil)
		} else {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
}

func hgetallCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; ok {
		if hashVal, ok := c.db.dict[c.args[0]].val.(GodisHash); ok {
			tmp_slice := []string{}
			for k, v := range hashVal {
				tmp_slice = append(tmp_slice, k)
				if strVal, ok := v.val.(string); ok {
					tmp_slice = append(tmp_slice, strVal)
				} else {
					log.Printf("command %s error\n", c.command)
				}
			}
			genReply(c, RE_HASH, nil, 0, tmp_slice)
		} else {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
}

func lpushCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	length := 0
	if _, ok := c.db.dict[c.args[0]]; !ok {
		c.db.dict[c.args[0]] = CreateObj(GODIS_LIST, nil)
	}
	if listVal, ok := c.db.dict[c.args[0]].val.(*GodisList); ok {
		for i := 1; i < c.arg_count; i++ {
			listVal.listAddNodeHead(CreateObj(GODIS_STRING, c.args[i]))
		}
		length = int(listVal.listLength())
	} else {
		genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		return
	}

	genReply(c, RE_INT, nil, length, nil)
}

func rpushCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	length := 0
	if _, ok := c.db.dict[c.args[0]]; !ok {
		c.db.dict[c.args[0]] = CreateObj(GODIS_LIST, nil)
	}
	if listVal, ok := c.db.dict[c.args[0]].val.(*GodisList); ok {
		for i := 1; i < c.arg_count; i++ {
			listVal.listAddNodeTail(CreateObj(GODIS_STRING, c.args[i]))
		}
		length = int(listVal.listLength())
	} else {
		genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		return
	}
	genReply(c, RE_INT, nil, length, nil)
}

func lpopCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_NONE, nil, 0, nil)
		return
	}
	if listVal, ok := c.db.dict[c.args[0]].val.(*GodisList); ok {
		node := listVal.listPopNodeHead()
		if node == nil {
			genReply(c, RE_NONE, nil, 0, nil)
		}
		if strVal, ok := node.val.val.(string); ok {
			genReply(c, RE_STRING, &strVal, 0, nil)
		} else {
			log.Printf("command %s error\n", c.command)
		}

	} else {
		genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		return
	}
}

func rpopCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_NONE, nil, 0, nil)
		return
	}
	if listVal, ok := c.db.dict[c.args[0]].val.(*GodisList); ok {
		node := listVal.listPopNodeTail()
		if node == nil {
			genReply(c, RE_NONE, nil, 0, nil)
		}
		if strVal, ok := node.val.val.(string); ok {
			genReply(c, RE_STRING, &strVal, 0, nil)
		} else {
			log.Printf("command %s error\n", c.command)
		}

	} else {
		genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		return
	}
}

func llenCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_INT, nil, 0, nil)
		return
	}
	if listVal, ok := c.db.dict[c.args[0]].val.(*GodisList); ok {
		genReply(c, RE_INT, nil, int(listVal.listLength()), nil)
	} else {
		genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		return
	}
}

func lindexCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_NONE, nil, 0, nil)
		return
	}
	if listVal, ok := c.db.dict[c.args[0]].val.(*GodisList); ok {

		index, err := strconv.Atoi(c.args[1])
		if err != nil || !listVal.listTestIndex(index) {
			genReply(c, RE_ERR, &str_err_outrange, 0, nil)
			return
		}

		node := listVal.listGetIndex(int64(index))
		if strVal, ok := node.val.val.(string); ok {
			genReply(c, RE_STRING, &strVal, 0, nil)
		} else {
			log.Printf("command %s error\n", c.command)
		}
	} else {
		genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		return
	}
}

func lsetCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_ERR, &str_err_nokey, 0, nil)
		return
	}
	if listVal, ok := c.db.dict[c.args[0]].val.(*GodisList); ok {

		index, err := strconv.Atoi(c.args[1])
		if err != nil || !listVal.listTestIndex(index) {
			genReply(c, RE_ERR, &str_err_outrange, 0, nil)
			return
		}

		node := listVal.listGetIndex(int64(index))
		node.val = CreateObj(GODIS_STRING, c.args[2])
		genReply(c, RE_OK, &str_ok, 0, nil)
	} else {
		genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		return
	}
}

func lrangeCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_LIST, nil, 0, []string{})
		return
	}
	if listVal, ok := c.db.dict[c.args[0]].val.(*GodisList); ok {

		start, err := strconv.Atoi(c.args[1])
		if err != nil {
			genReply(c, RE_ERR, &str_err_outrange, 0, nil)
			return
		}
		stop, err := strconv.Atoi(c.args[2])
		if err != nil {
			genReply(c, RE_ERR, &str_err_outrange, 0, nil)
			return
		}

		start = listVal.listAbsIndex(start)

		stop = listVal.listAbsIndex(stop)
		//log.Printf("Debug1 : start %d\n", start)
		//log.Printf("Debug2 : stop  %d\n", stop)
		if start > stop {
			genReply(c, RE_LIST, nil, 0, []string{})
			return
		}
		tmp := []string{}
		node := listVal.listGetIndex(int64(start))
		for start <= stop {

			if strVal, ok := node.val.val.(string); ok {
				tmp = append(tmp, strVal)
			} else {
				log.Printf("command %s error\n", c.command)
			}
			node = node.next
			start++
		}
		genReply(c, RE_LIST, nil, 0, tmp)
	} else {
		genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
		return
	}
}

func saddCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}

	if _, ok := c.db.dict[c.args[0]]; !ok {
		c.db.dict[c.args[0]] = CreateObj(GODIS_SET, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_SET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}

	var count int = 0
	if setVal, ok := c.db.dict[c.args[0]].val.(GodisSet); ok {
		for i := 1; i < c.arg_count; i++ {
			if _, ok := setVal[c.args[i]]; !ok {
				setVal[c.args[i]] = CreateObj(GODIS_NONE, nil)
				count++
			}
		}
		genReply(c, RE_INT, nil, count, nil)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

func scardCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])

	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_INT, nil, 0, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_SET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}

	if setVal, ok := c.db.dict[c.args[0]].val.(GodisSet); ok {
		genReply(c, RE_INT, nil, len(setVal), nil)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

func sismemberCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_INT, nil, 0, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_SET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}

	if setVal, ok := c.db.dict[c.args[0]].val.(GodisSet); ok {
		if _, ok := setVal[c.args[1]]; ok {
			genReply(c, RE_INT, nil, 1, nil)
		} else {
			genReply(c, RE_INT, nil, 0, nil)
		}
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

func sinterCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_SET, nil, 0, []string{})
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_SET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}

	if setVal, ok := c.db.dict[c.args[0]].val.(GodisSet); ok {
		tmp := []string{}
		for k := range setVal {
			tmp = append(tmp, k)
		}
		genReply(c, RE_SET, nil, 0, tmp)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

func sremCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_INT, nil, 0, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_SET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	var count int = 0
	if setVal, ok := c.db.dict[c.args[0]].val.(GodisSet); ok {
		for i := 1; i < c.arg_count; i++ {
			if _, ok := setVal[c.args[i]]; ok {
				delete(setVal, c.args[i])
				count++
			}
		}
		genReply(c, RE_INT, nil, count, nil)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

func zaddCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	if c.arg_count%2 != 1 {
		s := fmt.Sprintf("ERR wrong number of arguments for '%s' command", c.command)
		genReply(c, RE_ERR, &s, 0, nil)
		return
	}
	if _, ok := c.db.dict[c.args[0]]; !ok {
		c.db.dict[c.args[0]] = CreateObj(GODIS_ZSET, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_ZSET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	var count int = 0
	if zsetVal, ok := c.db.dict[c.args[0]].val.(*GodisZset); ok {
		score := []float64{}
		for i := 1; i < c.arg_count; i = i + 2 {
			f, err := strconv.ParseFloat(c.args[i], 64)
			if err != nil {
				genReply(c, RE_ERR, &str_err_notfloat, 0, nil)
				return
			}
			score = append(score, f)
		}
		j := 0
		for i := 1; i < c.arg_count; i = i + 2 {
			if _, ok := zsetVal.dict[c.args[i+1]]; !ok {
				zsetVal.dict[c.args[i+1]] = score[j]
				zsetVal.zskiplist.spInsert(score[j], CreateObj(GODIS_STRING, c.args[i+1]))
				count++
			}
			j++
		}
		genReply(c, RE_INT, nil, count, nil)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

func zcardCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_INT, nil, 0, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_ZSET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	if zsetVal, ok := c.db.dict[c.args[0]].val.(*GodisZset); ok {
		genReply(c, RE_INT, nil, int(zsetVal.zskiplist.length), nil)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

func zcountCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_INT, nil, 0, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_ZSET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	if zsetVal, ok := c.db.dict[c.args[0]].val.(*GodisZset); ok {
		start, err := strconv.ParseFloat(c.args[1], 64)
		if err != nil {
			genReply(c, RE_ERR, &str_err_notfloat, 0, nil)
			return
		}
		stop, err := strconv.ParseFloat(c.args[2], 64)
		if err != nil {
			genReply(c, RE_ERR, &str_err_notfloat, 0, nil)
			return
		}
		d := zsetVal.zskiplist.spGetRangeCount(start, stop)
		genReply(c, RE_INT, nil, d, nil)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

func zrangeCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_INT, nil, 0, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_ZSET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	if zsetVal, ok := c.db.dict[c.args[0]].val.(*GodisZset); ok {
		start, err := strconv.ParseFloat(c.args[1], 64)
		if err != nil {
			genReply(c, RE_ERR, &str_err_notfloat, 0, nil)
			return
		}
		stop, err := strconv.ParseFloat(c.args[2], 64)
		if err != nil {
			genReply(c, RE_ERR, &str_err_notfloat, 0, nil)
			return
		}
		node := zsetVal.zskiplist.spFirstInRange(start, stop)
		tmp := []string{}
		for node != nil && node.score <= stop {
			if strVal, ok := node.obj.val.(string); ok {
				tmp = append(tmp, strVal)
			} else {
				log.Printf("command %s error\n", c.command)
			}
			node = zsetVal.zskiplist.spNext(node)
		}
		genReply(c, RE_ZSET, nil, 0, tmp)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

func zrankCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_INT, nil, 0, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_ZSET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	if zsetVal, ok := c.db.dict[c.args[0]].val.(*GodisZset); ok {
		score, ok := zsetVal.dict[c.args[1]]
		log.Printf("Debug 1 : %v\n", ok)
		log.Printf("Debug 1 : %v\n", score)
		if !ok {
			genReply(c, RE_NONE, nil, 0, nil)
			return
		}
		rank := zsetVal.zskiplist.spGetRank(score, CreateObj(GODIS_STRING, c.args[1]))
		if rank == 0 {
			log.Printf("command %s error\n", c.command)
		}
		genReply(c, RE_INT, nil, rank-1, nil)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

func zremCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_INT, nil, 0, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_ZSET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	var count int = 0
	if zsetVal, ok := c.db.dict[c.args[0]].val.(*GodisZset); ok {
		for i := 1; i < c.arg_count; i++ {
			if score, ok := zsetVal.dict[c.args[i]]; ok {
				zsetVal.zskiplist.spDelete(score, CreateObj(GODIS_STRING, c.args[i]))
				delete(zsetVal.dict, c.args[i])
				count++
			}
		}
		genReply(c, RE_INT, nil, count, nil)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}

}

func zscoreCommand(c *GodisClient) {
	if err := checkArgsCount(c); err != nil {
		return
	}
	checkDel(c, c.args[0])
	if _, ok := c.db.dict[c.args[0]]; !ok {
		genReply(c, RE_INT, nil, 0, nil)
	} else {
		if c.db.dict[c.args[0]].obj_type != GODIS_ZSET {
			genReply(c, RE_ERR, &str_err_wrongtype, 0, nil)
			return
		}
	}
	if zsetVal, ok := c.db.dict[c.args[0]].val.(*GodisZset); ok {
		score, ok := zsetVal.dict[c.args[1]]
		if !ok {
			genReply(c, RE_NONE, nil, 0, nil)
			return
		}

		s := fmt.Sprintf("%.2f", score)

		genReply(c, RE_FLOAT, &s, 0, nil)
	} else {
		log.Printf("command %s error\n", c.command)
		return
	}
}

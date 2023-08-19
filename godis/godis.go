package godis

import (
	myProto "godisdb/proto"
	"log"
	"math/rand"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"
)

type GodisDB struct {
	dict    map[string]*GodisObj
	expires map[string]int
}

type GodisClient struct {
	fd               int
	db               *GodisDB
	db_id            int
	name             string
	arg_count        int
	command          string
	args             []string
	reply            []string
	ctime            int
	last_interaction int
	read_buf         [1024]byte
}

type GodisServer struct {
	fd                    int
	ip                    string
	port                  int
	loop                  *AeEventLoop
	db_count              int
	clients               map[int]*GodisClient
	db                    map[int]*GodisDB
	expire_check_count    int
	expire_check_interval int
}

func findExpiredKey(loop *AeEventLoop, fd int, extra interface{}) int {
	for i := 0; i < server.db_count; i++ {
		now := GetMsTime()
		if len(server.db[i].expires) == 0 {
			return server.expire_check_interval
		}
		keys := make([]string, 0, len(server.db[i].expires))
		for k := range server.db[i].expires {
			keys = append(keys, k)
		}
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		for j := 0; j < server.expire_check_count; j++ {
			randomKey := keys[r.Intn(len(keys))]
			v, ok := server.db[i].expires[randomKey]
			if ok && v < now {
				delete(server.db[i].expires, randomKey)
				delete(server.db[i].dict, randomKey)
			}
		}
	}

	return server.expire_check_interval
}

func replyToClient(loop *AeEventLoop, fd int, mask FileEventType, extra interface{}) {
	if len(server.clients[fd].reply) == 0 {
		loop.AeDeleteFileEvent(fd, AE_WRITABLE, nil)
		return
	}
	reply := &myProto.Reply{
		Args: server.clients[fd].reply,
	}
	data, err := proto.Marshal(reply)
	if err != nil {
		log.Printf("replyToClient proto error: %v\n", err)
		return
	}
	_, err = unix.Write(fd, data)
	if err != nil {
		log.Printf("replyToClient Write error: %v\n", err)
		return
	}
	server.clients[fd].reply = []string{}
}

func processClientCommand(c *GodisClient) error {
	c.last_interaction = GetMsTime()
	cmd, ok := CommandTable[c.command]
	if !ok {
		c.reply = append(c.reply, "(error) ERR unknown command")
		server.loop.AeCreateFileEvent(c.fd, AE_WRITABLE, replyToClient, nil)
		return nil
	}
	cmd.proc(c)
	server.loop.AeCreateFileEvent(c.fd, AE_WRITABLE, replyToClient, nil)
	return nil
}

func readClient(loop *AeEventLoop, fd int, mask FileEventType, extra interface{}) {
	n, err := unix.Read(fd, server.clients[fd].read_buf[:])
	if err != nil {
		log.Printf("readClient read error: %v\n", err)
		return
	}
	if n == 0 {
		return
	}
	var client_cmd myProto.Cmd
	err = proto.Unmarshal(server.clients[fd].read_buf[:n], &client_cmd)
	if err != nil {
		log.Printf("readClient proto error: %v\n", err)
		return
	}
	log.Printf("recv: %v\n", &client_cmd)
	server.clients[fd].command = strings.ToLower(client_cmd.Command)
	server.clients[fd].arg_count = len(client_cmd.Args)
	server.clients[fd].args = client_cmd.GetArgs()
	err = processClientCommand(server.clients[fd])
	if err != nil {
		log.Printf("readClient process error: %v\n", err)
	}
}

func createClient(fd int) *GodisClient {
	client := &GodisClient{
		fd:               fd,
		db:               server.db[0],
		db_id:            0,
		name:             "",
		arg_count:        0,
		command:          "",
		args:             []string{},
		reply:            []string{},
		ctime:            GetMsTime(),
		last_interaction: GetMsTime(),
		read_buf:         [1024]byte{},
	}
	return client
}

func handleClient(loop *AeEventLoop, fd int, mask FileEventType, extra interface{}) {
	nfd, _, err := unix.Accept(fd)
	if err != nil {
		log.Printf("handleClient-Accept err: %v\n", err)
		return
	}
	server.clients[nfd] = createClient(nfd)
	server.loop.AeCreateFileEvent(nfd, AE_READABLE, readClient, nil)

}

func initServerConfig() {
	server = &GodisServer{
		ip:                    "127.0.0.1",
		port:                  9736,
		db_count:              10,
		expire_check_count:    10,
		expire_check_interval: 100,
	}
}

func initServer() {
	//server db
	server.db = make(map[int]*GodisDB)
	for i := 0; i < server.db_count; i++ {
		server.db[i] = &GodisDB{
			dict:    make(map[string]*GodisObj),
			expires: make(map[string]int),
		}
	}
	//server client
	server.clients = make(map[int]*GodisClient)

	//server fd
	listen, err := TcpSocket(server.ip, server.port)
	if err != nil {
		panic(err)
	}
	server.fd = listen

	//aeloop
	lp, err := AeCreateEventLoop()
	if err != nil {
		panic(err)
	}
	server.loop = lp
	server.loop.AeCreateFileEvent(listen, AE_READABLE, handleClient, nil)
	server.loop.AeCreateTimeEvent(0, AE_NORMAL, findExpiredKey, nil)

}

var server *GodisServer = nil

func Run() {
	initServerConfig()
	//TODO :read config from file
	initServer()

	server.loop.AeMain()
}

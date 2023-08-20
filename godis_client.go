package main

import (
	"fmt"
	"godisdb/godis"
	myProto "godisdb/proto"
	"net"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"google.golang.org/protobuf/proto"
)

func main() {
	// connect to godisdb
	path := os.Args[1]
	conn, err := net.Dial("tcp", path)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer conn.Close()

	// readline
	fmt.Println("godis-client")
	var historyFile = "/tmp/readline.tmp"

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[31m»\033[0m ",
		HistoryFile:     historyFile,
		AutoComplete:    nil,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})

	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}

		line = strings.TrimSuffix(line, "\n")

		if line == "exit" {
			break
		}

		err = sendCommand(line, conn)
		if err != nil {
			break
		}
		recvReply(conn)
	}

}

func sendCommand(command string, conn net.Conn) error {

	slice := strings.Fields(command)
	if len(slice) == 0 {
		fmt.Println(">")
	} else if len(slice) == 1 {
		var cmd = &myProto.Cmd{
			Command: slice[0],
		}
		data, _ := proto.Marshal(cmd)

		_, err := conn.Write(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
	} else {
		var cmd = &myProto.Cmd{
			Command: slice[0],
			Args:    slice[1:],
		}
		data, _ := proto.Marshal(cmd)
		_, err := conn.Write(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
	}
	return nil
}

func recvReply(conn net.Conn) {
	var buf [1024]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var reply myProto.Reply
	err = proto.Unmarshal(buf[:n], &reply)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	switch reply.ReplyType {
	case int64(godis.RE_NONE):
		fmt.Print("(nil)\n")
	case int64(godis.RE_OK):
		for _, value := range reply.Args {
			fmt.Println(value)
		}
	case int64(godis.RE_INT):
		fmt.Print("(integer) ")
		for _, value := range reply.Args {
			fmt.Println(value)
		}
	case int64(godis.RE_ERR):
		fmt.Print("(error) ")
		for _, value := range reply.Args {
			fmt.Println(value)
		}
	case int64(godis.RE_STRING):
		fmt.Printf("\"%s\"\n", reply.Args[0])
	case int64(godis.RE_HASH):
		if len(reply.Args) == 0 {
			fmt.Println("(empty array)")
		} else {
			for i := 0; i < len(reply.Args); i++ {
				fmt.Printf("%d) \" %s\"\n", i+1, reply.Args[i])
			}
		}
	case int64(godis.RE_LIST):
		if len(reply.Args) == 0 {
			fmt.Println("(empty array)")
		} else {
			for i := 0; i < len(reply.Args); i++ {
				fmt.Printf("%d) \" %s\"\n", i+1, reply.Args[i])
			}
		}
	case int64(godis.RE_SET):
		if len(reply.Args) == 0 {
			fmt.Println("(empty array)")
		} else {
			for i := 0; i < len(reply.Args); i++ {
				fmt.Printf("%d) \" %s\"\n", i+1, reply.Args[i])
			}
		}
	case int64(godis.RE_ZSET):
		if len(reply.Args) == 0 {
			fmt.Println("(empty array)")
		} else {
			for i := 0; i < len(reply.Args); i++ {
				fmt.Printf("%d) \" %s\"\n", i+1, reply.Args[i])
			}
		}
	case int64(godis.RE_FLOAT):

		fmt.Printf("\"%s\"\n", reply.Args[0])

	default:
		for _, value := range reply.Args {
			fmt.Println(value)
		}
	}

}

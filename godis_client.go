package main

import (
	"fmt"
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

		sendCommand(line, conn)
		recvReply(conn)
	}

}

func sendCommand(command string, conn net.Conn) {

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
			return
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
			return
		}
	}
}

func recvReply(conn net.Conn) {
	var buf [1024]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var reply myProto.Reply
	proto.Unmarshal(buf[:n], &reply)

	for _, value := range reply.Args {
		fmt.Print(value)
	}

}

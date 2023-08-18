package main

import (
	"bufio"
	"fmt"
	myProto "godisdb/proto"
	"net"
	"os"

	"google.golang.org/protobuf/proto"
)

func main() {
	listen, err := net.Listen("tcp", "127.0.0.1:9736")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	conn, err := listen.Accept()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for {
		reader := bufio.NewReader(conn)
		var buf [128]byte
		n, err := reader.Read(buf[:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}
		var client_cmd myProto.Cmd
		err = proto.Unmarshal(buf[:n], &client_cmd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}

		fmt.Println("recv:", &client_cmd)

		reply := &myProto.Reply{
			Args: []string{"ok\n"},
		}
		data, err := proto.Marshal(reply)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}
		_, err = conn.Write(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}
	}

	conn.Close()

}

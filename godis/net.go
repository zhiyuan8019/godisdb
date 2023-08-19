package godis

import (
	"net"

	"golang.org/x/sys/unix"
)

const BACKLOG int = 511

func TcpSocket(ipAddr string, port int) (int, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)
	if err != nil {
		return -1, err
	}
	err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, port)
	if err != nil {
		unix.Close(fd)
		return -1, err
	}

	var addr [4]byte
	copy(addr[:], net.ParseIP(ipAddr).To4())
	err = unix.Bind(fd, &unix.SockaddrInet4{
		Addr: addr,
		Port: port,
	})
	if err != nil {
		unix.Close(fd)
		return -1, err
	}
	err = unix.Listen(fd, BACKLOG)
	if err != nil {
		unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

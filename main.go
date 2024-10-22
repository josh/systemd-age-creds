package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	ln := activationListener()

	// _, ok := ln.Addr().(*net.UnixAddr)
	// if !ok {
	// 	panic("server must bind to a unix addr")
	// }

	conn, err := ln.Accept()
	if err != nil {
		panic(err)
	}

	handleConnection(conn)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	unixAddr, ok := conn.RemoteAddr().(*net.UnixAddr)
	if !ok {
		log.Printf("client must be a unix addr")
		return
	}

	unitName, credID, err := parsePeerName(unixAddr.Name)
	if err != nil {
		log.Printf("Failed to parse peer name: %s", unixAddr.Name)
		return
	}
	log.Printf("%s requesting '%s' credential", unitName, credID)

	// TODO: Decrypt actual secret
	_, err = conn.Write([]byte("42\n"))
	if err != nil {
		log.Printf("Failed to write credential: %v", err)
		return
	}
}

func parsePeerName(s string) (string, string, error) {
	matches := regexp.MustCompile("^@.*/unit/(.*)/(.*)$").FindStringSubmatch(s)
	if matches == nil {
		return "", "", fmt.Errorf("failed to parse peer name: %s", s)
	}
	return matches[1], matches[2], nil
}

func activationListener() net.Listener {
	defer os.Unsetenv("LISTEN_PID")
	defer os.Unsetenv("LISTEN_FDS")
	defer os.Unsetenv("LISTEN_FDNAMES")

	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
	if err != nil || pid != os.Getpid() {
		panic("LISTEN_PID for someone else")
	}

	nfds, err := strconv.Atoi(os.Getenv("LISTEN_FDS"))
	if err != nil || nfds == 0 {
		panic("LISTEN_FDS not set")
	}

	names := strings.Split(os.Getenv("LISTEN_FDNAMES"), ":")
	if len(names) != nfds {
		panic("LISTEN_FDNAMES count should match LISTEN_FDS")
	}

	listeners := make([]net.Listener, nfds)

	for i := 0; i < nfds; i++ {
		fd := i + 3
		syscall.CloseOnExec(fd)

		f := os.NewFile(uintptr(fd), names[i])
		if f == nil {
			panic("bad file descriptor")
		}
		pc, err := net.FileListener(f)
		if err != nil {
			panic(err)
		}

		listeners[i] = pc
		f.Close()
	}

	if len(listeners) != 1 {
		panic("Unexpected number of socket activation fds")
	}
	return listeners[0]
}

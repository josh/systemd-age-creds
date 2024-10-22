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
	ln, err := activationListener()
	if err != nil {
		panic(err)
	}

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

func activationListener() (*net.UnixListener, error) {
	defer os.Unsetenv("LISTEN_PID")
	defer os.Unsetenv("LISTEN_FDS_START")
	defer os.Unsetenv("LISTEN_FDS")
	defer os.Unsetenv("LISTEN_FDNAMES")

	if os.Getenv("LISTEN_PID") == "" {
		return nil, fmt.Errorf("expected LISTEN_PID=%d", os.Getpid())
	}
	if os.Getenv("LISTEN_FDS") == "" {
		return nil, fmt.Errorf("expected LISTEN_FDS=1")
	}
	if os.Getenv("LISTEN_FDNAMES") == "" {
		return nil, fmt.Errorf("expected LISTEN_FDNAMES=foo.sock")
	}

	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
	if err != nil || pid != os.Getpid() {
		return nil, fmt.Errorf("expected LISTEN_PID=%d, but was '%s'", os.Getpid(), os.Getenv("LISTEN_PID"))
	}

	fd := 3
	if os.Getenv("LISTEN_FDS_START") != "" {
		if fd, err = strconv.Atoi(os.Getenv("LISTEN_FDS_START")); err != nil {
			return nil, fmt.Errorf("expected LISTEN_FDS_START to be a int, but was '%s'", os.Getenv("LISTEN_FDS_START"))
		}
	}

	nfds, err := strconv.Atoi(os.Getenv("LISTEN_FDS"))
	if err != nil || nfds != 1 {
		return nil, fmt.Errorf("expected LISTEN_FDS=1, but was '%s'", os.Getenv("LISTEN_FDS"))
	}

	names := strings.Split(os.Getenv("LISTEN_FDNAMES"), ":")
	if len(names) != 1 {
		return nil, fmt.Errorf("expected LISTEN_FDNAMES to set 1 name, but was '%s'", os.Getenv("LISTEN_FDNAMES"))
	}
	name := names[0]

	syscall.CloseOnExec(fd)
	f := os.NewFile(uintptr(fd), name)

	l, err := net.FileListener(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}
	f.Close()

	unixListener, ok := l.(*net.UnixListener)
	if !ok {
		return nil, fmt.Errorf("must be a unix socket")
	}

	return unixListener, nil
}

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
	listeners, err := Listeners()
	if err != nil {
		panic(err)
	}

	if len(listeners) != 1 {
		panic("Unexpected number of socket activation fds")
	}
	ln := listeners[0]

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
		log.Printf("Unexpected remote address type: %T", conn.RemoteAddr())
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

func Listeners() ([]net.Listener, error) {
	files := Files()
	listeners := make([]net.Listener, len(files))

	for i, f := range files {
		if pc, err := net.FileListener(f); err == nil {
			listeners[i] = pc
			f.Close()
		}
	}
	return listeners, nil
}

func Files() []*os.File {
	defer os.Unsetenv("LISTEN_PID")
	defer os.Unsetenv("LISTEN_FDS")
	defer os.Unsetenv("LISTEN_FDNAMES")

	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
	if err != nil || pid != os.Getpid() {
		return nil
	}

	nfds, err := strconv.Atoi(os.Getenv("LISTEN_FDS"))
	if err != nil || nfds == 0 {
		return nil
	}

	names := strings.Split(os.Getenv("LISTEN_FDNAMES"), ":")

	const listenFdsStart = 3

	files := make([]*os.File, 0, nfds)
	for fd := listenFdsStart; fd < listenFdsStart+nfds; fd++ {
		syscall.CloseOnExec(fd)
		name := "LISTEN_FD_" + strconv.Itoa(fd)
		offset := fd - listenFdsStart
		if offset < len(names) && len(names[offset]) > 0 {
			name = names[offset]
		}
		files = append(files, os.NewFile(uintptr(fd), name))
	}

	return files
}

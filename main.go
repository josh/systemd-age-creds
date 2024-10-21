package main

import (
	"fmt"
	"log"
	"net"
	"regexp"

	"github.com/coreos/go-systemd/v22/activation"
)

func main() {
	listeners, err := activation.Listeners()
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

//go:build linux

package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParsePeerNameOk(t *testing.T) {
	t.Parallel()

	unit, cred, err := parsePeerName("@f4b4692a71d9438e/unit/age-creds-test.service/foo")
	if err != nil {
		t.Error(err)
	}

	if unit != "age-creds-test.service" {
		t.Errorf("unit = %s; want age-creds-test.service", unit)
	}

	if cred != "foo" {
		t.Errorf("cred = %s; want foo", cred)
	}
}

func TestParsePeerNameBlank(t *testing.T) {
	t.Parallel()

	_, _, err := parsePeerName("")
	if err == nil {
		t.Error("expected parse error")
	}
}

//nolint:paralleltest
func TestActivationListener(t *testing.T) {
	socketPath := t.TempDir() + "/foo.sock"

	ln1, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Error(err)
	}
	defer ln1.Close()

	f1, err := ln1.(*net.UnixListener).File()
	if err != nil {
		t.Error(err)
	}

	//nolint:exhaustruct
	opts := &options{
		ListenPID:      os.Getpid(),
		ListenFDs:      1,
		ListenFDNames:  "foo.sock",
		ListenFDsStart: int(f1.Fd()),
	}

	ln2, err := activationListener(opts)
	if err != nil {
		t.Error(err)
	}

	defer ln2.Close()

	f2, err := ln2.File()
	if err != nil {
		t.Error(err)
	}

	if f1.Fd() != f2.Fd() {
		t.Errorf("fd = %d; want %d", f2.Fd(), f1.Fd())
	}
}

//nolint:paralleltest
func TestStartAccept(t *testing.T) {
	sname := t.TempDir() + "/connection"
	saddr := &net.UnixAddr{Name: sname, Net: "unix"}

	ln, err := net.ListenUnix("unix", saddr)
	if err != nil {
		t.Error(err)
		return
	}
	defer ln.Close()

	go func() {
		_, err := readCred("foo", sname)
		if err != nil {
			t.Error(err)
		}
	}()

	err = ln.SetDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		t.Error(err)
		return
	}

	conn, err := ln.AcceptUnix()
	if err != nil {
		t.Error(err)
		return
	}

	defer conn.Close()

	f, err := conn.File()
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()

	//nolint:exhaustruct
	opts := &options{
		Dir:            testDir(),
		Accept:         true,
		ListenPID:      os.Getpid(),
		ListenFDs:      1,
		ListenFDNames:  "connection",
		ListenFDsStart: int(f.Fd()),
	}

	err = start(opts)
	if err != nil {
		t.Error(err)
	}
}

func testDir() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "test")
}

var errReadCred = errors.New("could not read cred")

func readCred(credID string, socketPath string) (string, error) {
	lname := "@f4b4692a71d9438e/unit/test.service/" + credID
	laddr := &net.UnixAddr{Name: lname, Net: "unix"}
	raddr := &net.UnixAddr{Name: socketPath, Net: "unix"}

	conn, err := net.DialUnix("unix", laddr, raddr)
	if err != nil {
		return "", fmt.Errorf("%w: dail error: %w", errReadCred, err)
	}
	defer conn.Close()

	buf, err := io.ReadAll(conn)
	if err != nil {
		return "", fmt.Errorf("%w: read err: %w", errReadCred, err)
	}

	if len(buf) == 0 {
		return "", fmt.Errorf("%w: zero bytes", errReadCred)
	}

	return string(buf), nil
}

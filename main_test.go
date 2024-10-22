package main

import (
	"fmt"
	"net"
	"os"
	"testing"
)

func TestParsePeerNameOk(t *testing.T) {
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
	_, _, err := parsePeerName("")
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestActivationListener(t *testing.T) {
	socketPath := fmt.Sprintf("%s/foo.sock", t.TempDir())
	ln1, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Error(err)
	}
	defer ln1.Close()

	f1, err := ln1.(*net.UnixListener).File()
	if err != nil {
		t.Error(err)
	}

	t.Setenv("LISTEN_PID", fmt.Sprintf("%d", os.Getpid()))
	t.Setenv("LISTEN_FDS", "1")
	t.Setenv("LISTEN_FDNAMES", "foo.sock")
	t.Setenv("LISTEN_FDS_START", fmt.Sprintf("%d", f1.Fd()))
	ln2, err := activationListener()

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

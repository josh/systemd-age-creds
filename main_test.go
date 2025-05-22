//go:build linux

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
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
	socketPath := t.TempDir() + "/foo.sock"

	ln1, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if err := ln1.Close(); err != nil {
			t.Errorf("failed to close listener: %v", err)
		}
	}()

	f1, err := ln1.(*net.UnixListener).File()
	if err != nil {
		t.Error(err)
	}

	opts, err := testOptions()
	if err != nil {
		t.Error(err)
		return
	}

	opts.Accept = false
	opts.ListenFDNames = "foo.sock"
	opts.ListenFDsStart = int(f1.Fd())

	ln2, err := activationListener(opts)
	if err != nil {
		t.Error(err)
	}

	defer func() {
		if err := ln2.Close(); err != nil {
			t.Errorf("failed to close listener: %v", err)
		}
	}()

	f2, err := ln2.File()
	if err != nil {
		t.Error(err)
	}

	if f1.Fd() != f2.Fd() {
		t.Errorf("fd = %d; want %d", f2.Fd(), f1.Fd())
	}
}

func TestStartAccept(t *testing.T) {
	sname := t.TempDir() + "/connection"
	saddr := &net.UnixAddr{Name: sname, Net: "unix"}

	ln, err := net.ListenUnix("unix", saddr)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		if err := ln.Close(); err != nil {
			t.Errorf("failed to close listener: %v", err)
		}
	}()

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

	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("failed to close connection: %v", err)
		}
	}()

	f, err := conn.File()
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("failed to close file: %v", err)
		}
	}()

	opts, err := testOptions()
	if err != nil {
		t.Error(err)
		return
	}

	opts.Accept = true
	opts.ListenFDNames = "connection"
	opts.ListenFDsStart = int(f.Fd())

	err = start(opts)
	if err != nil {
		t.Error(err)
	}
}

func TestStartAcceptWrongIdentity(t *testing.T) {
	sname := t.TempDir() + "/connection"
	saddr := &net.UnixAddr{Name: sname, Net: "unix"}

	ln, err := net.ListenUnix("unix", saddr)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		if err := ln.Close(); err != nil {
			t.Errorf("failed to close listener: %v", err)
		}
	}()

	go func() {
		_, err := readCred("foo", sname)
		if err == nil {
			t.Errorf("expected readCred to fail, but was ok")
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

	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("failed to close connection: %v", err)
		}
	}()

	f, err := conn.File()
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("failed to close file: %v", err)
		}
	}()

	opts, err := testOptions()
	if err != nil {
		t.Error(err)
		return
	}

	opts.Identity = filepath.Join(opts.Dir, "test", "key2.txt")
	opts.Accept = true
	opts.ListenFDNames = "connection"
	opts.ListenFDsStart = int(f.Fd())

	err = start(opts)
	if err == nil {
		t.Errorf("expected server to fail to decrypt cred, but was ok")
	}
}

func testOptions() (*options, error) {
	ageBin, err := exec.LookPath("age")
	if err != nil {
		return nil, err
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	opts := options{
		AgeBin:         ageBin,
		Identity:       filepath.Join(wd, "test", "key.txt"),
		Dir:            filepath.Join(wd, "test"),
		Accept:         false,
		ListenPID:      os.Getpid(),
		ListenFDs:      1,
		ListenFDNames:  "foo.sock",
		ListenFDsStart: 3,
		ShowVersion:    false,
	}

	return &opts, nil
}

func readCred(credID string, socketPath string) (string, error) {
	lname := "@f4b4692a71d9438e/unit/test.service/" + credID
	laddr := &net.UnixAddr{Name: lname, Net: "unix"}
	raddr := &net.UnixAddr{Name: socketPath, Net: "unix"}

	conn, err := net.DialUnix("unix", laddr, raddr)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Printf("Warning: failed to close connection: %v\n", err)
		}
	}()

	buf, err := io.ReadAll(conn)
	if err != nil {
		return "", err
	}

	if len(buf) == 0 {
		return "", fmt.Errorf("read zero bytes")
	}

	return string(buf), nil
}

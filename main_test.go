//go:build linux

package main

import (
	"context"
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
	unit, cred, err := parsePeerName("@f4b4692a71d9438e/unit/systemd-age-creds-test.service/foo")
	if err != nil {
		t.Error(err)
	}

	if unit != "systemd-age-creds-test.service" {
		t.Errorf("unit = %s; want systemd-age-creds-test.service", unit)
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

	err = start(t.Context(), opts)
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

	err = start(t.Context(), opts)
	if err == nil {
		t.Errorf("expected server to fail to decrypt cred, but was ok")
	}
}

func TestIdleTimeoutGracefulExit(t *testing.T) {
	socketPath := t.TempDir() + "/idle.sock"
	saddr := &net.UnixAddr{Name: socketPath, Net: "unix"}

	ln, err := net.ListenUnix("unix", saddr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = ln.Close()
	}()

	f, err := ln.File()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	opts, err := testOptions()
	if err != nil {
		t.Fatal(err)
	}
	opts.Accept = false
	opts.ListenFDNames = "idle.sock"
	opts.ListenFDsStart = int(f.Fd())
	opts.IdleTimeout = 5 * time.Second

	ctx := t.Context()

	done := make(chan error, 1)
	go func() {
		done <- start(ctx, opts)
	}()

	time.Sleep(500 * time.Millisecond)

	conn, err := net.DialUnix("unix", nil, saddr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	_ = conn.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("server exited with error: %v", err)
		}
	case <-time.After(7 * time.Second):
		t.Error("server did not exit after idle timeout")
	}
}

func TestReadPeercred(t *testing.T) {
	l, err := net.Listen("unix", "@peercred-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("listener close error: %v", err)
		}
	}()

	ctx := t.Context()

	go func(ctx context.Context) {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		<-ctx.Done()
		if err := conn.Close(); err != nil {
			t.Errorf("conn close error: %v", err)
		}
	}(ctx)

	conn, err := net.Dial("unix", "@peercred-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("conn close error: %v", err)
		}
	}()

	cred, err := readPeercred(conn.(*net.UnixConn))
	if err != nil {
		t.Fatalf("want nil err, got %v", err)
	}

	pid, uid, gid := os.Getpid(), os.Getuid(), os.Getgid()
	if cred.Pid != int32(pid) {
		t.Errorf("pid: want %d, got %d", pid, cred.Pid)
	}
	if cred.Uid != uint32(uid) {
		t.Errorf("uid: want %d, got %d", uid, cred.Uid)
	}
	if cred.Gid != uint32(gid) {
		t.Errorf("gid: want %d, got %d", gid, cred.Gid)
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

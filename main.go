//go:build linux

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// constants settable at build time
var (
	AgeBin         = ""
	AgeDir         = ""
	AgeIdentity    = ""
	ListenFDsStart = 3
	Version        = "0.0.0"
)

type options struct {
	AgeBin         string
	Accept         bool
	Dir            string
	Identity       string
	ListenFDNames  string
	ListenFDs      int
	ListenFDsStart int
	ListenPID      int
	ShowVersion    bool
}

func parseFlags(progname string, args []string, out io.Writer) (*options, error) {
	var opts options

	defaultAgeBin := AgeBin
	if defaultAgeBin == "" {
		if path, err := exec.LookPath("age"); err == nil {
			defaultAgeBin = path
		}
	}

	fs := flag.NewFlagSet(progname, flag.ContinueOnError)
	fs.StringVar(&opts.AgeBin, "age-bin", defaultAgeBin, "path to age binary")
	fs.BoolVar(&opts.Accept, "accept", false, "assume connection already accepted")
	fs.StringVar(&opts.Dir, "dir", AgeDir, "directory to store credentials in")
	fs.StringVar(&opts.Identity, "identity", AgeIdentity, "age identity file")
	fs.StringVar(&opts.ListenFDNames, "listen-fdnames", "", "intended LISTEN_FDNAMES")
	fs.IntVar(&opts.ListenFDs, "listen-fds", 0, "intended number of LISTEN_FDS")
	fs.IntVar(&opts.ListenFDsStart, "listen-fds-start", ListenFDsStart, "intended start of LISTEN_FDS")
	fs.IntVar(&opts.ListenPID, "listen-pid", 0, "intended PID of listener")
	fs.BoolVar(&opts.ShowVersion, "version", false, "print version and exit")

	envFlags := map[string]string{
		"AGE_BIN":          "age-bin",
		"AGE_DIR":          "dir",
		"AGE_IDENTITY":     "identity",
		"LISTEN_PID":       "listen-pid",
		"LISTEN_FDS_START": "listen-fds-start",
		"LISTEN_FDS":       "listen-fds",
		"LISTEN_FDNAMES":   "listen-fdnames",
	}

	for envName, flagName := range envFlags {
		if val, ok := os.LookupEnv(envName); ok {
			if err := fs.Set(flagName, val); err != nil {
				fs.Usage()
				return &opts, fmt.Errorf("invalid value \"%s\" for flag -%s: %w", val, flagName, err)
			}

			os.Unsetenv(envName)
		}
	}

	fs.SetOutput(out)

	err := fs.Parse(args)
	if err != nil {
		return &opts, err
	}

	if opts.ShowVersion {
		return &opts, nil
	}

	if opts.Dir == "" {
		fs.Usage()
		return &opts, errors.New("missing credentials directory")
	}

	if opts.Identity == "" {
		fs.Usage()
		return &opts, errors.New("missing age identity file")
	}

	if opts.ListenFDNames == "connection" {
		opts.Accept = true
	}

	return &opts, nil
}

func main() {
	opts, err := parseFlags(os.Args[0], os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	} else if err != nil {
		fmt.Printf("%v", err)
		os.Exit(2)
	}

	if opts.ShowVersion {
		fmt.Printf("systemd-age-creds %s\n", Version)
		os.Exit(0)
	}

	fmt.Println("Starting systemd-age-creds")
	defer fmt.Println("Stopping systemd-age-creds")

	err = start(opts)
	if err != nil {
		panic(err)
	}
}

func start(opts *options) error {
	if opts.Accept {
		return startConnection(opts)
	}

	return startListener(opts)
}

func startConnection(opts *options) error {
	conn, err := activationConnection(opts)
	if err != nil {
		return fmt.Errorf("failed to accept connection: %w", err)
	}

	return handleConnection(conn, opts)
}

func startListener(opts *options) error {
	ln, err := activationListener(opts)
	if err != nil {
		return err
	}
	defer ln.Close()

	fmt.Printf("Listening on %s\n", ln.Addr())

	for {
		conn, err := ln.AcceptUnix()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		go func(conn *net.UnixConn, opts *options) {
			err := handleConnection(conn, opts)
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
			}
		}(conn, opts)
	}
}

func handleConnection(conn *net.UnixConn, opts *options) error {
	defer conn.Close()

	unixAddr, ok := conn.RemoteAddr().(*net.UnixAddr)
	if !ok {
		panic("expected unix connection to return a unix addr")
	}

	unitName, credID, err := parsePeerName(unixAddr.Name)
	if err != nil {
		return err
	}

	fmt.Printf("%s requesting '%s' credential\n", unitName, credID)

	filename := credID + ".age"
	path := filepath.Join(opts.Dir, filename)

	data, err := ageDecrypt(opts, path)
	if err != nil {
		return fmt.Errorf("failed to decrypt credential file %s: %w", path, err)
	}

	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write credential: %w", err)
	}

	return nil
}

func ageDecrypt(opts *options, path string) ([]byte, error) {
	cmd := exec.Command(opts.AgeBin, "--decrypt", "--identity", opts.Identity, path)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("age failed to decrypt '%s': %w", path, err)
	}

	return stdout.Bytes(), nil
}

func parsePeerName(s string) (string, string, error) {
	matches := regexp.MustCompile("^@.*/unit/(.*)/(.*)$").FindStringSubmatch(s)
	if matches == nil {
		return "", "", fmt.Errorf("invalid peer name: %s", s)
	}

	return matches[1], matches[2], nil
}

func activationFile(opts *options) (*os.File, error) {
	if opts.ListenPID != os.Getpid() {
		return nil, fmt.Errorf("expected LISTEN_PID=%d, but was %d", os.Getpid(), opts.ListenPID)
	}

	fd := opts.ListenFDsStart

	if opts.ListenFDs != 1 {
		return nil, fmt.Errorf("expected LISTEN_FDS=1, but was %d", opts.ListenFDs)
	}

	names := strings.Split(opts.ListenFDNames, ":")
	if len(names) != 1 {
		return nil, fmt.Errorf("expected LISTEN_FDNAMES to set 1 name, but was '%s'", opts.ListenFDNames)
	}

	name := names[0]

	syscall.CloseOnExec(fd)
	f := os.NewFile(uintptr(fd), name)

	return f, nil
}

func activationListener(opts *options) (*net.UnixListener, error) {
	f, err := activationFile(opts)
	if err != nil {
		return nil, err
	}

	l, err := net.FileListener(f)
	if err != nil {
		return nil, err
	}

	f.Close()

	unixListener, ok := l.(*net.UnixListener)
	if !ok {
		return nil, errors.New("must be a unix socket")
	}

	return unixListener, nil
}

func activationConnection(opts *options) (*net.UnixConn, error) {
	f, err := activationFile(opts)
	if err != nil {
		return nil, err
	}

	conn, err := net.FileConn(f)
	if err != nil {
		return nil, err
	}

	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return nil, errors.New("must be a unix socket")
	}

	return unixConn, nil
}

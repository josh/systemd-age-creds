//go:build linux

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

var (
	AGE_BIN          = ""
	AGE_DIR          = ""
	AGE_IDENTITY     = ""
	LISTEN_FDS_START = 3
)

var Version = "0.0.0"

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

	if AGE_BIN == "" {
		if path, err := exec.LookPath("age"); err == nil {
			AGE_BIN = path
		}
	}

	fs := flag.NewFlagSet(progname, flag.ContinueOnError)
	fs.StringVar(&opts.AgeBin, "age-bin", AGE_BIN, "path to age binary")
	fs.BoolVar(&opts.Accept, "accept", false, "assume connection already accepted")
	fs.StringVar(&opts.Dir, "dir", AGE_DIR, "directory to store credentials in")
	fs.StringVar(&opts.Identity, "identity", AGE_IDENTITY, "age identity file")
	fs.StringVar(&opts.ListenFDNames, "listen-fdnames", "", "intended LISTEN_FDNAMES")
	fs.IntVar(&opts.ListenFDs, "listen-fds", 0, "intended number of LISTEN_FDS")
	fs.IntVar(&opts.ListenFDsStart, "listen-fds-start", LISTEN_FDS_START, "intended start of LISTEN_FDS")
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
		return &opts, fmt.Errorf("argument error: %w", err)
	}

	if opts.ShowVersion {
		return &opts, nil
	}

	if opts.Dir == "" {
		fs.Usage()
		return &opts, errMissingCredentialsDir
	}

	if opts.Identity == "" {
		fs.Usage()
		return &opts, errMissingAgeIdentityFile
	}

	if opts.ListenFDNames == "connection" {
		opts.Accept = true
	}

	return &opts, nil
}

var (
	errInvalidPeerName        = errors.New("invalid peer name")
	errMissingAgeIdentityFile = errors.New("missing age identity file")
	errMissingCredentialsDir  = errors.New("missing credentials directory")
	errNotUnixSocket          = errors.New("must be a unix socket")
	errSocketActivation       = errors.New("socket activation error")
)

func main() {
	opts, err := parseFlags(os.Args[0], os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	} else if err != nil {
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

	return handleConnection(conn, opts.Dir)
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
			err := handleConnection(conn, opts.Dir)
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
			}
		}(conn, opts)
	}
}

func handleConnection(conn *net.UnixConn, directory string) error {
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
	path := filepath.Join(directory, filename)

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read credential file %s: %w", path, err)
	}

	// Write the content back to the connection
	_, err = conn.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write credential: %w", err)
	}

	return nil
}

func parsePeerName(s string) (string, string, error) {
	matches := regexp.MustCompile("^@.*/unit/(.*)/(.*)$").FindStringSubmatch(s)
	if matches == nil {
		return "", "", fmt.Errorf("%w: %s", errInvalidPeerName, s)
	}

	return matches[1], matches[2], nil
}

func activationFile(opts *options) (*os.File, error) {
	if opts.ListenPID != os.Getpid() {
		return nil, fmt.Errorf("%w: expected LISTEN_PID=%d, but was %d",
			errSocketActivation, os.Getpid(), opts.ListenPID)
	}

	fd := opts.ListenFDsStart

	if opts.ListenFDs != 1 {
		return nil, fmt.Errorf("%w: expected LISTEN_FDS=1, but was %d",
			errSocketActivation, opts.ListenFDs)
	}

	names := strings.Split(opts.ListenFDNames, ":")
	if len(names) != 1 {
		return nil, fmt.Errorf("%w: expected LISTEN_FDNAMES to set 1 name, but was '%s'",
			errSocketActivation, opts.ListenFDNames)
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
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	f.Close()

	unixListener, ok := l.(*net.UnixListener)
	if !ok {
		return nil, errNotUnixSocket
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
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return nil, errNotUnixSocket
	}

	return unixConn, nil
}

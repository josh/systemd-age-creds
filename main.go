package main

import (
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

var envFlags = map[string]string{
	"AGE_BIN":          "age-bin",
	"AGE_DIR":          "dir",
	"AGE_IDENTITY":     "identity",
	"LISTEN_PID":       "listen-pid",
	"LISTEN_FDS_START": "listen-fds-start",
	"LISTEN_FDS":       "listen-fds",
	"LISTEN_FDNAMES":   "listen-fdnames",
}

func parseFlags(progname string, args []string, out io.Writer) (*options, error) {
	var opts options

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

	for envName, flagName := range envFlags {
		if val, ok := os.LookupEnv(envName); ok {
			if err := fs.Set(flagName, val); err != nil {
				return nil, err
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
		return &opts, fmt.Errorf("missing credentials directory")
	}

	if opts.Identity == "" {
		fs.Usage()
		return &opts, fmt.Errorf("missing age identity file")
	}

	if opts.ListenFDNames == "connection" {
		opts.Accept = true
	}

	return &opts, nil
}

func main() {
	opts, err := parseFlags(os.Args[0], os.Args[1:], os.Stderr)
	if err == flag.ErrHelp {
		os.Exit(0)
	} else if err != nil {
		os.Exit(2)
	}

	if opts.ShowVersion {
		fmt.Printf("systemd-age-creds %s\n", Version)
		os.Exit(0)
	}

	fmt.Printf("Starting systemd-age-creds\n")

	if opts.Accept {
		conn, err := activationConnection(opts)
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			return
		}
		handleConnection(conn, opts.Dir)
	} else {
		ln, err := activationListener(opts)
		if err != nil {
			panic(err)
		}
		defer ln.Close()

		fmt.Printf("Listening on %s\n", ln.Addr())

		for {
			conn, err := ln.AcceptUnix()
			if err != nil {
				fmt.Printf("Failed to accept connection: %v\n", err)
				continue
			}
			go handleConnection(conn, opts.Dir)
		}
	}

	fmt.Printf("Stopping systemd-age-creds\n")
}

func handleConnection(conn *net.UnixConn, directory string) {
	defer conn.Close()

	unixAddr, ok := conn.RemoteAddr().(*net.UnixAddr)
	if !ok {
		panic("client must be a unix addr")
	}

	unitName, credID, err := parsePeerName(unixAddr.Name)
	if err != nil {
		fmt.Printf("Failed to parse peer name: %s\n", unixAddr.Name)
		return
	}
	fmt.Printf("%s requesting '%s' credential\n", unitName, credID)

	filename := fmt.Sprintf("%s.age", credID)
	path := filepath.Join(directory, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Failed to read credential file %s: %v\n", path, err)
		return
	}

	// Write the content back to the connection
	_, err = conn.Write(content)
	if err != nil {
		fmt.Printf("Failed to write credential: %v\n", err)
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
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}
	f.Close()

	unixListener, ok := l.(*net.UnixListener)
	if !ok {
		return nil, fmt.Errorf("must be a unix socket")
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
		return nil, fmt.Errorf("must be a unix socket")
	}

	return unixConn, nil
}

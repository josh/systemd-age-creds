package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

type Options struct {
	Accept         bool
	Dir            string
	ListenPID      int
	ListenFDsStart int
	ListenFDs      int
	ListenFDNames  string
}

func parseFlags(progname string, args []string, out io.Writer) (*Options, error) {
	defer os.Unsetenv("LISTEN_PID")
	defer os.Unsetenv("LISTEN_FDS_START")
	defer os.Unsetenv("LISTEN_FDS")
	defer os.Unsetenv("LISTEN_FDNAMES")

	fs := flag.NewFlagSet(progname, flag.ContinueOnError)

	var opts Options
	fs.BoolVar(&opts.Accept, "accept", false, "assume connection already accepted")
	fs.StringVar(&opts.Dir, "dir", "", "directory to store credentials in")
	fs.IntVar(&opts.ListenPID, "listen-pid", 0, "intended PID of listener")
	fs.IntVar(&opts.ListenFDsStart, "listen-fds-start", 3, "intended start of LISTEN_FDS")
	fs.IntVar(&opts.ListenFDs, "listen-fds", 0, "intended number of LISTEN_FDS")
	fs.StringVar(&opts.ListenFDNames, "listen-fdnames", "", "intended LISTEN_FDNAMES")

	if val, ok := os.LookupEnv("LISTEN_PID"); ok {
		fs.Set("listen-pid", val)
	}
	if val, ok := os.LookupEnv("LISTEN_FDS_START"); ok {
		fs.Set("listen-fds-start", val)
	}
	if val, ok := os.LookupEnv("LISTEN_FDS"); ok {
		fs.Set("listen-fds", val)
	}
	if val, ok := os.LookupEnv("LISTEN_FDNAMES"); ok {
		fs.Set("listen-fdnames", val)
		if val == "connection" {
			fs.Set("accept", "true")
		}
	}

	fs.SetOutput(out)
	err := fs.Parse(args)
	if err != nil {
		return &opts, err
	}

	if opts.Dir == "" && len(fs.Args()) > 0 {
		opts.Dir = fs.Args()[0]
	}

	if opts.Dir == "" {
		fs.Usage()
		return &opts, fmt.Errorf("missing credentials directory")
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

	fmt.Printf("Starting systemd-age-creds with directory: %s\n", opts.Dir)

	if opts.Accept {
		conn, err := activationConnection(opts)
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
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
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				continue
			}
			go handleConnection(conn, opts.Dir)
		}
	}
}

func handleConnection(conn net.Conn, directory string) {
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

	filename := fmt.Sprintf("%s.age", credID)
	path := filepath.Join(directory, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Failed to read credential file %s: %v", path, err)
		return
	}

	// Write the content back to the connection
	_, err = conn.Write(content)
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

func activationFile(opts *Options) (*os.File, error) {
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

func activationListener(opts *Options) (*net.UnixListener, error) {
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

func activationConnection(opts *Options) (*net.UnixConn, error) {
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

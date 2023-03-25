package main

import (
	"flag"
	"fmt"
	"github.com/cybwan/gadget/pkg/sockopt"
	"github.com/spf13/pflag"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/cybwan/gadget/pkg/log"
	"github.com/cybwan/gadget/pkg/proxy"
)

var (
	flags = pflag.NewFlagSet(`tcp-proxy`, pflag.ExitOnError)

	localAddr   string
	remoteAddr  string
	verbose     bool
	veryverbose bool
	nagles      bool
	hex         bool
	colors      bool
	unwrapTLS   bool
	match       string
	replace     string
)

var (
	matchid = uint64(0)
	connid  = uint64(0)

	logger log.ColorLogger
)

func init() {
	flags.StringVarP(&localAddr, "localAddr", "l", ":15001", "local address")
	flags.StringVarP(&remoteAddr, "remoteAddr", "r", "httpbin.org:80", "remote address")
	flags.BoolVarP(&verbose, "verbose", "v", false, "display server actions")
	flags.BoolVarP(&veryverbose, "veryverbose", "a", false, "display server actions and all tcp data")
	flags.BoolVarP(&nagles, "nagles", "n", false, "disable nagles algorithm")
	flags.BoolVarP(&hex, "hex", "h", false, "output hex")
	flags.BoolVarP(&colors, "colors", "c", false, "output ansi colors")
	flags.BoolVarP(&unwrapTLS, "unwrapTLS", "u", false, "remote connection with TLS exposed unencrypted locally")
	flags.StringVarP(&match, "match", "m", "", "match regex (in the form 'regex')")
	flags.StringVarP(&replace, "replace", "e", "", "replace regex (in the form 'regex~replacer')")
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

func main() {
	if err := parseFlags(); err != nil {
		os.Exit(1)
	}

	logger := log.ColorLogger{
		Verbose: verbose,
		Color:   colors,
	}

	logger.Info("tcp-proxy proxing from %v to %v ", localAddr, remoteAddr)

	laddr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		logger.Warn("Failed to resolve local address: %s", err)
		os.Exit(1)
	}
	raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		logger.Warn("Failed to resolve remote address: %s", err)
		os.Exit(1)
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		logger.Warn("Failed to open local port to listen: %s", err)
		os.Exit(1)
	}

	matcher := createMatcher(match)
	replacer := createReplacer(replace)

	if veryverbose {
		verbose = true
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			logger.Warn("Failed to accept connection '%s'", err)
			continue
		}

		orginalIp, orginalPort, _ := sockopt.GetOriginalDest(conn)
		logger.Warn("OrginalDest ******* %s:%d *******", orginalIp, orginalPort)

		connid++

		var p *proxy.Proxy
		if unwrapTLS {
			logger.Info("Unwrapping TLS")
			p = proxy.NewTLSUnwrapped(conn, laddr, raddr, remoteAddr)
		} else {
			p = proxy.New(conn, laddr, raddr)
		}

		p.Matcher = matcher
		p.Replacer = replacer

		p.Nagles = nagles
		p.OutputHex = hex
		p.Log = log.ColorLogger{
			Verbose:     verbose,
			VeryVerbose: veryverbose,
			Prefix:      fmt.Sprintf("Connection #%03d ", connid),
			Color:       colors,
		}

		go p.Start()
	}
}

func createMatcher(match string) func([]byte) {
	if match == "" {
		return nil
	}
	re, err := regexp.Compile(match)
	if err != nil {
		logger.Warn("Invalid match regex: %s", err)
		return nil
	}

	logger.Info("Matching %s", re.String())
	return func(input []byte) {
		ms := re.FindAll(input, -1)
		for _, m := range ms {
			matchid++
			logger.Info("Match #%d: %s", matchid, string(m))
		}
	}
}

func createReplacer(replace string) func([]byte) []byte {
	if replace == "" {
		return nil
	}
	//split by / (TODO: allow slash escapes)
	parts := strings.Split(replace, "~")
	if len(parts) != 2 {
		logger.Warn("Invalid replace option")
		return nil
	}

	re, err := regexp.Compile(string(parts[0]))
	if err != nil {
		logger.Warn("Invalid replace regex: %s", err)
		return nil
	}

	repl := []byte(parts[1])

	logger.Info("Replacing %s with %s", re.String(), repl)
	return func(input []byte) []byte {
		return re.ReplaceAll(input, repl)
	}
}

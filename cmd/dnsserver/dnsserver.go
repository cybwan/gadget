package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/spf13/pflag"

	"github.com/cybwan/gadget/pkg/dns"
)

var (
	flags = pflag.NewFlagSet(`dnsserver`, pflag.ExitOnError)

	addr string
	port uint16
)

func init() {
	flags.StringVar(&addr, "addr", "127.0.0.1", "addr")
	flags.Uint16Var(&port, "port", 15053, "port")
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
		log.Fatal("Error parsing cmd line arguments")
	}

	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(`%s:%d`, addr, port))

	if err != nil {
		log.Println("Error resolving UDP address: ", err.Error())
		os.Exit(1)
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)

	if err != nil {
		log.Println("Error listening: ", err.Error())
		os.Exit(1)
	}

	log.Println("Listening at: ", serverAddr)

	defer serverConn.Close()

	for {
		requestBytes := make([]byte, dns.UDPMaxMessageSizeBytes)

		_, clientAddr, err := serverConn.ReadFromUDP(requestBytes)

		if err != nil {
			log.Println("Error receiving: ", err.Error())
		} else {
			log.Println("Received request from ", clientAddr)
			go dns.HandleDNSClient(requestBytes, serverConn, clientAddr) // array is value type (call-by-value), i.e. copied
		}
	}
}

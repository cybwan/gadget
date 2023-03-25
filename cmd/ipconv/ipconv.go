package main

import (
	"flag"
	"fmt"
	"github.com/cybwan/gadget/pkg/ipconv"
	"github.com/spf13/pflag"
	"log"
	"os"
)

var (
	flags = pflag.NewFlagSet(`ipconv`, pflag.ExitOnError)
	addr  string
	num   uint32
)

func init() {
	flags.StringVarP(&addr, "addr", "a", "", "addr")
	flags.Uint32VarP(&num, "num", "n", 0, "num")
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

	if len(addr) > 0 {
		if ip, _, err := ipconv.ParseIP(addr); err == nil {
			if num, err := ipconv.IPv4ToInt(ip); err == nil {
				fmt.Println(fmt.Sprintf("%s -> %d", ip, num))
			} else {
				fmt.Println(err.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	if num > 0 {
		ip := ipconv.IntToIPv4(num)
		fmt.Println(fmt.Sprintf("%d -> %s", num, ip))
	}
}

package main

import (
	"os"

	"github.com/5000K/5000mails/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}

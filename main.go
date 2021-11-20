// This project isn't set up to be a complete application.
// It's designed to be a super trivial script.
package main

import (
	"fmt"
	"os"

	cmd "github.com/Eagerod/hostsfile-generator/cmd/hostsfile-generator"
)

func main() {
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}

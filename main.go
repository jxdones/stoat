package main

import (
	"os"

	"github.com/jxdones/stoat/cmd"
)

func main() {
	err := cmd.RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

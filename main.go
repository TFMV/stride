// Package main is the entry point of stride.
//
// This package contains the implementation of the `stride` command, which is a
// high-performance file walking utility that extends the standard `filepath.Walk`
// functionality with concurrency, filtering, and monitoring capabilities.
//
// The `stride` command supports various options for filtering files based on name,
// path, size, modification time, and more. It also provides functionality to execute
// commands for each matched file or format the output using templates.
//
// The `stride` command also includes a `find` subcommand that allows users to search
// for files in a given directory with advanced filtering capabilities.
package main

import (
	"fmt"
	"os"

	"github.com/TFMV/stride/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

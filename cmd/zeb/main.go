// zeb is the entrypoint for the Zebra Link command-line client.
// Command wiring lives under internal/cli; this file only passes version
// metadata into the root command.
package main

import "github.com/kerns/zlink-zeb/internal/cli"

var version = "0.1.0-dev"

func main() {
	cli.Execute(version)
}

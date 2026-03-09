package main

import "github.com/mldotink/cli/cmd"

var version = "dev"

func main() {
	cmd.Version = version
	cmd.Execute()
}

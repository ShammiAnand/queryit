package main

import "github.com/shammianand/queryit/cmd"

// set by ldflags at build time
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	cmd.SetVersion(version, commit, buildDate)
	cmd.Execute()
}

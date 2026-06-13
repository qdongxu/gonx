package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

type options struct {
	configPath string
	version    bool
}

func main() {
	opts := parseFlags()

	if opts.version {
		fmt.Printf("gonx version %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	if err := run(opts); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() options {
	var opts options
	flag.StringVar(&opts.configPath, "c", "", "path to configuration file")
	flag.BoolVar(&opts.version, "version", false, "print version and exit")
	flag.Parse()
	return opts
}

func run(opts options) error {
	if opts.configPath == "" {
		return fmt.Errorf("configuration file required (-c flag)")
	}

	fmt.Fprintf(os.Stderr, "gonx %s starting...\n", version)
	fmt.Fprintf(os.Stderr, "config: %s\n", opts.configPath)

	return fmt.Errorf("not implemented: full server startup")
}

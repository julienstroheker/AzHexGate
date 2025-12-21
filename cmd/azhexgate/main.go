package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	defaultPort = 3000
	usageText   = `azhexgate - Azure Hybrid Connection reverse tunnel

Usage:
  azhexgate start --port <port>

Commands:
  start    Start the tunnel and forward traffic to localhost

Flags:
  --port   Local port to forward traffic to (default: 3000)
  --help   Show this help message
`
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "start":
		startCmd()
	case "--help", "-h", "help":
		fmt.Print(usageText)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}
}

func startCmd() {
	startFlags := flag.NewFlagSet("start", flag.ExitOnError)
	port := startFlags.Int("port", defaultPort, "Local port to forward traffic to")

	startFlags.Usage = func() {
		fmt.Fprint(os.Stderr, usageText)
	}

	if err := startFlags.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	fmt.Printf("Starting tunnel on port %d...\n", *port)
	fmt.Println("Tunnel logic not yet implemented")
}

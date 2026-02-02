package main

import (
	"fmt"
	"os"

	"github.com/emilianotisato/vibeit/internal/doctor"
	"github.com/emilianotisato/vibeit/internal/tui"
)

const version = "0.1.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "doctor":
			os.Exit(doctor.Run())
		case "version", "--version", "-v":
			fmt.Printf("vibeit %s\n", version)
			os.Exit(0)
		case "help", "--help", "-h":
			printHelp()
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}

	// Run the TUI
	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`vibeit - Workspace-centric vibe coding TUI

Usage:
  vibeit              Launch the TUI in current directory
  vibeit doctor       Check system dependencies
  vibeit version      Show version
  vibeit help         Show this help

Keybindings (in TUI):
  Ctrl+\              Enter command mode
  1-9                 Switch workspace
  n                   Open notes
  g                   Open lazygit
  t                   New terminal
  w                   New workspace
  q                   Quit / close tab`)
}

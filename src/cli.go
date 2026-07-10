package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func RunCLI(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 0
	}
	switch args[0] {
	case "--help", "-h", "help":
		printUsage()
		return 0
	case "--list", "list":
		for _, scenario := range AvailableScenarios() {
			fmt.Println(scenario)
		}
		return 0
	case "scenario":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "scenario name required")
			return 2
		}
		run, err := RunScenario(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 1
		}
		if err := PrintJSON(run.Report()); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 1
		}
		return 0
	case "validate":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "scenario name required")
			return 2
		}
		run, err := RunScenario(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 1
		}
		report := run.Report()
		if err := ValidateReport(report); err != nil {
			if errors.Is(err, ErrInvariantViolation) {
				fmt.Fprintf(os.Stderr, "validation failed for %s\n", args[1])
			} else {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			return 1
		}
		fmt.Printf("ok %s %s\n", args[1], report.StateDigest)
		return 0
	default:
		if strings.TrimSpace(args[0]) == "" {
			printUsage()
			return 0
		}
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func printUsage() {
	fmt.Println("OmenDTL")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  omendtl --list")
	fmt.Println("  omendtl scenario <name>")
	fmt.Println("  omendtl validate <name>")
}

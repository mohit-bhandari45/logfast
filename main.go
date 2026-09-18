package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var filePath = ""
	var filterKeyword string

	var rootCmd = &cobra.Command{
		Use:   "logfast",
		Short: "A high-throughput log stream analyzer",
		Long:  `logfast is a blazing fast CLI tool to ingest, parse, and analyze massive log files in real-time.`,

		Run: func(cmd *cobra.Command, args []string) {
			if filePath == "" {
				fmt.Println("Error: You must provide a log file path using the --file flag.")
				cmd.Usage() // Prints the help menu
				os.Exit(1)
			}

			fmt.Printf("Starting logfast...\n")
			fmt.Printf("Target File: %s\n", filePath)

			if filterKeyword != "" {
				fmt.Printf("Filtering for: %s\n", filterKeyword)
			} else {
				fmt.Printf("No filter applied (reading all lines).\n")
			}
		},
	}

	rootCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the log file (required)");
	rootCmd.Flags().StringVarP(&filterKeyword, "filter", "k", "", "Keyword to filter logs by (optional)");

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err);
		os.Exit(1);
	}
}

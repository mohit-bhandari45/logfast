package main

import (
	"bufio" // New! For scanning the file
	"fmt"
	"os"
	"strings" // New! For searching text

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
				cmd.Usage()
				os.Exit(1)
			}

			fmt.Printf("Analyzing file: %s...\n", filePath)

			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("Error opening file: %v\n", err)
				os.Exit(1)
			}
			defer file.Close()

			// we are not using any io or os package here, because they loads the whole file here.
			scanner := bufio.NewScanner(file)

			totalLines := 0
			matchCount := 0

			for scanner.Scan() {
				totalLines++
				line := scanner.Text()

				if filterKeyword != "" || strings.Contains(line, filterKeyword) {
					matchCount++

					if matchCount <= 5 {
						fmt.Println("->", line)
					}
				}
			}

			if err := scanner.Err(); err != nil {
				fmt.Printf("Error reading file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("\n--- Analysis Complete ---\n")
			fmt.Printf("Total lines processed: %d\n", totalLines)
			fmt.Printf("Total matches found: %d\n", matchCount)
		},
	}

	rootCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the log file (required)")
	rootCmd.Flags().StringVarP(&filterKeyword, "filter", "k", "", "Keyword to filter logs by (optional)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func main() {
	var filePath string
	var filterKeyword string

	var rootCmd = &cobra.Command{
		Use:   "logfast",
		Short: "A high-throughput log stream analyzer",
		
		Run: func(cmd *cobra.Command, args []string) {
			// Stop the program if the user forgot to pass a file path
			if filePath == "" {
				fmt.Println("Error: You must provide a log file path using the --file flag.")
				cmd.Usage()
				os.Exit(1)
			}

			fmt.Printf("Analyzing file: %s...\n", filePath)

			// Grab the file from the hard drive
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("Error opening file: %v\n", err)
				os.Exit(1)
			}
			// This ensures the file gets closed when we are done, even if the program crashes
			defer file.Close() 

			// bufio.Scanner is the secret sauce here. It reads the file in small chunks
			// instead of loading the whole 10GB file into RAM at once.
			scanner := bufio.NewScanner(file)
			
			totalLines := 0
			matchCount := 0

			// scanner.Scan() automatically looks for the Enter key (\n) 
			// and stops at the end of each line.
			for scanner.Scan() {
				totalLines++
				line := scanner.Text() 

				// We want the filter to be optional.
				// If they didn't provide a filter (filterKeyword == ""), match everything.
				// Otherwise, check if the line actually contains the keyword.
				if filterKeyword == "" || strings.Contains(line, filterKeyword) {
					matchCount++
					
					// Printing to the terminal is really slow. 
					// We only print the first 5 so we don't lag the computer on massive files.
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

	// Tie the command line flags to our variables
	rootCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the log file (required)")
	rootCmd.Flags().StringVarP(&filterKeyword, "filter", "k", "", "Keyword to filter logs by (optional)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

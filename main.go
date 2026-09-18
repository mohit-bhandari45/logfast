package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

// worker is our independent clone. We spin up one of these for every CPU core.
func worker(workerID int, jobs <-chan string, results chan<- int, filterKeyword string, wg *sync.WaitGroup) {
	// 1. wg.Done() checks this worker off the roll-call sheet when it finishes and dies.
	defer wg.Done()

	// 2. This is totally private to this specific clone. No other worker can see or touch it.
	localMatchCount := 0

	// 3. Pull lines off the conveyor belt until the main thread closes it.
	for line := range jobs {
		if filterKeyword == "" || strings.Contains(line, filterKeyword) {
			localMatchCount++ // Silently tally
		}
	}

	// 4. The jobs channel closed. We are done! 
	// Drop our final tally into the results bucket.
	results <- localMatchCount
}

func main() {
	var filePath string
	var filterKeyword string

	var rootCmd = &cobra.Command{
		Use:   "logfast",
		Short: "A high-throughput log stream analyzer",

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

			// --- CONCURRENCY ARCHITECTURE ---

			numWorkers := runtime.NumCPU()
			fmt.Printf("Starting %d workers...\n", numWorkers)

			// The Jobs Bucket: Holds up to 1000 lines. 
			// If it gets full, the main thread pauses reading the file until workers catch up.
			jobs := make(chan string, 1000)

			// The Results Bucket: Exactly one slot for every worker. 
			// It is mathematically impossible for this to get full and block a worker.
			results := make(chan int, numWorkers)

			var wg sync.WaitGroup

			// FAN-OUT: Spawn the independent worker clones
			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go worker(i, jobs, results, filterKeyword, &wg)
			}

			scanner := bufio.NewScanner(file)
			totalLines := 0

			// The main thread's ONLY job is reading the disk and feeding the workers
			for scanner.Scan() {
				totalLines++
				jobs <- scanner.Text()
			}

			if err := scanner.Err(); err != nil {
				fmt.Printf("Error reading file: %v\n", err)
				os.Exit(1)
			}

			// We hit the bottom of the file. Put up the "Closed" sign on the jobs bucket.
			close(jobs)

			// Our background guard. It waits for all workers to finish, 
			// then puts a padlock on the results bucket so the main thread knows when to stop waiting.
			go func() {
				wg.Wait()
				close(results)
			}()

			// FAN-IN: Reach into the results bucket, pull out the numbers, and add them up.
			// This loop automatically breaks when it feels the padlock (close(results)).
			totalMatches := 0
			for count := range results {
				totalMatches += count
			}

			fmt.Printf("\n--- Analysis Complete ---\n")
			fmt.Printf("Total lines processed: %d\n", totalLines)
			fmt.Printf("Total matches found: %d\n", totalMatches)
		},
	}

	rootCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the log file (required)")
	rootCmd.Flags().StringVarP(&filterKeyword, "filter", "k", "", "Keyword to filter logs by (optional)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

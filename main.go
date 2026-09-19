package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/spf13/cobra"
)

func worker(workerID int, jobs <-chan []byte, results chan<- int, filterKeyword []byte, wg *sync.WaitGroup, pool *sync.Pool) {
	defer wg.Done()

	localMatchCount := 0

	for chunk := range jobs {
		// Keep slicing the chunk until there is no data left
		originalArray := chunk;
		
		for len(chunk) > 0 {
			// Find the location of the next line break
			newlineIndex := bytes.IndexByte(chunk, '\n')
			
			var line []byte
			if newlineIndex == -1 {
				// No line break means this is the very last piece of text in the chunk
				line = chunk
				// Empty the chunk to stop the loop
				chunk = nil
			} else {
				// Slice out the exact line using zero-copy memory pointers
				line = chunk[:newlineIndex]
				// Shrink the remaining chunk by moving past the line break we just found
				chunk = chunk[newlineIndex+1:]
			}

			// Check if the line contains our keyword directly in the raw bytes
			if len(filterKeyword) == 0 || bytes.Contains(line, filterKeyword) {
				localMatchCount++
			}
		}
		pool.Put(originalArray[:cap(originalArray)])
	}

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

			numWorkers := runtime.NumCPU()
			fmt.Printf("Starting %d workers...\n", numWorkers)

			// Convert the string keyword to bytes once so workers don't have to
			filterBytes := []byte(filterKeyword)

			var chunkPool = sync.Pool{
				New: func() interface{} {
					return make([]byte, 64*1024)
				},
			}

			// Update the jobs channel to hold blocks of raw bytes
			jobs := make(chan []byte, 100)
			results := make(chan int, numWorkers)
			var wg sync.WaitGroup

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go worker(i, jobs, results, filterBytes, &wg, &chunkPool)
			}

			// Create a 64KB bucket to scoop data from the hard drive
			buf := make([]byte, 64*1024)
			// This cup holds words that get accidentally chopped in half
			var tail []byte

			for {
				// Scoop up to 64KB of data from the file
				n, err := file.Read(buf)
				
				if n > 0 {
					// Glue the chopped word from the previous scoop to the front of this new scoop
					chunk := append(tail, buf[:n]...)

					// Scan backwards to find the last clean line break
					lastNewline := bytes.LastIndexByte(chunk, '\n')
					
					if lastNewline != -1 {
						// Create a fresh, safe memory box for the clean lines
						// We must make a copy so the next file read doesn't overwrite these bytes
						borrowedArray := chunkPool.Get().([]byte);
						sendChunk := borrowedArray[:lastNewline+1]
						copy(sendChunk, chunk[:lastNewline+1])
						
						// Ship the clean lines to the workers
						jobs <- sendChunk
						
						// Save the remaining chopped-off word into the tail cup for the next loop
						tail = chunk[lastNewline+1:]
					} else {
						// The entire chunk had no line breaks, so it's all one giant tail
						tail = chunk
					}
				}

				// Check if we hit the bottom of the file
				if err == io.EOF {
					// If there is any leftover chopped text, send it to the workers
					if len(tail) > 0 {
						jobs <- tail
					}
					// Exit the infinite loop
					break
				}
				if err != nil {
					fmt.Printf("Error reading file: %v\n", err)
					os.Exit(1)
				}
			}

			// Put up the closed sign on the jobs channel
			close(jobs)

			go func() {
				wg.Wait()
				close(results)
			}()

			totalMatches := 0
			for count := range results {
				totalMatches += count
			}

			fmt.Printf("\n--- Analysis Complete ---\n")
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

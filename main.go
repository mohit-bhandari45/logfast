package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"sync"

	"github.com/spf13/cobra"
)

// WorkerResult bundles the match count and collected latency values from a worker
type WorkerResult struct {
	matchCount int
	latencies  []int
}

// parseLatency extracts numeric latency values directly from raw bytes with zero heap allocations
func parseLatency(line []byte) (int, bool) {
	key := []byte("latency=")
	idx := bytes.Index(line, key)
	if idx == -1 {
		return 0, false
	}

	pos := idx + len(key)
	val := 0
	found := false

	// Convert ASCII byte characters directly into integer digits using CPU register math
	for pos < len(line) && line[pos] >= '0' && line[pos] <= '9' {
		val = val*10 + int(line[pos]-'0')
		pos++
		found = true
	}

	return val, found
}

// worker runs concurrently across CPU cores, parsing byte chunks and aggregating statistics
func worker(workerID int, jobs <-chan []byte, results chan<- WorkerResult, filterKeyword []byte, wg *sync.WaitGroup, pool *sync.Pool) {
	defer wg.Done()

	localMatchCount := 0
	var localLatencies []int

	for chunk := range jobs {
		// Keep a pointer to the original borrowed slice so we can return it to the pool
		originalArray := chunk

		for len(chunk) > 0 {
			// Locate line breaks inside the raw byte chunk
			newlineIndex := bytes.IndexByte(chunk, '\n')

			var line []byte
			if newlineIndex == -1 {
				// The chunk has reached its end without an ending newline
				line = chunk
				chunk = nil
			} else {
				// Slice out the exact line using zero-copy memory pointers
				line = chunk[:newlineIndex]
				// Advance the remaining chunk forward past the newline
				chunk = chunk[newlineIndex+1:]
			}

			// Match lines against the optional filter keyword
			if len(filterKeyword) == 0 || bytes.Contains(line, filterKeyword) {
				localMatchCount++

				// Extract the latency value from this line without string conversions
				if lat, ok := parseLatency(line); ok {
					localLatencies = append(localLatencies, lat)
				}
			}
		}

		// Return the borrowed buffer to the sync.Pool to avoid garbage collection
		pool.Put(originalArray[:cap(originalArray)])
	}

	// Send local tallies and latencies back to the main thread
	results <- WorkerResult{
		matchCount: localMatchCount,
		latencies:  localLatencies,
	}
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

			// Convert filter to bytes once so workers do not perform string conversions
			filterBytes := []byte(filterKeyword)

			// Buffer pool to recycle 64KB arrays and prevent GC allocation pressure
			var chunkPool = sync.Pool{
				New: func() interface{} {
					return make([]byte, 64*1024)
				},
			}

			// Channels for distributing chunks and gathering worker results
			jobs := make(chan []byte, 100)
			results := make(chan WorkerResult, numWorkers)
			var wg sync.WaitGroup

			// Fan-Out: start worker routines across CPU cores
			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go worker(i, jobs, results, filterBytes, &wg, &chunkPool)
			}

			// A reusable buffer to stream chunks from disk
			buf := make([]byte, 64*1024)
			var tail []byte

			for {
				// Read a 64KB block from the file
				n, err := file.Read(buf)

				if n > 0 {
					// Prepend leftover bytes from the previous block to keep lines intact
					chunk := append(tail, buf[:n]...)

					// Look backwards for the last complete newline character
					lastNewline := bytes.LastIndexByte(chunk, '\n')

					if lastNewline != -1 {
						// Borrow a recycled buffer from the pool
						borrowedArray := chunkPool.Get().([]byte)
						sendChunk := borrowedArray[:lastNewline+1]
						copy(sendChunk, chunk[:lastNewline+1])

						// Forward the complete lines chunk to the worker channel
						jobs <- sendChunk

						// Retain leftover sliced text for the next read cycle
						tail = chunk[lastNewline+1:]
					} else {
						// No line breaks present in this block; carry all bytes over
						tail = chunk
					}
				}

				// Check for end of file
				if err == io.EOF {
					// Forward any final remaining bytes in the tail to the workers
					if len(tail) > 0 {
						jobs <- tail
					}
					break
				}
				if err != nil {
					fmt.Printf("Error reading file: %v\n", err)
					os.Exit(1)
				}
			}

			// Signal workers that all chunks have been dispatched
			close(jobs)

			// Monitor workers and close results channel upon full completion
			go func() {
				wg.Wait()
				close(results)
			}()

			// Fan-In: aggregate counts and response times across all workers
			totalMatches := 0
			var allLatencies []int

			for res := range results {
				totalMatches += res.matchCount
				allLatencies = append(allLatencies, res.latencies...)
			}

			fmt.Printf("\n--- Analysis Complete ---\n")
			fmt.Printf("Total matches found: %d\n", totalMatches)

			// Compute percentile metrics if response times were recorded
			if len(allLatencies) > 0 {
				sort.Ints(allLatencies)

				p50 := allLatencies[len(allLatencies)*50/100]
				p95 := allLatencies[len(allLatencies)*95/100]
				p99 := allLatencies[len(allLatencies)*99/100]

				fmt.Printf("Latency p50: %dms\n", p50)
				fmt.Printf("Latency p95: %dms\n", p95)
				fmt.Printf("Latency p99: %dms\n", p99)
			}
		},
	}

	rootCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the log file (required)")
	rootCmd.Flags().StringVarP(&filterKeyword, "filter", "k", "", "Keyword to filter logs by (optional)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

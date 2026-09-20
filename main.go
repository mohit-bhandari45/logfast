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

	"logfast/pkg/parser"
	"logfast/pkg/ringbuffer"
)

// WorkerResult bundles the match count and collected latency values from a worker routine
type WorkerResult struct {
	matchCount int
	latencies  []int
}

// worker runs concurrently across CPU cores, parsing byte chunks and collecting metrics in a local ring buffer
func worker(workerID int, jobs <-chan []byte, results chan<- WorkerResult, filterKeyword []byte, wg *sync.WaitGroup, pool *sync.Pool) {
	defer wg.Done()

	localMatchCount := 0
	// Each worker maintains its own fixed-size ring buffer to prevent unbounded slice allocations
	localRing := ringbuffer.New(10000)

	for chunk := range jobs {
		// Retain reference to the original borrowed slice to safely return it to the sync.Pool
		originalArray := chunk

		for len(chunk) > 0 {
			// Locate line breaks inside the raw byte chunk
			newlineIndex := bytes.IndexByte(chunk, '\n')

			var line []byte
			if newlineIndex == -1 {
				// The chunk has reached its end without a trailing newline
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

				// Extract latency values directly from the byte slice without string conversions
				if lat, ok := parser.ParseLatency(line); ok {
					localRing.Add(lat)
				}
			}
		}

		// Return the borrowed buffer to the sync.Pool to eliminate garbage collection pauses
		pool.Put(originalArray[:cap(originalArray)])
	}

	// Send local match tallies and sampled latency window back to the main thread
	results <- WorkerResult{
		matchCount: localMatchCount,
		latencies:  localRing.Values(),
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

			// Convert filter keyword to raw bytes once so workers do not perform string conversions
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

			// Fan-In: aggregate counts and feed latencies into a global fixed-size streaming window
			globalRing := ringbuffer.New(100000)
			totalMatches := 0

			for res := range results {
				totalMatches += res.matchCount
				for _, lat := range res.latencies {
					globalRing.Add(lat)
				}
			}

			fmt.Printf("\n--- Analysis Complete ---\n")
			fmt.Printf("Total matches found: %d\n", totalMatches)

			// Compute percentile metrics over the streaming window
			samples := globalRing.Values()
			if len(samples) > 0 {
				sort.Ints(samples)

				p50 := samples[len(samples)*50/100]
				p95 := samples[len(samples)*95/100]
				p99 := samples[len(samples)*99/100]

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

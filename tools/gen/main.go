package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func main() {
	fileName := "massive.log"
	targetLines := 10000000 // 10 Million log lines (~700 Megabytes)

	fmt.Printf("Generating %d log lines into %s...\n", targetLines, fileName)
	start := time.Now()

	file, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	// Use a 256KB write buffer so disk writes happen in bulk
	writer := bufio.NewWriterSize(file, 256*1024)

	// Pre-made realistic log templates
	sampleLines := [][]byte{
		[]byte("[INFO] 2026-09-20 10:00:01 user authentication succeeded latency=12\n"),
		[]byte("[ERROR] 2026-09-20 10:00:02 database connection timeout latency=450\n"),
		[]byte("[INFO] 2026-09-20 10:00:03 cache hit for key profile_102 latency=5\n"),
		[]byte("[ERROR] 2026-09-20 10:00:04 payment gateway unreachable latency=1250\n"),
		[]byte("[WARNING] 2026-09-20 10:00:05 high memory pressure detected latency=85\n"),
	}

	for i := 0; i < targetLines; i++ {
		writer.Write(sampleLines[i%len(sampleLines)])
	}

	writer.Flush()
	fmt.Printf("Done! Generated 10,000,000 lines in %v.\n", time.Since(start))
}
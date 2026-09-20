package parser

import (
	"strconv"
	"strings"
	"testing"
)

// The standard way 99% of developers write it
func ParseLatencyStrconv(line []byte) (int, bool) {
	// 1. Convert bytes to string (ALLOCATION!)
	s := string(line)
	key := "latency="
	idx := strings.Index(s, key)
	if idx == -1 {
		return 0, false
	}

	sub := s[idx+len(key):]
	end := 0
	for end < len(sub) && sub[end] >= '0' && sub[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}

	// 2. Call strconv.Atoi on string slice
	val, err := strconv.Atoi(sub[:end])
	if err != nil {
		return 0, false
	}
	return val, true
}

// Benchmark our Zero-Allocation Byte Engine
func BenchmarkOurZeroAlloc(b *testing.B) {
	sampleLine := []byte("[ERROR] 2026-09-20 database connection timeout latency=450")
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ParseLatency(sampleLine)
	}
}

// Benchmark the Standard Strconv way
func BenchmarkStandardStrconv(b *testing.B) {
	sampleLine := []byte("[ERROR] 2026-09-20 database connection timeout latency=450")
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ParseLatencyStrconv(sampleLine)
	}
}
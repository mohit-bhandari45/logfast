package parser

import (
	"testing"
)

func BenchmarkParseLatency(b *testing.B) {
	// Sample log line to feed the parser
	sampleLine := []byte("[ERROR] 2026-09-20 database connection timeout latency=450")

	// Enable memory reporting so Go measures heap allocations
	b.ReportAllocs()

	// Reset the timer so setting up sampleLine above doesn't count against our speed
	b.ResetTimer()

	// b.N is dynamically adjusted by Go (e.g. 10 million times) until the result is accurate
	for i := 0; i < b.N; i++ {
		val, ok := ParseLatency(sampleLine)
		if !ok || val != 450 {
			b.Fatalf("unexpected parse result: %d", val)
		}
	}
}
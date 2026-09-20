package ringbuffer

import (
	"testing"
)

func BenchmarkRingBufferAdd(b *testing.B) {
	// Create a 100,000 slot ring buffer
	rb := New(100000)

	// Turn on memory tracking
	b.ReportAllocs()

	// Start the stopwatch
	b.ResetTimer()

	// Continuously stream integers into the ring buffer
	for i := 0; i < b.N; i++ {
		rb.Add(i)
	}
}
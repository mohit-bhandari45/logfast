package ringbuffer

import (
	"testing"
)

// BenchmarkOurRingBuffer measures our fixed-size circular buffer
// It should achieve 0 B/op and 0 allocs/op because memory is bounded and recycled
func BenchmarkOurRingBuffer(b *testing.B) {
	rb := New(100000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rb.Add(i)
	}
}

// BenchmarkStandardDynamicSlice measures the naive way 99% of developers collect metrics
// Using append() on an unbounded slice forces frequent memory reallocations (runtime.growslice)
func BenchmarkStandardDynamicSlice(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	var slice []int
	for i := 0; i < b.N; i++ {
		slice = append(slice, i)
	}
}
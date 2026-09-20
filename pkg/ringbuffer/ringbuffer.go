package ringbuffer

// RingBuffer stores a fixed-size window of integers in a circular buffer
type RingBuffer struct {
	data     []int
	capacity int
	writePos int
	isFull   bool
}

// New allocates a ring buffer with a predetermined capacity
func New(capacity int) *RingBuffer {
	return &RingBuffer{
		data:     make([]int, capacity),
		capacity: capacity,
	}
}

// Add appends a value into the ring buffer, overwriting the oldest entry when capacity is reached
func (rb *RingBuffer) Add(val int) {
	rb.data[rb.writePos] = val
	rb.writePos = (rb.writePos + 1) % rb.capacity
	if rb.writePos == 0 {
		rb.isFull = true
	}
}

// Values returns a slice copy of the active elements currently recorded in the buffer
func (rb *RingBuffer) Values() []int {
	if !rb.isFull {
		out := make([]int, rb.writePos)
		copy(out, rb.data[:rb.writePos])
		return out
	}
	out := make([]int, rb.capacity)
	copy(out, rb.data)
	return out
}

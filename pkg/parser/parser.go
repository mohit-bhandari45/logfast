package parser

import (
	"bytes"
)

// ParseLatency extracts integer values following the key pattern 'latency=' directly from raw bytes
// It performs in-place ASCII digit conversion without allocating heap memory or strings
func ParseLatency(line []byte) (int, bool) {
	key := []byte("latency=")
	idx := bytes.Index(line, key)
	if idx == -1 {
		return 0, false
	}

	pos := idx + len(key)
	val := 0
	found := false

	// Convert ASCII byte characters directly into numeric digits using CPU register math
	for pos < len(line) && line[pos] >= '0' && line[pos] <= '9' {
		val = val*10 + int(line[pos]-'0')
		pos++
		found = true
	}

	return val, found
}

package bash

import (
	"bytes"
	"sync"
)

// threadSafeBuffer provides a thread-safe wrapper around bytes.Buffer
type threadSafeBuffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// Write implements io.Writer interface
func (b *threadSafeBuffer) Write(p []byte) (n int, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.Write(p)
}

// String returns the contents of the buffer as a string
func (b *threadSafeBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.String()
}

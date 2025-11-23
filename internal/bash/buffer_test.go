package bash

import (
	"sync"
	"testing"
)

func TestThreadSafeBuffer_SingleThread(t *testing.T) {
	buf := &threadSafeBuffer{}

	// Test single write
	n, err := buf.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected to write 5 bytes, but wrote %d", n)
	}
	if buf.String() != "hello" {
		t.Errorf("Expected buffer content to be 'hello', but got '%s'", buf.String())
	}

	// Test multiple writes
	n, err = buf.Write([]byte(" world"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != 6 {
		t.Errorf("Expected to write 6 bytes, but wrote %d", n)
	}
	if buf.String() != "hello world" {
		t.Errorf("Expected buffer content to be 'hello world', but got '%s'", buf.String())
	}
}

func TestThreadSafeBuffer_ConcurrentWrites(t *testing.T) {
	buf := &threadSafeBuffer{}
	const numGoroutines = 10
	const writesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines to write to the buffer concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				_, err := buf.Write([]byte("a"))
				if err != nil {
					t.Errorf("Concurrent write failed: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// Check if the total number of 'a' characters matches what we expect
	content := buf.String()
	expectedLen := numGoroutines * writesPerGoroutine
	if len(content) != expectedLen {
		t.Errorf("Expected buffer length to be %d, but got %d", expectedLen, len(content))
	}

	// Verify all characters are 'a'
	for i, c := range content {
		if c != 'a' {
			t.Errorf("Unexpected character at position %d: expected 'a', got '%c'", i, c)
		}
	}
}

func TestThreadSafeBuffer_ConcurrentWriteAndRead(t *testing.T) {
	buf := &threadSafeBuffer{}
	const numWriters = 5
	const numReaders = 5
	const writesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numWriters + numReaders)

	// Launch writer goroutines
	for i := 0; i < numWriters; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				_, err := buf.Write([]byte("a"))
				if err != nil {
					t.Errorf("Concurrent write failed: %v", err)
				}
			}
		}()
	}

	// Launch reader goroutines
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				_ = buf.String() // Just read the content
			}
		}()
	}

	wg.Wait()

	// Verify final content
	content := buf.String()
	expectedLen := numWriters * writesPerGoroutine
	if len(content) != expectedLen {
		t.Errorf("Expected buffer length to be %d, but got %d", expectedLen, len(content))
	}
}

package matroska

import (
	"bytes"
	"io"
	"testing"
)

// TestFakeSeeker tests the behavior of the fakeSeeker.
func TestFakeSeeker(t *testing.T) {
	data := []byte("hello world")
	r := bytes.NewReader(data)
	fs := &fakeSeeker{r: r}

	// Test Read
	t.Run("Read", func(t *testing.T) {
		buf := make([]byte, 5)
		n, err := fs.Read(buf)
		if err != nil {
			t.Fatalf("Read() failed: %v", err)
		}
		if n != 5 {
			t.Errorf("Expected to read 5 bytes, got %d", n)
		}
		if string(buf) != "hello" {
			t.Errorf("Expected to read 'hello', got %q", string(buf))
		}
	})

	// Test Read to EOF
	t.Run("Read_EOF", func(t *testing.T) {
		// Drain the rest of the reader
		_, _ = io.ReadAll(fs)

		buf := make([]byte, 1)
		n, err := fs.Read(buf)
		if err != io.EOF {
			t.Errorf("Expected EOF, got %v", err)
		}
		if n != 0 {
			t.Errorf("Expected to read 0 bytes at EOF, got %d", n)
		}
	})

	// Test Seek
	t.Run("Seek", func(t *testing.T) {
		pos, err := fs.Seek(0, io.SeekStart)
		if err == nil {
			t.Error("Seek() should always return an error")
		}
		if pos != -1 {
			t.Errorf("Seek() should return position -1 on error, got %d", pos)
		}
	})
}

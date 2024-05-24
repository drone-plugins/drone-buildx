package docker

import (
	"bytes"
	"io"
	"testing"
)

// helper function for creating a tee with a bidirectional channel for testing
func createTestTee(w io.Writer) (*tee, chan string) {
	status := make(chan string, 100) // create a channel with buffer size
	return &tee{w: w, status: status}, status
}

// TestTeeWrite checks that data is correctly written to the underlying writer and the status channel.
func TestTeeWrite(t *testing.T) {
	buf := new(bytes.Buffer)
	teeInstance, statusChan := createTestTee(buf)

	input := []byte("hello")
	written, err := teeInstance.Write(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if written != len(input) {
		t.Errorf("Expected %d bytes to be written, got %d", len(input), written)
	}
	if buf.String() != string(input) {
		t.Errorf("Expected buffer content to be '%s', got '%s'", string(input), buf.String())
	}

	select {
	case msg := <-statusChan:
		if msg != string(input) {
			t.Errorf("Expected status channel to receive '%s', got '%s'", string(input), msg)
		}
	default:
		t.Error("Expected message to be available on status channel")
	}
}

// TestTeeWriteChannelFull tests that the tee does not block if the status channel is full.
func TestTeeWriteChannelFull(t *testing.T) {
	buf := new(bytes.Buffer)
	teeInstance, statusChan := createTestTee(buf)

	// Fill the channel
	for i := 0; i < 100; i++ {
		statusChan <- "fill"
	}

	input := []byte("test")
	_, err := teeInstance.Write(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// No way to directly test that it does not block, but if it did block, this test would not complete.
}

// TestTeeClose checks that the Close method closes the status channel.
func TestTeeClose(t *testing.T) {
	buf := new(bytes.Buffer)
	teeInstance, statusChan := createTestTee(buf)

	err := teeInstance.Close() // Close the tee instance
	if err != nil {
		t.Fatalf("Expected no error on close, got %v", err)
	}

	_, ok := <-statusChan
	if ok {
		t.Error("Expected status channel to be closed, but it was still open")
	}
}

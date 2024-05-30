package docker

import "io"

// tee is a structure that wraps an io.Writer and includes a status channel to
// report the data written.
type (
	tee struct {
		w      io.Writer   // The underlying writer where data will be written
		status chan string // Channel to report the written data
	}
)

// Write writes data to the underlying writer and sends a copy to the status channel.
func (t *tee) Write(p []byte) (n int, err error) {
	n, err = t.w.Write(p) // Write data to the underlying writer
	if err != nil {
		return n, err // Return if there's an error during writing
	}
	select {
	case t.status <- string(p):
		// Successfully sent data to the status channel
	default:
		// If the status channel is full, drop the message to avoid blocking
	}
	return n, nil // Return the number of bytes written and any error encountered
}

// Close closes the status channel. This indicates that no more data will be sent.
func (t *tee) Close() error {
	close(t.status) // Close the status channel
	return nil
}

// Tee creates a new tee instance that writes data to the provided io.Writer and
// sends copies of the written data to a status channel.
func Tee(w io.Writer) (*tee, <-chan string) {
	status := make(chan string, 100)          // Create a buffered channel with a capacity of 100 to reduce to risk of dropping messages
	return &tee{w: w, status: status}, status // Return the new tee instance and the status channel
}

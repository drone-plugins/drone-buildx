package docker

import "io"

type (
	tee struct {
		w      io.Writer
		status chan string
	}
)

func (t *tee) Write(p []byte) (n int, err error) {
	n, err = t.w.Write(p)
	if err != nil {
		return n, err
	}
	select {
	case t.status <- string(p):
		// Successfully sent to the channel
	default:
		// Drop the message if the channel is full to avoid blocking
	}
	return n, nil
}

func (t *tee) Close() error {
	close(t.status)
	return nil
}

func Tee(w io.Writer) (*tee, <-chan string) {
	status := make(chan string, 100) // Buffered channel to reduce the risk of dropping messages
	return &tee{w: w, status: status}, status
}

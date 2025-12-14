package systray

// wait for channel to receive a value, with timeout
import (
	"testing"
	"time"
)

// waitForChannel waits for a value on the given channel or times out after the specified duration.
// It returns true if a value was received, false if it timed out.
func waitForChannel[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	timeout := 2 * time.Second

	if ch == nil {
		t.Fatal("Channel is nil")
	}

	select {
	case t := <-ch:
		return t
	case <-time.After(timeout):
		t.Fatal("Timeout waiting for channel")
	}
	panic("unreachable")
}

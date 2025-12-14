package mock

import (
	"log"
	"time"
)

func sendNonBlockingToCh[T any](ch chan<- T, value T) {
	select {
	case ch <- value:
	case <-time.After(10 * time.Millisecond):
		log.Println("value lost, no reader")
	}
}

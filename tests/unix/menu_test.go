//go:build (linux || freebsd || openbsd || netbsd) && !android

package unix

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"fyne.io/systray"
	"fyne.io/systray/tests/unix/mock"
	"github.com/stretchr/testify/assert"
)

func TestInitWithMockDBus(t *testing.T) {
	// Start a dedicated D-Bus daemon for this test
	w := mock.NewStatusNotifierWatcher(t)
	defer w.Close()

	onExit := func() {
		now := time.Now()
		fmt.Println("Exit at", now.String())
	}

	var initialMenuBuilt sync.WaitGroup
	initialMenuBuilt.Add(1)

	title := "Awesome App"
	onReady := func() {
		systray.SetTitle(title)
		systray.SetTooltip("Lantern")
		mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
		go func() {
			for range mQuit.ClickedCh {
				fmt.Println("Requesting quit")
				systray.Quit()
			}
		}()

		initialMenuBuilt.Done()
	}

	go systray.Run(onReady, onExit)
	fmt.Println("Waiting for onReady")
	initialMenuBuilt.Wait()
	fmt.Println("On ready finished")

	time.Sleep(10 * time.Millisecond)

	item := w.Items()[0]
	itemProperties := item.Properties
	assert.Equal(t, title, itemProperties.Title)

	systray.Quit()
}

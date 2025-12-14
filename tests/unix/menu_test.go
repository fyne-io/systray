//go:build (linux || freebsd || openbsd || netbsd) && !android

package systray

import (
	"sync/atomic"
	"testing"

	"fyne.io/systray"
	"fyne.io/systray/example/icon"
	"fyne.io/systray/tests/unix/mock"
	"github.com/stretchr/testify/assert"
)

func TestCheckSniProperties(t *testing.T) {
	// Start a dedicated D-Bus daemon for this test
	w := mock.NewStatusNotifierWatcher(t)
	defer w.Close()

	onReadyCalled := atomic.Bool{}
	onReady := func() {
		onReadyCalled.Store(true)
	}

	onExitCalled := atomic.Bool{}
	onExit := func() {
		onExitCalled.Store(true)
	}

	title := "App Title"
	systray.SetTitle(title)

	tooltip := "Tooltip 棒棒嗒"
	systray.SetTooltip(tooltip)
	systray.SetTemplateIcon(icon.Data, icon.Data)

	go systray.Run(onReady, onExit)

	// wait for the item to be registered
	item := waitForChannel(t, w.ItemRegisteredCh)
	assert.True(t, onReadyCalled.Load(), "onReady have been called")

	itemProperties := item.Properties
	assert.Equal(t, title, itemProperties.Title.Get())
	assert.Equal(t, tooltip, itemProperties.ToolTip.Get().Title)
	assert.Equal(t, mock.IconPixmapFromData(icon.Data), itemProperties.IconPixmap.Get()[0])

	// check if updates are reflected
	newTitle := "Title 2.0"
	systray.SetTitle(newTitle)
	waitForChannel(t, itemProperties.Title.ChangedCh)
	assert.Equal(t, newTitle, itemProperties.Title.Get())

	newTooltip := "New tooltip 🚀"
	systray.SetTooltip(newTooltip)
	waitForChannel(t, itemProperties.ToolTip.ChangedCh)
	assert.Equal(t, newTooltip, itemProperties.ToolTip.Get().Title)

	systray.Quit()

	waitForChannel(t, w.ItemRemoveCh)
	assert.True(t, onExitCalled.Load(), "onExit have been called")

	w.Close()
}

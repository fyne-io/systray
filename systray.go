// Package systray is a cross-platform Go library to place an icon and menu in the notification area.
package systray

import (
	"log"
	"sync"
	"sync/atomic"
)

type Icon struct {
	native *nativeIcon

	id    uint32
	items sync.Map
}

func NewIcon() (*Icon, error) {
	native, err := newNativeIcon()
	if err != nil {
		return nil, err
	}

	return &Icon{
		native: native,
	}, nil
}

func (icon *Icon) nextID() uint32 {
	return atomic.AddUint32(&icon.id, 1)
}

// This helper function allows us to call systrayExit only once,
// without accidentally calling it twice in the same lifetime.
func runSystrayExit() {
	if !systrayExitCalled {
		systrayExitCalled = true
		systrayExit()
	}
}

func systrayMenuItemSelected(id uint32) {
	menuItemsLock.RLock()
	item, ok := menuItems[id]
	menuItemsLock.RUnlock()
	if !ok {
		log.Printf("systray error: no menu item with ID %d\n", id)
		return
	}
	select {
	case item.ClickedCh <- struct{}{}:
	// in case no one waiting for the channel
	default:
	}
}

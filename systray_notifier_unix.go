//go:build (linux || freebsd || openbsd || netbsd) && !android

package systray

import (
	"fyne.io/systray/internal/generated/notifier"
	"github.com/godbus/dbus/v5"
)

type leftRightNotifierItem struct {
}

func newLeftRightNotifierItem() notifier.StatusNotifierItemer {
	return &leftRightNotifierItem{}
}

func (i *leftRightNotifierItem) Activate(_, _ int32) *dbus.Error {
	if f := tappedLeft; f == nil {
		return &dbus.ErrMsgUnknownMethod
	}

	tappedLeft()
	return nil
}

func (i *leftRightNotifierItem) ContextMenu(_, _ int32) *dbus.Error {
	if f := tappedRight; f == nil {
		return &dbus.ErrMsgUnknownMethod
	}

	tappedRight()
	return nil
}

func (i *leftRightNotifierItem) SecondaryActivate(_, _ int32) *dbus.Error {
	// Middle-click: use dedicated handler if set, otherwise fall back to right-click
	if f := tappedMiddle; f != nil {
		tappedMiddle()
		return nil
	}

	if f := tappedRight; f == nil {
		return &dbus.ErrMsgUnknownMethod
	}

	tappedRight()
	return nil
}

func (i *leftRightNotifierItem) Scroll(delta int32, orientation string) *dbus.Error {
	if f := scrolled; f == nil {
		return &dbus.ErrMsgUnknownMethod
	}

	var direction ScrollDirection
	if orientation == "horizontal" {
		if delta > 0 {
			direction = ScrollRight
		} else {
			direction = ScrollLeft
		}
	} else {
		// "vertical" or default
		if delta > 0 {
			direction = ScrollUp
		} else {
			direction = ScrollDown
		}
	}

	scrolled(direction)
	return nil
}

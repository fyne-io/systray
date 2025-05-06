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
	systrayLeftClick()
	return nil
}

func (i *leftRightNotifierItem) ContextMenu(_, _ int32) *dbus.Error {
	return &dbus.ErrMsgUnknownMethod
}

func (i *leftRightNotifierItem) SecondaryActivate(_, _ int32) *dbus.Error {
	systrayRightClick()
	return nil
}

func (i *leftRightNotifierItem) Scroll(_ int32, _ string) *dbus.Error {
	return &dbus.ErrMsgUnknownMethod
}

package systray

import (
	"sync"
	"sync/atomic"
)

var (
	defaultIcon     *Icon
	defaultIconOnce sync.Once
)

func initDefaultIcon() {
	defaultIconOnce.Do(func() {
		defaultIcon, _ = NewIcon()
	})
}

// Run initializes GUI and starts the event loop, then invokes the onReady
// callback. It blocks until systray.Quit() is called.
func Run(onReady, onExit func()) {
	initDefaultIcon()

	defaultIcon.setInternalLoop(true)
	Register(onReady, onExit)

	defaultIcon.nativeLoop()
}

// RunWithExternalLoop allows the systemtray module to operate with other tookits.
// The returned start and end functions should be called by the toolkit when the application has started and will end.
func RunWithExternalLoop(onReady, onExit func()) (start, end func()) {
	initDefaultIcon()

	Register(onReady, onExit)

	return nativeStart, func() {
		nativeEnd()
		Quit()
	}
}

// Register initializes GUI and registers the callbacks but relies on the
// caller to run the event loop somewhere else. It's useful if the program
// needs to show other UI elements, for example, webview.
// To overcome some OS weirdness, On macOS versions before Catalina, calling
// this does exactly the same as Run().
func Register(onReady func(), onExit func()) {
	initDefaultIcon()

	if onReady == nil {
		systrayReady = func() {}
	} else {
		// Run onReady on separate goroutine to avoid blocking event loop
		readyCh := make(chan interface{})
		go func() {
			<-readyCh
			onReady()
		}()
		systrayReady = func() {
			close(readyCh)
		}
	}
	// unlike onReady, onExit runs in the event loop to make sure it has time to
	// finish before the process terminates
	if onExit == nil {
		onExit = func() {}
	}
	systrayExit = onExit
	systrayExitCalled = false
	defaultIcon.registerSystray()
}

// ResetMenu will remove all menu items
func ResetMenu() {
	initDefaultIcon()

	defaultIcon.resetMenu()
}

// Quit the systray
func Quit() {
	defaultIcon.Quit()
}

// AddMenuItem adds a menu item with the designated title and tooltip.
// It can be safely invoked from different goroutines.
// Created menu items are checkable on Windows and OSX by default. For Linux you have to use AddMenuItemCheckbox
func AddMenuItem(title string, tooltip string) *MenuItem {
	initDefaultIcon()

	item := defaultIcon.newMenuItem(title, tooltip, nil)
	item.update()
	return item
}

// AddMenuItemCheckbox adds a menu item with the designated title and tooltip and a checkbox for Linux.
// It can be safely invoked from different goroutines.
// On Windows and OSX this is the same as calling AddMenuItem
func AddMenuItemCheckbox(title string, tooltip string, checked bool) *MenuItem {
	initDefaultIcon()

	item := defaultIcon.newMenuItem(title, tooltip, nil)
	item.isCheckable = true
	item.checked = checked
	item.update()
	return item
}

// AddSeparator adds a separator bar to the menu
func AddSeparator() {
	initDefaultIcon()

	defaultIcon.addSeparator(atomic.AddUint32(&currentID, 1), 0)
}

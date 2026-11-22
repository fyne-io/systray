//go:build (linux || freebsd || openbsd || netbsd) && !android

package systray

import (
	"testing"

	"github.com/godbus/dbus/v5"

	"fyne.io/systray/internal/generated/notifier"
)

func TestPropSpecCoversIntrospection(t *testing.T) {
	served := instance.createPropSpec()["org.kde.StatusNotifierItem"]
	for _, p := range notifier.IntrospectDataStatusNotifierItem.Properties {
		if _, ok := served[p.Name]; !ok {
			t.Errorf("property %q is advertised via introspection but not served", p.Name)
		}
	}
}

func TestWatcherOwnerChanged(t *testing.T) {
	const (
		nameOwnerChanged = "org.freedesktop.DBus.NameOwnerChanged"
		watcher          = "org.kde.StatusNotifierWatcher"
	)

	for name, tt := range map[string]struct {
		sig              *dbus.Signal
		reregister, stop bool
	}{
		"watcher replaced": {
			sig:        &dbus.Signal{Name: nameOwnerChanged, Body: []any{watcher, ":1.7", ":1.9"}},
			reregister: true,
		},
		"watcher appeared": {
			sig:        &dbus.Signal{Name: nameOwnerChanged, Body: []any{watcher, "", ":1.9"}},
			reregister: true,
		},
		"watcher went away": {
			sig: &dbus.Signal{Name: nameOwnerChanged, Body: []any{watcher, ":1.7", ""}},
		},
		"another name changed owner": {
			sig: &dbus.Signal{Name: nameOwnerChanged, Body: []any{"org.example.Other", "", ":1.9"}},
		},
		"NameAcquired": {
			sig: &dbus.Signal{Name: "org.freedesktop.DBus.NameAcquired", Body: []any{":1.42"}},
		},
		"NameLost": {
			sig: &dbus.Signal{Name: "org.freedesktop.DBus.NameLost", Body: []any{":1.42"}},
		},
		"unrelated signal sharing the connection": {
			sig: &dbus.Signal{Name: "org.freedesktop.Notifications.NotificationClosed", Body: []any{uint32(1), uint32(2)}},
		},
		"NameOwnerChanged with a short body": {
			sig: &dbus.Signal{Name: nameOwnerChanged, Body: []any{watcher}},
		},
		"NameOwnerChanged with a non-string owner": {
			sig: &dbus.Signal{Name: nameOwnerChanged, Body: []any{watcher, "", 42}},
		},
		"connection gone": {
			sig:  nil,
			stop: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			reregister, stop := watcherOwnerChanged(tt.sig)
			if reregister != tt.reregister {
				t.Errorf("reregister = %v, want %v", reregister, tt.reregister)
			}
			if stop != tt.stop {
				t.Errorf("stop = %v, want %v", stop, tt.stop)
			}
		})
	}
}

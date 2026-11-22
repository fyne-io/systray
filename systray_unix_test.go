//go:build (linux || freebsd || openbsd || netbsd) && !android

package systray

import (
	"testing"

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

// A hidden tray is Passive, which StatusNotifierItem hosts do not show; a
// visible one is Active. The state set before the tray is exported is the
// one it is exported with.
func TestStatusFollowsVisibility(t *testing.T) {
	defer Show()

	status := func() string {
		return instance.createPropSpec()["org.kde.StatusNotifierItem"]["Status"].Value.(string)
	}
	if got := status(); got != "Active" {
		t.Fatalf("a new tray has Status %q, want Active", got)
	}
	Hide()
	if got := status(); got != "Passive" {
		t.Fatalf("after Hide Status is %q, want Passive", got)
	}
	Show()
	if got := status(); got != "Active" {
		t.Fatalf("after Show Status is %q, want Active", got)
	}
}

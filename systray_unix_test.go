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

// Attention shows as NeedsAttention until the user activates the item, or the
// application shows or hides it. A hidden tray stays Passive.
func TestStatusFollowsAttention(t *testing.T) {
	defer Show()

	status := func() string {
		return instance.createPropSpec()["org.kde.StatusNotifierItem"]["Status"].Value.(string)
	}
	steps := []struct {
		name string
		do   func()
		want string
	}{
		{"RequestAttention", RequestAttention, "NeedsAttention"},
		{"clearAttention", clearAttention, "Active"},
		{"Hide", Hide, "Passive"},
		{"RequestAttention while hidden", RequestAttention, "Passive"},
		{"Show after ignored request", Show, "Active"},
		{"RequestAttention again", RequestAttention, "NeedsAttention"},
		{"Show", Show, "Active"},
		{"RequestAttention before Hide", RequestAttention, "NeedsAttention"},
		{"Hide", Hide, "Passive"},
		{"Show after Hide", Show, "Active"},
	}
	for _, s := range steps {
		s.do()
		if got := status(); got != s.want {
			t.Fatalf("after %s Status is %q, want %s", s.name, got, s.want)
		}
	}
}

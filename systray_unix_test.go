//go:build (linux || freebsd || openbsd || netbsd) && !android

package systray

import (
	"bytes"
	"image"
	"image/png"
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

// pngBytes returns the PNG encoding of a size x size image.
func pngBytes(t *testing.T, size int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, size, size))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestIconPixmapPrecedence pins the IconPixmap property source: the
// multi-size list from SetIconPixmaps when set, the single image from SetIcon
// otherwise, and an empty array when neither is set — the empty array is what
// lets a host fall back to IconName. A later SetIcon clears the list, so the
// last setter wins.
func TestIconPixmapPrecedence(t *testing.T) {
	defer func() {
		SetIcon(nil)
		SetIconPixmaps(nil)
	}()

	pixmaps := func() []PX {
		return instance.createPropSpec()["org.kde.StatusNotifierItem"]["IconPixmap"].Value.([]PX)
	}

	if got := pixmaps(); len(got) != 0 {
		t.Fatalf("a fresh tray serves %d pixmaps, want 0 (the empty array allows the IconName fallback)", len(got))
	}

	small, large := pngBytes(t, 1), pngBytes(t, 2)
	SetIcon(small)
	if got := pixmaps(); len(got) != 1 {
		t.Fatalf("after SetIcon, %d pixmaps served, want 1", len(got))
	}

	SetIconPixmaps([][]byte{small, large})
	got := pixmaps()
	if len(got) != 2 {
		t.Fatalf("after SetIconPixmaps, %d pixmaps served, want 2", len(got))
	}
	if got[0].W != 1 || got[0].H != 1 || got[1].W != 2 || got[1].H != 2 {
		t.Errorf("pixmap dims = (%d,%d) and (%d,%d), want (1,1) and (2,2) — entries must be converted, not stored raw",
			got[0].W, got[0].H, got[1].W, got[1].H)
	}

	SetIcon(large)
	if got := pixmaps(); len(got) != 1 {
		t.Fatalf("a later SetIcon must clear the pixmap list (last setter wins), got %d", len(got))
	}
}

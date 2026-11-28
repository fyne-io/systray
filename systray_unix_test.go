//go:build (linux || freebsd || openbsd || netbsd) && !android

package systray

import (
	"bytes"
	"image"
	"image/color"
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

// TestArgbForImage pins the IconPixmap byte layout: ARGB32, network byte
// order, straight (non-premultiplied) alpha at full 8-bit range. The
// half-transparent white pixel is the case the old conversion corrupted:
// Color.RGBA() returns premultiplied 16-bit channels, so byte() both scaled
// the colour down and kept only its low bits, turning white into an invalid
// grey.
func TestArgbForImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x80})
	img.Set(1, 0, color.NRGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff})

	want := []byte{
		0x80, 0xff, 0xff, 0xff, // A R G B: straight alpha keeps the white
		0xff, 0x12, 0x34, 0x56,
	}
	if got := argbForImage(img); !bytes.Equal(got, want) {
		t.Errorf("argbForImage = % X, want % X", got, want)
	}
}

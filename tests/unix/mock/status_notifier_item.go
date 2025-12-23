package mock

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"sync"

	"github.com/godbus/dbus/v5"
)

type StatusNotifierItem struct {
	Service    string
	Sender     dbus.Sender
	Properties *ItemProperties
}

func newStatusNotifierItem(service string, sender dbus.Sender) *StatusNotifierItem {
	return &StatusNotifierItem{
		Service:    service,
		Sender:     sender,
		Properties: newItemProperties(),
	}
}

type Property[T any] struct {
	mu        sync.Mutex
	value     T
	ChangedCh chan T
}

func newProperty[T any](initial T) *Property[T] {
	return &Property[T]{
		mu:        sync.Mutex{},
		value:     initial,
		ChangedCh: make(chan T),
	}
}

func (p *Property[T]) Get() T {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.value
}

func (p *Property[T]) Set(newValue T) {
	p.mu.Lock()
	p.value = newValue
	p.mu.Unlock()
	sendNonBlockingToCh(p.ChangedCh, newValue)
}

type ItemProperties struct {
	Id            string
	ItemIsMenu    bool
	Title         *Property[string]
	IconThemePath *Property[string]
	IconPixmap    *Property[[]IconPixmap]
	Category      *Property[string]
	Status        *Property[string]
	ToolTip       *Property[ToolTip]
}

func newItemProperties() *ItemProperties {
	return &ItemProperties{
		Title:         newProperty(""),
		IconThemePath: newProperty(""),
		IconPixmap:    newProperty([]IconPixmap{}),
		Category:      newProperty(""),
		Status:        newProperty(""),
		ToolTip:       newProperty(ToolTip{}),
	}
}

type IconPixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

func IconPixmapFromData(data []byte) IconPixmap {
	if len(data) == 0 {
		return IconPixmap{}
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Panicf("Failed to read icon format %v", err)
	}

	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	d := make([]byte, w*h*4)
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			d[i] = byte(a)
			d[i+1] = byte(r)
			d[i+2] = byte(g)
			d[i+3] = byte(b)
			i += 4
		}
	}

	return IconPixmap{
		Width:  int32(img.Bounds().Dx()),
		Height: int32(img.Bounds().Dy()),
		Data:   d,
	}
}

type ToolTip struct {
	IconName string
	IconData []byte
	Title    string
	SubTitle string
}

func NewItemProperties(id string, isMenu bool, title string, iconThemePath string, iconPixmap IconPixmap, category string, status string, tooltip ToolTip) ItemProperties {
	return ItemProperties{
		Id:            id,
		ItemIsMenu:    isMenu,
		Title:         newProperty(title),
		IconThemePath: newProperty(iconThemePath),
		IconPixmap:    newProperty([]IconPixmap{iconPixmap}),
		Category:      newProperty(category),
		Status:        newProperty(status),
		ToolTip:       newProperty(tooltip),
	}
}

func DecodeToolTip(v dbus.Variant) (*ToolTip, error) {
	raw, ok := v.Value().([]any)
	if !ok || len(raw) != 4 {
		return nil, fmt.Errorf("unexpected ToolTip format: %T", v.Value())
	}

	tt := &ToolTip{}

	// Field 0: icon name
	if s, ok := raw[0].(string); ok {
		tt.IconName = s
	}

	// Field 1: icon data (often empty)
	switch data := raw[1].(type) {
	case []byte:
		tt.IconData = data
	case []interface{}:
		// Some implementations send []interface{}
		tt.IconData = nil
	}

	// Field 2: title
	if s, ok := raw[2].(string); ok {
		tt.Title = s
	}

	// Field 3: subtitle
	if s, ok := raw[3].(string); ok {
		tt.SubTitle = s
	}

	return tt, nil
}

func DecodeIconData(v dbus.Variant) ([]IconPixmap, error) {
	raw, ok := v.Value().([][]any)
	if !ok {
		return nil, fmt.Errorf("unexpected IconData format: %T", v.Value())
	}

	icons := make([]IconPixmap, 0, len(raw))

	for _, entry := range raw {
		if len(entry) < 2 {
			continue
		}

		ip := IconPixmap{}

		// width
		if w, ok := entry[0].(int32); ok {
			ip.Width = w
		}

		// height + pixels
		switch len(entry) {
		case 3:
			ip.Height = entry[1].(int32)
			ip.Data = entry[2].([]byte)
		case 2:
			// seen in practice: height omitted
			ip.Height = 0
			if b, ok := entry[1].([]byte); ok {
				ip.Data = b
			}
		}

		icons = append(icons, ip)
	}

	return icons, nil
}

func ParseItemProperties(itemProperties *ItemProperties, props map[string]dbus.Variant) error {
	if v, ok := props["ToolTip"]; ok {
		t, err := DecodeToolTip(v)
		if err != nil {
			return fmt.Errorf("failed to decode tooltip: %w", err)
		}
		itemProperties.ToolTip.Set(*t)
	}

	if v, ok := props["IconPixmap"]; ok {
		data, err := DecodeIconData(v)
		if err != nil {
			return fmt.Errorf("failed to decode icon pixmap: %w", err)
		}
		itemProperties.IconPixmap.Set(data)
	}

	if id, ok := props["Id"].Value().(string); ok {
		itemProperties.Id = id
	}
	if isMenu, ok := props["ItemIsMenu"].Value().(bool); ok {
		itemProperties.ItemIsMenu = isMenu
	}
	if title, ok := props["Title"].Value().(string); ok {
		itemProperties.Title.Set(title)
	}
	if iconThemePath, ok := props["IconThemePath"].Value().(string); ok {
		itemProperties.IconThemePath.Set(iconThemePath)
	}
	if category, ok := props["Category"].Value().(string); ok {
		itemProperties.Category.Set(category)
	}
	if status, ok := props["Status"].Value().(string); ok {
		itemProperties.Status.Set(status)
	}

	return nil
}

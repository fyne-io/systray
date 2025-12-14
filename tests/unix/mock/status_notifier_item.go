package mock

import (
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
)

type StatusNotifierItem struct {
	Service    string
	Sender     dbus.Sender
	Properties ItemProperties
}

type ItemProperties struct {
	Id            string
	ItemIsMenu    bool
	Title         string
	IconThemePath string
	IconPixmap    []IconPixmap
	Category      string
	Status        string
	ToolTip       ToolTip
}

type IconPixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

type ToolTip struct {
	IconName string
	IconData []byte
	Title    string
	SubTitle string
}

func NewItemProperties(id string, isMenu bool, title string, iconThemePath string, iconPixmap []IconPixmap, category string, status string, tooltip ToolTip) ItemProperties {
	return ItemProperties{
		Id:            id,
		ItemIsMenu:    isMenu,
		Title:         title,
		IconThemePath: iconThemePath,
		IconPixmap:    iconPixmap,
		Category:      category,
		Status:        status,
		ToolTip:       tooltip,
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

func ParseItemProperties(sender string, props map[string]dbus.Variant) (ItemProperties, error) {
	log.Println("Fetched properties for sender:", sender)
	for key, value := range props {
		log.Printf("  %s: %v\n", key, value)
	}

	tooltip, err := DecodeToolTip(props["ToolTip"])
	if err != nil {
		return ItemProperties{}, fmt.Errorf("failed to decode tooltip: %w", err)
	}

	iconData, err := DecodeIconData(props["IconPixmap"])
	if err != nil {
		return ItemProperties{}, fmt.Errorf("failed to decode icon pixmap: %w", err)
	}

	return NewItemProperties(
		props["Id"].Value().(string),
		props["ItemIsMenu"].Value().(bool),
		props["Title"].Value().(string),
		props["IconThemePath"].Value().(string),
		iconData,
		props["Category"].Value().(string),
		props["Status"].Value().(string),
		*tooltip,
	), nil
}

//go:build windows
// +build windows

package icon

import _ "embed" // embed is used only here

var (
	// Data is the icon data
	//go:embed iconwin.ico
	Data []byte
)

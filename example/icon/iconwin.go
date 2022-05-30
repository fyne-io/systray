//go:build windows
// +build windows

package icon

import _ "embed" // embed is used only here

var (
	// Data is the icon data
	//go:embed iconwin.ico
	Data []byte
	// Data is the turned icon data
	//go:embed iconwin1.ico
	Data1 []byte
)

//go:build linux || darwin
// +build linux darwin

package icon

import _ "embed" // embed is used only here

var (
	// Data is the icon data
	//go:embed icon.png
	Data []byte

	// Data1 is the turned icon data
	//go:embed icon1.png
	Data1 []byte
)

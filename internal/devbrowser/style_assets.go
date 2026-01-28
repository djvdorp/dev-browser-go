package devbrowser

import _ "embed"

//go:embed style_assets/style_capture.js
var styleCaptureJS string

func styleCaptureScript() string {
	return styleCaptureJS
}

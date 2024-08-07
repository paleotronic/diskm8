package main

import (
	"encoding/base64"
	"os"
)

var text = `ICBfXyAgICAgICAgICAgICAgIF9fICAgICAgICAgICAgICAgICAgX18gICAgX18gICAgIAogL1wgXCAgX18gICAgICAgICAvXCBcICAgICAgLydcXy9gXCAgLydfIGBcIC9cIFwgICAgCiBcX1wgXC9cX1wgICAgX19fX1wgXCBcLydcIC9cICAgICAgXC9cIFxMXCBcXCBcIFwgICAKIC8nX2AgXC9cIFwgIC8nLF9fXFwgXCAsIDwgXCBcIFxfX1wgXC9fPiBfIDxfXCBcIFwgIAovXCBcTFwgXCBcIFwvXF9fLCBgXFwgXCBcXGBcXCBcIFxfL1wgXC9cIFxMXCBcXCBcX1wgClwgXF9fXyxfXCBcX1wvXF9fX18vIFwgXF9cIFxfXCBcX1xcIFxfXCBcX19fXy8gXC9cX1wKIFwvX18sXyAvXC9fL1wvX19fLyAgIFwvXy9cL18vXC9fLyBcL18vXC9fX18vICAgXC9fLwogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAK`

func banner() {

	t, _ := base64.StdEncoding.DecodeString(text)

	os.Stderr.WriteString(string(t) + "\r\n")
	os.Stderr.WriteString("(c) 2015 - 2024 Paleotronic.com\n\n")
	os.Stderr.WriteString("type 'help' to see commands\n\n")

}

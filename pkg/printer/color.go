/*
The MIT License (MIT)

Copyright (c) 2015 Takashi Kokubun
Copyright (c) 2022 Arthur Skowronek

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package printer

import (
	"fmt"
)

const (
	// No color
	NoColor uint16 = 1 << 15
)

const (
	// Foreground colors for ColorScheme.
	_ uint16 = iota | NoColor
	Black
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
	bitsForeground       = 0
	maskForeground       = 0xf
	ansiForegroundOffset = 30 - 1
)

const (
	// Background colors for ColorScheme.
	_ uint16 = iota<<bitsBackground | NoColor
	BackgroundBlack
	BackgroundRed
	BackgroundGreen
	BackgroundYellow
	BackgroundBlue
	BackgroundMagenta
	BackgroundCyan
	BackgroundWhite
	bitsBackground       = 4
	maskBackground       = 0xf << bitsBackground
	ansiBackgroundOffset = 40 - 1
)

const (
	// Bold flag for ColorScheme.
	Bold     uint16 = 1<<bitsBold | NoColor
	bitsBold        = 8
	maskBold        = 1 << bitsBold
	ansiBold        = 1
)

var (
	DefaultScheme = &ColorScheme{
		Bool:            Cyan | Bold,
		Integer:         Blue | Bold,
		Float:           Magenta | Bold,
		String:          Red,
		StringQuotation: Red | Bold,
		EscapedChar:     Magenta | Bold,
		FieldName:       Yellow,
		PointerAdress:   Blue | Bold,
		Nil:             Cyan | Bold,
		Time:            Blue | Bold,
		StructName:      Green,
		ObjectLength:    Blue,
	}
)

type ColorField int

const (
	_ ColorField = iota
	BoolColor
	IntegerColor
	FloatColor
	StringColor
	StringQuotationColor
	EscapedCharColor
	FieldNameColor
	PointerAdressColor
	NilColor
	TimeColor
	StructNameColor
	ObjectLengthColor
)

type ColorScheme struct {
	Bool            uint16
	Integer         uint16
	Float           uint16
	String          uint16
	StringQuotation uint16
	EscapedChar     uint16
	FieldName       uint16
	PointerAdress   uint16
	Nil             uint16
	Time            uint16
	StructName      uint16
	ObjectLength    uint16
}

func (s ColorScheme) Get(field ColorField) uint16 {
	switch field {
	case BoolColor:
		return s.Bool
	case IntegerColor:
		return s.Integer
	case FloatColor:
		return s.Float
	case StringColor:
		return s.String
	case StringQuotationColor:
		return s.StringQuotation
	case EscapedCharColor:
		return s.EscapedChar
	case FieldNameColor:
		return s.FieldName
	case PointerAdressColor:
		return s.PointerAdress
	case NilColor:
		return s.Nil
	case TimeColor:
		return s.Time
	case StructNameColor:
		return s.StructName
	case ObjectLengthColor:
		return s.ObjectLength
	}
	panic("bad field value")
}

func ColorizeText(text string, color uint16) string {
	foreground := color & maskForeground >> bitsForeground
	background := color & maskBackground >> bitsBackground
	bold := color & maskBold

	if foreground == 0 && background == 0 && bold == 0 {
		return text
	}

	modBold := ""
	modForeground := ""
	modBackground := ""

	if bold > 0 {
		modBold = "\033[1m"
	}
	if foreground > 0 {
		modForeground = fmt.Sprintf("\033[%dm", foreground+ansiForegroundOffset)
	}
	if background > 0 {
		modBackground = fmt.Sprintf("\033[%dm", background+ansiBackgroundOffset)
	}

	return fmt.Sprintf("%s%s%s%s\033[0m", modForeground, modBackground, modBold, text)
}

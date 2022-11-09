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
	"bytes"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/mattn/go-colorable"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	indentWidth = 4
)

var (
	DefaultOut = colorable.NewColorableStdout()
)

func NewPrinter(
	object interface{},
	colorScheme *ColorScheme,
	decimalUint bool,
	exportedOnly bool,
	thousandsSeparator bool,
) *Printer {
	buffer := bytes.NewBufferString("")
	tw := new(tabwriter.Writer)
	tw.Init(buffer, indentWidth, 0, 1, ' ', 0)

	printer := &Printer{
		Buffer:             buffer,
		tw:                 tw,
		depth:              0,
		value:              reflect.ValueOf(object),
		visited:            map[uintptr]bool{},
		colorScheme:        colorScheme,
		decimalUint:        decimalUint,
		exportedOnly:       exportedOnly,
		thousandsSeparator: thousandsSeparator,
	}

	if thousandsSeparator {
		printer.localizedPrinter = message.NewPrinter(language.English)
	}

	return printer
}

type Printer struct {
	*bytes.Buffer
	tw                 *tabwriter.Writer
	depth              int
	value              reflect.Value
	visited            map[uintptr]bool
	colorScheme        *ColorScheme
	decimalUint        bool
	exportedOnly       bool
	thousandsSeparator bool
	localizedPrinter   *message.Printer
}

func (p *Printer) String() string {
	p.tw.Flush()
	return p.Buffer.String()
}

func (p *Printer) IsColoringEnabled() bool                  { return p.colorScheme != nil }
func (p *Printer) Print(text string)                        { fmt.Fprint(p.tw, text) }
func (p *Printer) Println(text string)                      { p.Print(text + "\n") }
func (p *Printer) IndentPrint(text string)                  { p.Print(p.Indent() + text) }
func (p *Printer) ColorPrint(text string, color ColorField) { p.Print(p.Colorize(text, color)) }

func (p *Printer) Printf(format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	p.Print(text)
}

func (p *Printer) IndentPrintf(format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	p.IndentPrint(text)
}

func (p *Printer) PrintKeyValue(k, v any) {
	p.IndentPrintf("%s:\t%s,\n", p.Format(k), p.Format(v))
}

func (p *Printer) PrintFieldValue(k string, v any) {
	colorizedFieldName := p.Colorize(k, FieldNameColor)
	p.IndentPrintf("%s:\t%s\n", colorizedFieldName, p.Format(v))
}

func (p *Printer) printString() {
	quoted := strconv.Quote(p.value.String())
	quoted = quoted[1 : len(quoted)-1]

	p.ColorPrint(`"`, StringQuotationColor)
	for len(quoted) > 0 {
		pos := strings.IndexByte(quoted, '\\')
		if pos == -1 {
			p.ColorPrint(quoted, StringColor)
			break
		}
		if pos != 0 {
			p.ColorPrint(quoted[0:pos], StringColor)
		}

		n := 1
		switch quoted[pos+1] {
		case 'x': // "\x00"
			n = 3
		case 'u': // "\u0000"
			n = 5
		case 'U': // "\U00000000"
			n = 9
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9': // "\000"
			n = 3
		}
		p.ColorPrint(quoted[pos:pos+n+1], EscapedCharColor)
		quoted = quoted[pos+n+1:]
	}
	p.ColorPrint(`"`, StringQuotationColor)
}

func (p *Printer) printMap() {
	if p.value.Len() == 0 {
		p.Printf("%s{}", p.typeString())
		return
	}

	if p.visited[p.value.Pointer()] {
		p.Printf("%s{...}", p.typeString())
		return
	}
	p.visited[p.value.Pointer()] = true

	p.Printf("%s{\n", p.typeString())

	p.indented(func() {
		value := sortMap(p.value)
		for i := 0; i < value.Len(); i++ {
			p.IndentPrintf("%s:\t%s,\n", p.Format(value.keys[i]), p.Format(value.values[i]))
		}
	})
	p.IndentPrint("}")
}

func (p *Printer) printStruct() {
	if p.value.CanInterface() {
		if p.value.Type().String() == "time.Time" && p.value.Type().PkgPath() == "time" {
			p.printTime()
			return
		} else if p.value.Type().String() == "big.Int" {
			bigInt := p.value.Interface().(big.Int)
			p.Print(p.Colorize(bigInt.String(), IntegerColor))
			return
		} else if p.value.Type().String() == "big.Float" {
			bigFloat := p.value.Interface().(big.Float)
			p.Print(p.Colorize(bigFloat.String(), FloatColor))
			return
		}
	}

	var fields []int
	for i := 0; i < p.value.NumField(); i++ {
		field := p.value.Type().Field(i)
		value := p.value.Field(i)
		// ignore unexported if needed
		if p.exportedOnly && field.PkgPath != "" {
			continue
		}
		// ignore fields if zero value, or explicitly set
		if tag := field.Tag.Get("pp"); tag != "" {
			parts := strings.Split(tag, ",")
			if len(parts) == 2 && parts[1] == "omitempty" && valueIsZero(value) {
				continue
			}
			if parts[0] == "-" {
				continue
			}
		}
		fields = append(fields, i)
	}

	if len(fields) == 0 {
		p.Print(p.typeString() + "{}")
		return
	}

	p.Println(p.typeString() + "{")
	p.indented(func() {
		for _, i := range fields {
			field := p.value.Type().Field(i)
			value := p.value.Field(i)

			fieldName := field.Name
			if tag := field.Tag.Get("pp"); tag != "" {
				tagName := strings.Split(tag, ",")
				if tagName[0] != "" {
					fieldName = tagName[0]
				}
			}

			colorizedFieldName := p.Colorize(fieldName, FieldNameColor)
			p.IndentPrintf("%s:\t%s,\n", colorizedFieldName, p.Format(value))
		}
	})
	p.IndentPrint("}")
}

func (p *Printer) printTime() {
	tm := p.value.Interface().(time.Time)
	p.Printf(
		"%s-%s-%s %s:%s:%s %s",
		p.Colorize(strconv.Itoa(tm.Year()), TimeColor),
		p.Colorize(fmt.Sprintf("%02d", tm.Month()), TimeColor),
		p.Colorize(fmt.Sprintf("%02d", tm.Day()), TimeColor),
		p.Colorize(fmt.Sprintf("%02d", tm.Hour()), TimeColor),
		p.Colorize(fmt.Sprintf("%02d", tm.Minute()), TimeColor),
		p.Colorize(fmt.Sprintf("%02d", tm.Second()), TimeColor),
		p.Colorize(tm.Location().String(), TimeColor),
	)
}

func (p *Printer) printSlice() {
	if p.value.Kind() == reflect.Slice && p.value.IsNil() {
		p.Printf("%s(%s)", p.typeString(), p.nil())
		return
	}
	if p.value.Len() == 0 {
		p.Printf("%s{}", p.typeString())
		return
	}

	if p.value.Kind() == reflect.Slice {
		if p.visited[p.value.Pointer()] {
			// Stop travarsing cyclic reference
			p.Printf("%s{...}", p.typeString())
			return
		}
		p.visited[p.value.Pointer()] = true
	}

	// Fold a large buffer
	if p.value.Len() > 1024 {
		p.Printf("%s{...}", p.typeString())
		return
	}

	var groupSize int
	switch p.value.Type().Elem().Kind() {
	case reflect.Uint8:
		groupSize = 16
	case reflect.Uint16:
		groupSize = 8
	case reflect.Uint32:
		groupSize = 8
	case reflect.Uint64:
		groupSize = 4
	case reflect.String:
		groupSize = 36 / stringGroupSize(p.value.Interface())
	}

	if p.value.Len() < groupSize {
		p.Print("{")
		p.Printf("%s", p.Format(p.value.Index(0)))
		for i := 1; i < p.value.Len(); i++ {
			p.Printf(", %s", p.Format(p.value.Index(i)))
		}
		p.Print("}")
	} else {
		p.Println("{")
		p.indented(func() {
			if groupSize > 0 {
				for i := 0; i < p.value.Len(); i++ {
					// Indent for new group
					if i%groupSize == 0 {
						p.Print(p.Indent())
					}
					// slice element
					p.Printf("%s,", p.Format(p.value.Index(i)))
					// space or newline
					if (i+1)%groupSize == 0 || i+1 == p.value.Len() {
						p.Print("\n")
					} else {
						p.Print(" ")
					}
				}
			} else {
				for i := 0; i < p.value.Len(); i++ {
					p.IndentPrintf("%s,\n", p.Format(p.value.Index(i)))
				}
			}
		})
		p.IndentPrint("}")
	}
}

func stringGroupSize(i any) (max int) {
	for _, s := range i.([]string) {
		if l := len(s); l > max {
			max = l
		}
	}
	return
}

func (p *Printer) printInterface() {
	e := p.value.Elem()
	if e.Kind() == reflect.Invalid {
		p.Print(p.nil())
	} else if e.IsValid() {
		p.Print(p.Format(e))
	} else {
		p.Printf("%s(%s)", p.typeString(), p.nil())
	}
}

func (p *Printer) printPtr() {
	if p.visited[p.value.Pointer()] {
		p.Printf("&%s{...}", p.elemTypeString())
		return
	}
	if p.value.Pointer() != 0 {
		p.visited[p.value.Pointer()] = true
	}

	if p.value.Elem().IsValid() {
		p.Printf("&%s", p.Format(p.value.Elem()))
	} else {
		p.Printf("(%s)(%s)", p.typeString(), p.nil())
	}
}

func (p *Printer) pointerAddr() string {
	return p.Colorize(fmt.Sprintf("%#v", p.value.Pointer()), PointerAdressColor)
}

func (p *Printer) typeString() string {
	return p.colorizeType(p.value.Type().String())
}

func (p *Printer) elemTypeString() string {
	return p.colorizeType(p.value.Elem().Type().String())
}

func (p *Printer) colorizeType(t string) string {
	prefix := ""

	if p.matchRegexp(t, `^\[\].+$`) {
		prefix = "[]"
		t = t[2:]
	}

	if p.matchRegexp(t, `^\[\d+\].+$`) {
		num := regexp.MustCompile(`\d+`).FindString(t)
		prefix = fmt.Sprintf("[%s]", p.Colorize(num, ObjectLengthColor))
		t = t[2+len(num):]
	}

	if p.matchRegexp(t, `^[^\.]+\.[^\.]+$`) {
		ts := strings.Split(t, ".")
		t = fmt.Sprintf("%s.%s", ts[0], p.Colorize(ts[1], StructNameColor))
	} else {
		t = p.Colorize(t, StructNameColor)
	}
	return prefix + t
}

func (p *Printer) matchRegexp(text, exp string) bool {
	return regexp.MustCompile(exp).MatchString(text)
}

func (p *Printer) indented(proc func()) {
	p.depth++

	proc()

	p.depth--
}

func (p *Printer) fmtOrLocalizedSprintf(format string, a ...interface{}) string {
	if p.localizedPrinter == nil {
		return fmt.Sprintf(format, a...)
	}

	return p.localizedPrinter.Sprintf(format, a...)
}

func (p *Printer) raw() string {
	// Some value causes panic when Interface() is called.
	switch p.value.Kind() {
	case reflect.Bool:
		return fmt.Sprintf("%#v", p.value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return p.fmtOrLocalizedSprintf("%v", p.value.Int())
	case reflect.Uint, reflect.Uintptr:
		if p.decimalUint {
			return p.fmtOrLocalizedSprintf("%d", p.value.Uint())
		} else {
			return fmt.Sprintf("%#v", p.value.Uint())
		}
	case reflect.Uint8:
		if p.decimalUint {
			return fmt.Sprintf("%d", p.value.Uint())
		} else {
			return fmt.Sprintf("0x%02x", p.value.Uint())
		}
	case reflect.Uint16:
		if p.decimalUint {
			return p.fmtOrLocalizedSprintf("%d", p.value.Uint())
		} else {
			return fmt.Sprintf("0x%04x", p.value.Uint())
		}
	case reflect.Uint32:
		if p.decimalUint {
			return p.fmtOrLocalizedSprintf("%d", p.value.Uint())
		} else {
			return fmt.Sprintf("0x%08x", p.value.Uint())
		}
	case reflect.Uint64:
		if p.decimalUint {
			return p.fmtOrLocalizedSprintf("%d", p.value.Uint())
		} else {
			return fmt.Sprintf("0x%016x", p.value.Uint())
		}
	case reflect.Float32, reflect.Float64:
		return p.fmtOrLocalizedSprintf("%f", p.value.Float())
	case reflect.Complex64, reflect.Complex128:
		return fmt.Sprintf("%#v", p.value.Complex())
	default:
		return fmt.Sprintf("%#v", p.value.Interface())
	}
}

func (p *Printer) nil() string {
	return p.Colorize("nil", NilColor)
}

func (p *Printer) Colorize(text string, color ColorField) string {
	if p.IsColoringEnabled() {
		return ColorizeText(text, p.colorScheme.Get(color))
	} else {
		return text
	}
}

func (p *Printer) Format(object interface{}) string {
	pp := NewPrinter(object, p.colorScheme, p.decimalUint, p.exportedOnly, p.thousandsSeparator)
	if value, ok := object.(reflect.Value); ok {
		pp.value = value
	}
	pp.depth = p.depth
	pp.visited = p.visited

	if f, ok := pp.value.Interface().(interface{ PrettyPrint(*Printer) }); ok {
		f.PrettyPrint(pp)
	} else {
		switch pp.value.Kind() {
		case reflect.Bool:
			pp.ColorPrint(pp.raw(), BoolColor)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Uintptr, reflect.Complex64, reflect.Complex128:
			pp.ColorPrint(pp.raw(), IntegerColor)
		case reflect.Float32, reflect.Float64:
			pp.ColorPrint(pp.raw(), FloatColor)
		case reflect.String:
			pp.printString()
		case reflect.Map:
			pp.printMap()
		case reflect.Struct:
			pp.printStruct()
		case reflect.Array, reflect.Slice:
			pp.printSlice()
		case reflect.Chan:
			pp.Printf("(%s)(%s)", pp.typeString(), pp.pointerAddr())
		case reflect.Interface:
			pp.printInterface()
		case reflect.Ptr:
			pp.printPtr()
		case reflect.Func:
			pp.Printf("%s {...}", pp.typeString())
		case reflect.UnsafePointer:
			pp.Printf("%s(%s)", pp.typeString(), pp.pointerAddr())
		case reflect.Invalid:
			pp.Print(pp.nil())
		default:
			pp.Print(pp.raw())
		}
	}
	return pp.String()
}

func (p *Printer) Indent() string {
	return strings.Repeat("\t", p.depth)
}

// valueIsZero reports whether v is the zero value for its type.
// It returns false if the argument is invalid.
// This is a copy paste of reflect#IsZero from go1.15. It is not present before go1.13 (source: https://golang.org/doc/go1.13#library)
// 	source: https://golang.org/src/reflect/value.go?s=34297:34325#L1090
// This will need to be updated for new types or the decision should be made to drop support for Go version pre go1.13
func valueIsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return math.Float64bits(v.Float()) == 0
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !valueIsZero(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	case reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !valueIsZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		// this is the only difference between stdlib reflect#IsZero and this function. We're not going to
		// panic on the default cause, even
		return false
	}
}

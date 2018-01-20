package disk

import (
	"fmt"
	"strconv"
	"strings"

	"regexp"
)

//import "strings"

var ApplesoftTokens = map[int]string{
	0x80: "END",
	0x81: "FOR",
	0x82: "NEXT",
	0x83: "DATA",
	0x84: "INPUT",
	0x85: "DEL",
	0x86: "DIM",
	0x87: "READ",
	0x88: "GR",
	0x89: "TEXT",
	0x8A: "PR#",
	0x8B: "IN#",
	0x8C: "CALL",
	0x8D: "PLOT",
	0x8E: "HLIN",
	0x8F: "VLIN",
	0x90: "HGR2",
	0x91: "HGR",
	0x92: "HCOLOR=",
	0x93: "HPLOT",
	0x94: "DRAW",
	0x95: "XDRAW",
	0x96: "HTAB",
	0x97: "HOME",
	0x98: "ROT=",
	0x99: "SCALE=",
	0x9A: "SHLOAD",
	0x9B: "TRACE",
	0x9C: "NOTRACE",
	0x9D: "NORMAL",
	0x9E: "INVERSE",
	0x9F: "FLASH",
	0xA0: "COLOR=",
	0xA1: "POP",
	0xA2: "VTAB",
	0xA3: "HIMEM:",
	0xA4: "LOMEM:",
	0xA5: "ONERR",
	0xA6: "RESUME",
	0xA7: "RECALL",
	0xA8: "STORE",
	0xA9: "SPEED=",
	0xAA: "LET",
	0xAB: "GOTO",
	0xAC: "RUN",
	0xAD: "IF",
	0xAE: "RESTORE",
	0xAF: "&",
	0xB0: "GOSUB",
	0xB1: "RETURN",
	0xB2: "REM",
	0xB3: "STOP",
	0xB4: "ON",
	0xB5: "WAIT",
	0xB6: "LOAD",
	0xB7: "SAVE",
	0xB8: "DEF",
	0xB9: "POKE",
	0xBA: "PRINT",
	0xBB: "CONT",
	0xBC: "LIST",
	0xBD: "CLEAR",
	0xBE: "GET",
	0xBF: "NEW",
	0xC0: "TAB(",
	0xC1: "TO",
	0xC2: "FN",
	0xC3: "SPC(",
	0xC4: "THEN",
	0xC5: "AT",
	0xC6: "NOT",
	0xC7: "STEP",
	0xC8: "+",
	0xC9: "-",
	0xCA: "*",
	0xCB: "/",
	0xCC: "^",
	0xCD: "AND",
	0xCE: "OR",
	0xCF: ">",
	0xD0: "=",
	0xD1: "<",
	0xD2: "SGN",
	0xD3: "INT",
	0xD4: "ABS",
	0xD5: "USR",
	0xD6: "FRE",
	0xD7: "SCRN(",
	0xD8: "PDL",
	0xD9: "POS",
	0xDA: "SQR",
	0xDB: "RND",
	0xDC: "LOG",
	0xDD: "EXP",
	0xDE: "COS",
	0xDF: "SIN",
	0xE0: "TAN",
	0xE1: "ATN",
	0xE2: "PEEK",
	0xE3: "LEN",
	0xE4: "STR$",
	0xE5: "VAL",
	0xE6: "ASC",
	0xE7: "CHR$",
	0xE8: "LEFT$",
	0xE9: "RIGHT$",
	0xEA: "MID$",
}

var ApplesoftReverse map[string]int
var IntegerReverse map[string]int

func init() {
	ApplesoftReverse = make(map[string]int)
	IntegerReverse = make(map[string]int)
	for k, v := range ApplesoftTokens {
		ApplesoftReverse[v] = k
	}
	for k, v := range IntegerTokens {
		IntegerReverse[v] = k
	}

	tst()

}

var IntegerTokens = map[int]string{
	0x00: "HIMEM:",
	0x02: "_",
	0x03: ":",
	0x04: "LOAD",
	0x05: "SAVE",
	0x06: "CON",
	0x07: "RUN",
	0x08: "RUN",
	0x09: "DEL",
	0x0A: ",",
	0x0B: "NEW",
	0x0C: "CLR",
	0x0D: "AUTO",
	0x0E: ",",
	0x0F: "MAN",
	0x10: "HIMEM:",
	0x11: "LOMEM:",
	0x12: "+",
	0x13: "-",
	0x14: "*",
	0x15: "/",
	0x16: "=",
	0x17: "#",
	0x18: ">=",
	0x19: ">",
	0x1A: "<=",
	0x1B: "<>",
	0x1C: "<",
	0x1D: "AND",
	0x1E: "OR",
	0x1F: "MOD",
	0x20: "^",
	0x21: "+",
	0x22: "(",
	0x23: ",",
	0x24: "THEN",
	0x25: "THEN",
	0x26: ",",
	0x27: ",",
	0x28: "\"",
	0x29: "\"",
	0x2A: "(",
	0x2B: "!",
	0x2C: "!",
	0x2D: "(",
	0x2E: "PEEK",
	0x2F: "RND",
	0x30: "SGN",
	0x31: "ABS",
	0x32: "PDL",
	0x33: "RNDX",
	0x34: "(",
	0x35: "+",
	0x36: "-",
	0x37: "NOT",
	0x38: "(",
	0x39: "=",
	0x3A: "#",
	0x3B: "LEN(",
	0x3C: "ASC(",
	0x3D: "SCRN(",
	0x3E: ",",
	0x3F: "(",
	0x40: "$",
	0x41: "$",
	0x42: "(",
	0x43: ",",
	0x44: ",",
	0x45: ";",
	0x46: ";",
	0x47: ";",
	0x48: ",",
	0x49: ",",
	0x4A: ",",
	0x4B: "TEXT",
	0x4C: "GR",
	0x4D: "CALL",
	0x4E: "DIM",
	0x4F: "DIM",
	0x50: "TAB",
	0x51: "END",
	0x52: "INPUT",
	0x53: "INPUT",
	0x54: "INPUT",
	0x55: "FOR",
	0x56: "=",
	0x57: "TO",
	0x58: "STEP",
	0x59: "NEXT",
	0x5A: ",",
	0x5B: "RETURN",
	0x5C: "GOSUB",
	0x5D: "REM",
	0x5E: "LET",
	0x5F: "GOTO",
	0x60: "IF",
	0x61: "PRINT",
	0x62: "PRINT",
	0x63: "PRINT",
	0x64: "POKE",
	0x65: ",",
	0x66: "COLOR=",
	0x67: "PLOT",
	0x68: ",",
	0x69: "HLIN",
	0x6A: ",",
	0x6B: "AT",
	0x6C: "VLIN",
	0x6D: ",",
	0x6E: "AT",
	0x6F: "VTAB",
	0x70: "=",
	0x71: "=",
	0x72: ")",
	0x73: ")",
	0x74: "LIST",
	0x75: ",",
	0x76: "LIST",
	0x77: "POP",
	0x78: "NODSP",
	0x79: "DSP",
	0x7A: "NOTRACE",
	0x7B: "DSP",
	0x7C: "DSP",
	0x7D: "TRACE",
	0x7E: "PR#",
	0x7F: "IN#",
}

func Read16(srcptr, length *int, buffer []byte) int {

	// if *length < 2 {
	// 	*srcptr += *length
	// 	*length = 0
	// 	return 0
	// }
	//fmt.Printf("-- srcptr=%d, length=%d, len(buffer)=%d\n", *srcptr, *length, len(buffer))

	v := int(buffer[*srcptr]) + 256*int(buffer[*srcptr+1])

	*srcptr += 2
	*length -= 2

	return v

}

func Read8(srcptr, length *int, buffer []byte) byte {

	// if *length < 1 {
	// 	*srcptr += *length
	// 	*length = 0
	// 	return 0
	// }
	//fmt.Printf("-- srcptr=%d, length=%d, len(buffer)=%d\n", *srcptr, *length, len(buffer))

	v := buffer[*srcptr]

	*srcptr += 1
	*length -= 1

	return v

}

func StripText(b []byte) []byte {
	c := make([]byte, len(b))
	for i, v := range b {
		c[i] = v & 127
	}
	return c
}

func ApplesoftDetoks(data []byte) []byte {

	//var baseaddr int = 0x801
	var srcptr int = 0x00
	var length int = len(data)
	var out []byte = make([]byte, 0)

	if length < 2 {
		// not enough here
		return []byte("\r\n")
	}

	for length > 0 {

		var nextAddr int
		var lineNum int
		var inQuote bool = false
		var inRem bool = false

		if length < 2 {
			break
		}

		nextAddr = Read16(&srcptr, &length, data)

		if nextAddr == 0 {
			break
		}

		/* output line number */

		if length < 2 {
			break
		}

		lineNum = Read16(&srcptr, &length, data)
		ln := fmt.Sprintf("%d", lineNum)

		out = append(out, []byte(" "+ln+" ")...)

		if length == 0 {
			break
		}

		var t byte = Read8(&srcptr, &length, data)

		for t != 0 && length > 0 {
			// process token
			if t&0x80 != 0 {
				/* token */
				tokstr, ok := ApplesoftTokens[int(t)]
				if ok {
					out = append(out, []byte(" "+tokstr+" ")...)
				} else {
					out = append(out, []byte(" ERROR ")...)
				}
				if t == 0xb2 {
					inRem = true
				}
			} else {
				/* simple character */
				r := rune(t)
				if r == '"' && !inRem {
					if !inQuote {
						out = append(out, t)
					} else {
						out = append(out, t)
					}
					inQuote = !inQuote
				} else if r == ':' && !inRem && !inQuote {
					out = append(out, t)
				} else if inRem && (r == '\r' || r == '\n') {
					out = append(out, []byte("*")...)
				} else {
					out = append(out, t)
				}
			}

			// Advance
			t = Read8(&srcptr, &length, data)
		}

		out = append(out, []byte("\r\n")...)

		inQuote, inRem = false, false

		if length == 0 {
			break
		}

	}

	//fmt.Println(string(out))

	return out

}

func IntegerDetoks(data []byte) []byte {

	var srcptr int = 0x00
	var length int = len(data)
	var out []byte = make([]byte, 0)

	if length < 2 {
		// not enough here
		return []byte("\r\n")
	}

	for length > 0 {

		// starting state for line
		var lineLen byte
		var lineNum int
		var trailingSpace bool
		var newTrailingSpace bool = false

		// read the line length
		lineLen = Read8(&srcptr, &length, data)

		if lineLen == 0 {
			break // zero length line found
		}

		// read line number
		lineNum = Read16(&srcptr, &length, data)
		out = append(out, []byte(fmt.Sprintf("%d ", lineNum))...)

		// now process line
		var t byte
		t = Read8(&srcptr, &length, data)
		for t != 0x01 && length > 0 {
			if t == 0x03 {
				out = append(out, []byte(" :")...)
				t = Read8(&srcptr, &length, data)
			} else if t == 0x28 {
				/* start of quoted text */
				out = append(out, 34)

				t = Read8(&srcptr, &length, data)
				for t != 0x29 && length > 0 {
					out = append(out, t&0x7f)
					t = Read8(&srcptr, &length, data)
				}
				if t != 0x29 {
					break
				}

				out = append(out, 34)

				t = Read8(&srcptr, &length, data)
			} else if t == 0x5d {
				/* start of REM statement, run to EOL */
				if trailingSpace {
					out = append(out, 32)
				}
				out = append(out, []byte("REM ")...)

				t = Read8(&srcptr, &length, data)
				for t != 0x01 && length > 0 {
					out = append(out, t&0x7f)
					t = Read8(&srcptr, &length, data)
				}
				if t != 0x01 {
					break
				}
			} else if t >= 0xb0 && t <= 0xb9 {
				/* start of integer constant */
				if length < 2 {
					break
				}
				val := Read16(&srcptr, &length, data)
				out = append(out, []byte(fmt.Sprintf("%d", val))...)
				t = Read8(&srcptr, &length, data)
			} else if t >= 0xc1 && t <= 0xda {
				/* start of variable name */
				for (t >= 0xc1 && t <= 0xda) || (t >= 0xb0 && t <= 0xb9) {
					/* note no RTF-escaped chars in this range */
					out = append(out, t&0x7f)
					t = Read8(&srcptr, &length, data)
				}
			} else if t < 0x80 {
				/* found a token; try to get the whitespace right */
				/* (maybe should've left whitespace on the ends of tokens
				   that are always followed by whitespace...?) */
				token, _ := IntegerTokens[int(t)]
				if token[0] >= 0x21 && token[0] <= 0x3f || t < 0x12 {
					/* does not need leading space */
					out = append(out, []byte(token)...)
				} else {
					/* needs leading space; combine with prev if it exists */
					if trailingSpace {
						out = append(out, []byte(token)...)
					} else {
						out = append(out, []byte(" "+token)...)
					}
					out = append(out, 32)
				}
				if token[len(token)-1] == 32 {
					newTrailingSpace = true
				}
				t = Read8(&srcptr, &length, data)
			} else {
				/* should not happen */
				t = Read8(&srcptr, &length, data)
			}

			trailingSpace = newTrailingSpace
			newTrailingSpace = false
		}

		if t != 0x01 && length > 0 {
			break // must have failed
		}

		// ok, new line
		out = append(out, []byte("\r\n")...)

	}

	return out

}

func breakingChar(ch rune) bool {
	return ch == '(' || ch == ')' || ch == '.' || ch == ',' || ch == ';' || ch == ':' || ch == ' '
}

func ApplesoftTokenize(lines []string) []byte {

	start := 0x801
	currAddr := start

	buffer := make([]byte, 0)

	for _, l := range lines {

		l = strings.Trim(l, "\r")
		if l == "" {
			continue
		}

		chunk := ""
		inqq := false
		tmp := strings.SplitN(l, " ", 2)
		ln, _ := strconv.Atoi(tmp[0])
		rest := strings.Trim(tmp[1], " ")

		linebuffer := make([]byte, 4)

		// LINE NUMBER
		linebuffer[0x02] = byte(ln & 0xff)
		linebuffer[0x03] = byte(ln / 0x100)

		// PROCESS LINE
		for _, ch := range rest {

			switch {
			case inqq && ch != '"':
				linebuffer = append(linebuffer, byte(ch))
				continue
			case ch == '"':
				linebuffer = append(linebuffer, byte(ch))
				inqq = !inqq
				continue
			case !inqq && breakingChar(ch):
				linebuffer = append(linebuffer, []byte(chunk)...)
				chunk = ""
				linebuffer = append(linebuffer, byte(ch))
				continue
			}

			chunk += string(ch)
			code, ok := ApplesoftReverse[strings.ToUpper(chunk)]
			if ok {
				linebuffer = append(linebuffer, byte(code))
				chunk = ""
			}
		}
		if chunk != "" {
			linebuffer = append(linebuffer, []byte(chunk)...)
		}

		// ENDING ZERO BYTE
		linebuffer = append(linebuffer, 0x00)

		nextAddr := currAddr + len(linebuffer)
		linebuffer[0x00] = byte(nextAddr & 0xff)
		linebuffer[0x01] = byte(nextAddr / 0x100)
		currAddr = nextAddr

		buffer = append(buffer, linebuffer...)
	}

	buffer = append(buffer, 0x00, 0x00)

	return buffer

}

var reInt = regexp.MustCompile("^(-?[0-9]+)$")

func isInt(s string) (bool, [3]byte) {
	if reInt.MatchString(s) {

		m := reInt.FindAllStringSubmatch(s, -1)
		i, _ := strconv.ParseInt(m[0][1], 10, 32)
		return true, [3]byte{0xb9, byte(i % 256), byte(i / 256)}

	} else {
		return false, [3]byte{0x00, 0x00, 0x00}
	}
}

func IntegerTokenize(lines []string) []byte {

	start := 0x801
	currAddr := start

	buffer := make([]byte, 0)

	var linebuffer []byte

	add := func(chunk string) {
		if chunk != "" {
			if ok, ival := isInt(chunk); ok {
				linebuffer = append(linebuffer, ival[:]...)
				//fmt.Printf("TOK Integer(%d)\n", int(ival[1])+256*int(ival[2]))
			} else {
				// Encode strings with high bit (0x80) set
				//fmt.Printf("TOK String(%s)\n", strings.ToUpper(chunk))
				data := []byte(strings.ToUpper(chunk))
				for i, v := range data {
					data[i] = v | 0x80
				}
				linebuffer = append(linebuffer, data...)
			}
		}
	}

	for _, l := range lines {

		l = strings.Trim(l, "\r")
		if l == "" {
			continue
		}

		chunk := ""
		inqq := false
		tmp := strings.SplitN(l, " ", 2)
		ln, _ := strconv.Atoi(tmp[0])
		rest := strings.Trim(tmp[1], " ")

		linebuffer = make([]byte, 3)

		// LINE NUMBER
		linebuffer[0x01] = byte(ln & 0xff)
		linebuffer[0x02] = byte(ln / 0x100)

		// PROCESS LINE
		for _, ch := range rest {

			switch {
			case inqq && ch != '"':
				linebuffer = append(linebuffer, byte(ch|0x80))
				continue
			case ch == ':' && !inqq:
				linebuffer = append(linebuffer, 0x03)
				continue
			case ch == ',' && !inqq:
				linebuffer = append(linebuffer, 0x0A)
				continue
			case ch == ';' && !inqq:
				linebuffer = append(linebuffer, 0x45)
				continue
			case ch == '(' && !inqq:
				linebuffer = append(linebuffer, 0x22)
				continue
			case ch == ')' && !inqq:
				linebuffer = append(linebuffer, 0x72)
				continue
			case ch == '+' && !inqq:
				linebuffer = append(linebuffer, 0x12)
				continue
			case ch == '"':
				inqq = !inqq
				if inqq {
					ch = 0x28
				} else {
					ch = 0x29
				}
				linebuffer = append(linebuffer, byte(ch))
				continue
			case !inqq && breakingChar(ch):
				add(chunk)
				chunk = ""

				//linebuffer = append(linebuffer, byte(ch|0x80))
				continue
			}

			chunk += string(ch)
			code, ok := IntegerReverse[strings.ToUpper(chunk)]
			if ok {
				//fmt.Printf("TOK Token(%s)\n", chunk)
				linebuffer = append(linebuffer, byte(code))
				chunk = ""
			}
		}
		if chunk != "" {
			add(chunk)
		}

		linebuffer = append(linebuffer, 0x01) // EOL token

		nextAddr := currAddr + len(linebuffer)
		linebuffer[0x00] = byte(len(linebuffer))
		currAddr = nextAddr

		buffer = append(buffer, linebuffer...)
	}

	// Encode file length
	// buffer[0] = byte((len(buffer) - 2) % 256)
	// buffer[1] = byte((len(buffer) - 2) / 256)

	return buffer

}

func tst() {

	// lines := []string{
	// 	"10 PRINT \"HELLO WORLD!\"",
	// 	"20 GOTO 10",
	// }

	// b := IntegerTokenize(lines)

	// Dump(b)

	// os.Exit(1)

}

package rdf

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
)

// canonicalizeJSONText canonicalizes JSON using the JSON Canonicalization Scheme (JCS).
// This is adapted from WebPKI.org's JCS implementation, allowing any top-level JSON value.
func canonicalizeJSONText(jsonData []byte) ([]byte, error) {
	// JSON data MUST be UTF-8 encoded
	var jsonDataLength = len(jsonData)

	// Current pointer in jsonData
	var index = 0

	var parseElement func() string
	var parseSimpleType func() string
	var parseQuotedString func() string
	var parseObject func() string
	var parseArray func() string

	var globalError error

	checkError := func(err error) {
		if globalError == nil {
			globalError = err
		}
	}

	setError := func(msg string) {
		checkError(errors.New(msg))
	}

	isWhiteSpace := func(c byte) bool {
		return c == 0x20 || c == 0x0a || c == 0x0d || c == 0x09
	}

	nextChar := func() byte {
		if index < jsonDataLength {
			c := jsonData[index]
			index++
			return c
		}
		setError("Unexpected EOF reached")
		return '"'
	}

	scan := func() byte {
		for {
			c := nextChar()
			if isWhiteSpace(c) {
				continue
			}
			return c
		}
	}

	scanFor := func(expected byte) {
		c := scan()
		if c != expected {
			setError("Expected '" + string(expected) + "' but got '" + string(c) + "'")
		}
	}

	getUEscape := func() rune {
		start := index
		nextChar()
		nextChar()
		nextChar()
		nextChar()
		if globalError != nil {
			return 0
		}
		u16, err := strconv.ParseUint(string(jsonData[start:index]), 16, 64)
		checkError(err)
		return rune(u16)
	}

	testNextNonWhiteSpaceChar := func() byte {
		save := index
		c := scan()
		index = save
		return c
	}

	decorateString := func(rawUTF8 string) string {
		var quotedString strings.Builder
		quotedString.WriteByte('"')
	CoreLoop:
		for _, c := range []byte(rawUTF8) {
			for i, esc := range binaryEscapes {
				if esc == c {
					quotedString.WriteByte('\\')
					quotedString.WriteByte(asciiEscapes[i])
					continue CoreLoop
				}
			}
			if c < 0x20 {
				quotedString.WriteString(fmt.Sprintf("\\u%04x", c))
			} else {
				quotedString.WriteByte(c)
			}
		}
		quotedString.WriteByte('"')
		return quotedString.String()
	}

	parseQuotedString = func() string {
		var rawString strings.Builder
	CoreLoop:
		for globalError == nil {
			var c byte
			if index < jsonDataLength {
				c = jsonData[index]
				index++
			} else {
				nextChar()
				break
			}
			if c == '"' {
				break
			}
			if c == '\\' {
				c = nextChar()
				if c == 'u' {
					firstUTF16 := getUEscape()
					if utf16.IsSurrogate(firstUTF16) {
						if nextChar() != '\\' || nextChar() != 'u' {
							setError("Missing surrogate")
						} else {
							rawString.WriteRune(utf16.DecodeRune(firstUTF16, getUEscape()))
						}
					} else {
						rawString.WriteRune(firstUTF16)
					}
				} else if c == '/' {
					rawString.WriteByte('/')
				} else {
					for i, esc := range asciiEscapes {
						if esc == c {
							rawString.WriteByte(binaryEscapes[i])
							continue CoreLoop
						}
					}
					setError("Unexpected escape: \\" + string(c))
				}
			} else {
				// Allow raw control characters so we can re-escape them canonically.
				rawString.WriteByte(c)
			}
		}
		return rawString.String()
	}

	parseSimpleType = func() string {
		start := index - 1
		for index < jsonDataLength && !isWhiteSpace(jsonData[index]) && jsonData[index] != ',' && jsonData[index] != ']' && jsonData[index] != '}' {
			index++
		}
		value := string(jsonData[start:index])
		for _, literal := range literals {
			if literal == value {
				return literal
			}
		}
		ieeeF64, err := strconv.ParseFloat(value, 64)
		checkError(err)
		value, err = numberToJSON(ieeeF64)
		checkError(err)
		return value
	}

	parseElement = func() string {
		switch scan() {
		case '{':
			return parseObject()
		case '"':
			return decorateString(parseQuotedString())
		case '[':
			return parseArray()
		default:
			return parseSimpleType()
		}
	}

	parseArray = func() string {
		var arrayData strings.Builder
		arrayData.WriteByte('[')
		var next = false
		for globalError == nil && testNextNonWhiteSpaceChar() != ']' {
			if next {
				scanFor(',')
				arrayData.WriteByte(',')
			} else {
				next = true
			}
			arrayData.WriteString(parseElement())
		}
		scan()
		arrayData.WriteByte(']')
		return arrayData.String()
	}

	lexicographicallyPrecedes := func(sortKey []uint16, e *nameValueType) bool {
		oldSortKey := e.sortKey
		minLength := len(oldSortKey)
		if minLength > len(sortKey) {
			minLength = len(sortKey)
		}
		for q := 0; q < minLength; q++ {
			diff := int(sortKey[q]) - int(oldSortKey[q])
			if diff < 0 {
				return true
			} else if diff > 0 {
				return false
			}
		}
		if len(sortKey) < len(oldSortKey) {
			return true
		}
		if len(sortKey) == len(oldSortKey) {
			setError("Duplicate key: " + e.name)
		}
		return false
	}

	parseObject = func() string {
		nameValueList := make([]nameValueType, 0)
		var next = false
	CoreLoop:
		for globalError == nil && testNextNonWhiteSpaceChar() != '}' {
			if next {
				scanFor(',')
			}
			next = true
			scanFor('"')
			rawUTF8 := parseQuotedString()
			if globalError != nil {
				break
			}
			sortKey := utf16.Encode([]rune(rawUTF8))
			scanFor(':')
			nameValue := nameValueType{rawUTF8, sortKey, parseElement()}
			for i := 0; i < len(nameValueList); i++ {
				if lexicographicallyPrecedes(sortKey, &nameValueList[i]) {
					nameValueList = append(nameValueList[:i], append([]nameValueType{nameValue}, nameValueList[i:]...)...)
					continue CoreLoop
				}
			}
			nameValueList = append(nameValueList, nameValue)
		}
		scan()
		var objectData strings.Builder
		objectData.WriteByte('{')
		next = false
		for i := 0; i < len(nameValueList); i++ {
			if next {
				objectData.WriteByte(',')
			}
			next = true
			nameValue := nameValueList[i]
			objectData.WriteString(decorateString(nameValue.name))
			objectData.WriteByte(':')
			objectData.WriteString(nameValue.value)
		}
		objectData.WriteByte('}')
		return objectData.String()
	}

	transformed := parseElement()
	for index < jsonDataLength {
		if !isWhiteSpace(jsonData[index]) {
			setError("Improperly terminated JSON object")
			break
		}
		index++
	}
	if globalError != nil {
		return nil, globalError
	}
	return []byte(transformed), nil
}

type nameValueType struct {
	name    string
	sortKey []uint16
	value   string
}

// JSON standard escapes (modulo \u)
var asciiEscapes = []byte{'\\', '"', 'b', 'f', 'n', 'r', 't'}
var binaryEscapes = []byte{'\\', '"', '\b', '\f', '\n', '\r', '\t'}

// JSON literals
var literals = []string{"true", "false", "null"}

const invalidPattern uint64 = 0x7ff0000000000000

func numberToJSON(ieeeF64 float64) (res string, err error) {
	ieeeU64 := math.Float64bits(ieeeF64)
	if (ieeeU64 & invalidPattern) == invalidPattern {
		return "null", errors.New("Invalid JSON number: " + strconv.FormatUint(ieeeU64, 16))
	}
	if ieeeF64 == 0 {
		return "0", nil
	}
	sign := ""
	if ieeeF64 < 0 {
		ieeeF64 = -ieeeF64
		sign = "-"
	}
	format := byte('e')
	if ieeeF64 < 1e+21 && ieeeF64 >= 1e-6 {
		format = 'f'
	}
	es6Formatted := strconv.FormatFloat(ieeeF64, format, -1, 64)
	exponent := strings.IndexByte(es6Formatted, 'e')
	if exponent > 0 {
		gform := strconv.FormatFloat(ieeeF64, 'g', 17, 64)
		if len(gform) == len(es6Formatted) {
			es6Formatted = gform
		}
		if es6Formatted[exponent+2] == '0' {
			es6Formatted = es6Formatted[:exponent+2] + es6Formatted[exponent+3:]
		}
	} else if strings.IndexByte(es6Formatted, '.') < 0 && len(es6Formatted) >= 12 {
		i := len(es6Formatted)
		for es6Formatted[i-1] == '0' {
			i--
		}
		if i != len(es6Formatted) {
			fix := strconv.FormatFloat(ieeeF64, 'f', 0, 64)
			if fix[i] >= '5' {
				es6Formatted = fix[:i-1] + string(fix[i-1]+1) + es6Formatted[i:]
			}
		}
	}
	return sign + es6Formatted, nil
}

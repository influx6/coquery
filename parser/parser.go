package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// // Logger defines message logger that allows us to record parser actions.
// type Logger interface {
// 	Log(context interface{}, name string, message string, data ...interface{})
// 	Error(context interface{}, name string, err error, message string, data ...interface{})
// }

// section defines a regexp to part specific section of a query request string.
var section = regexp.MustCompile("([a-zA-Z0-9_\\-]+)\\((.+)\\)")

// ParseQuery returns the giving information as regarding the necessary data to
// be processed.
func ParseQuery(context interface{}, data string) []string {

	var parts []string

	// We need to adjust the final data to be able to capture its entity.
	data = fmt.Sprintf("%s.", data)

	buf := bytes.NewBufferString(data)
	read := bufio.NewReader(buf)

	for {

		line, err := read.ReadString('.')
		if err != nil {
			break
		}

		var hasOps bool

		if strings.Contains(line, "(") {
			hasOps = true
		}

		if !hasOps {
			line = strings.TrimSuffix(line, ".")
			parts = append(parts, line)
			continue
		}

		fline := strings.TrimSuffix(line, ".")
		if section.MatchString(line) && strings.HasSuffix(fline, ")") {
			line = strings.TrimSuffix(line, ".")
			parts = append(parts, line)
			continue
		}

		var mline string
		for {

			// Read next so we can capture possible failure.
			xline, err := read.ReadString('.')

			if err != nil {
				break
			}

			if strings.HasSuffix(mline, ")") {
				xline = strings.TrimSuffix(xline, ".")
				mline = mline + xline
				break
			}

			mline = mline + xline
		}

		line = line + mline

		line = strings.TrimSuffix(line, ".")
		parts = append(parts, line)
	}

	return parts
}

// runes provides a map of special symbols with their endings to allows us
// capture adequately proper parts of a query.
var runes = map[string]string{
	"{": "}", "}": "{",
	"(": ")", ")": "(",
	"[": "]", "]": "[",
	"\"": "\"", "'": "'",
	"`": "`",
}

// SplitQuery returns a method name and the content of that mkethod name for a
// query section .eg SplitQuery("find(id,1)") => returns (find, "id,1").
func SplitQuery(context interface{}, sec string) (method string, content string, contentPart []string) {
	if !section.MatchString(sec) {
		return
	}

	subs := section.FindStringSubmatch(sec)
	method = subs[1]
	content = subs[2]

	// We need to adjust the final data to be able to capture its entity.
	data := fmt.Sprintf("%s,", content)

	buf := bytes.NewBufferString(data)
	read := bufio.NewReader(buf)

	for {

		line, err := read.ReadString(',')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)

		// Get the prefix of this part.
		pl := string(line[0])

		// Do we have a possible special rune, if not add to list and continue
		if _, ok := runes[pl]; !ok {
			line = strings.TrimSuffix(line, ",")
			contentPart = append(contentPart, line)
			continue
		}

		el := runes[pl]

		// If we have such a special character, then check if the special character
		// also ends the item else then setup the depth level and
		// the state we are in.
		if strings.HasSuffix(line, el) {
			line = strings.TrimSuffix(line, ",")
			contentPart = append(contentPart, line)
			continue
		}

		depth := 1
		subs := []string{line}

	iloop:
		for {
			if depth <= 0 {
				break iloop
			}

			mline, err := read.ReadString(',')
			if err != nil {
				break iloop
			}

			// mline = strings.TrimSpace(mline)

			if strings.HasSuffix(mline, el) && depth > 0 {
				subs = append(subs, mline)
				depth--
				continue iloop
			}

			if strings.HasPrefix(mline, pl) {
				subs = append(subs, mline)
				depth++
				continue iloop
			}

			subs = append(subs, mline)
		}

		subline := strings.Join(subs, "")
		subline = strings.TrimSuffix(subline, ",")
		contentPart = append(contentPart, subline)
	}

	return
}

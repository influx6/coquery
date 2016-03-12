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

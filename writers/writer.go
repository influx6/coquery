package writers

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
)

//==============================================================================

// NopWriter provides a no-close io.Writer that allows us to decorate a flat
// io.Writer as a io.WriteCloser.
type NopWriter struct {
	w io.Writer
}

// NewNopWriter returns a new No-Op writer decorated over the supplied io.Writer.
func NewNopWriter(w io.Writer) *NopWriter {
	hx := NopWriter{w: w}
	return &hx
}

// Close returns nil since this a no-op operation.
func (n *NopWriter) Close() error {
	return nil
}

// Writer calls the internal io.Writer write method.
func (n *NopWriter) Write(b []byte) (int, error) {
	return n.w.Write(b)
}

//==============================================================================

// HexToBase64 returns a new io.WriteCloser which encodes that into hexadecimal
// format then base64 encodes the stream.
func HexToBase64(w io.Writer) io.WriteCloser {
	return HexToBase64WithEncoding(w, base64.StdEncoding)
}

// HexToBase64WithEncoding takes a writer then wraps a base64 writer as the
// output writer for the returned HexWriter, hence coverting all data into
// hexadecimal then base64 encoded using the provided encoding.
func HexToBase64WithEncoding(w io.Writer, enc *base64.Encoding) io.WriteCloser {
	bs := base64.NewEncoder(enc, w)
	return NewHexWriter(bs)
}

// NewGzippedHexWriter returns a new gzip writer combined with a hex writer.
func NewGzippedHexWriter(w io.Writer) io.WriteCloser {
	hx := NewHexWriter(w)
	return gzip.NewWriter(hx)
}

//==============================================================================

// HexWriter turns a series of bytes into stringed hexadecimal format.
type HexWriter struct {
	w io.WriteCloser
}

// NewHexWriter returns a new HexWriter using the supplied writer to
// write a hex version of the data.
func NewHexWriter(w io.Writer) *HexWriter {
	var wc io.WriteCloser

	if wx, ok := w.(io.WriteCloser); ok {
		wc = wx
	} else {
		wc = NewNopWriter(w)
	}

	hx := HexWriter{w: wc}
	return &hx
}

// lowerhex provides the set characters within the hexadecimal notation.
const lowerhex = "0123456789abcdef"

// Write meets the io.Write interface Write method
func (sw *HexWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	var sod = []byte(`\x00`)
	var b byte

	for n, b = range p {
		sod[2] = lowerhex[b/16]
		sod[3] = lowerhex[b%16]
		sw.w.Write(sod)
	}

	n++

	return
}

// Close calls the internal writer's close method.
func (sw *HexWriter) Close() error {
	return sw.w.Close()
}

//==============================================================================

// SanitizeBytes prepares a valid UTF-8 string as a raw string constant.
// Removing BOM characters when found with `+"\xEF\xBB\xBF"+`.
// Allows us to embed such bytes in go files if needed.
func SanitizeBytes(b []byte) []byte {

	// Replace ` with `+"`"+`
	b = bytes.Replace(b, []byte("`"), []byte("`+\"`\"+`"), -1)

	// Replace BOM with `+"\xEF\xBB\xBF"+`
	// (A BOM is valid UTF-8 but not permitted in Go source files.
	// I wouldn't bother handling this, but for some insane reason
	// jquery.js has a BOM somewhere in the middle.)
	return bytes.Replace(b, []byte("\xEF\xBB\xBF"), []byte("`+\"\\xEF\\xBB\\xBF\"+`"), -1)
}

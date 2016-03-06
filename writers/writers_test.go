package writers_test

import (
	"bytes"
	"testing"

	"github.com/ardanlabs/kit/tests"
	"github.com/influx6/coquery/writers"
)

//==============================================================================

var context = "testing"

//==============================================================================

func init() {
	tests.Init("")
}

//==============================================================================

var expectedHex = `\x77\x65\x6c\x69\x6e\x67\x68\x74\x6f\x6e\x2e`

// TestHexWriter validates our hex writer works in converting all data into the
// appropriate hexadecimal counterpart.
func TestHexWriter(t *testing.T) {
	tests.ResetLog()
	defer tests.DisplayLog()

	t.Logf("Given the need to be write data in hexdecimal form.")
	{

		t.Logf("\tWhen giving a hex writer")
		{

			var buf bytes.Buffer

			hex := writers.NewHexWriter(&buf)

			hex.Write([]byte("welinghton."))
			hex.Close()

			if buf.String() != expectedHex {
				t.Fatalf("\t%s\tShould have recieved expect hex data: %s", tests.Failed, buf.String())
			}
			t.Logf("\t%s\tShould have recieved expect hex data: %s", tests.Success, expectedHex)
		}
	}

}

//==============================================================================

var expected64 = `XHg3N1x4NjVceDZjXHg2OVx4NmVceDY3XHg2OFx4NzRceDZmXHg2ZVx4MmU=`

// TestHexBase64Writer validates encoding of hex as base64 data using the
// writes.HexToBase64 writer.
func TestHexBase64Writer(t *testing.T) {
	tests.ResetLog()
	defer tests.DisplayLog()

	t.Logf("Given the need to be write data in hexdecimal form.")
	{

		t.Logf("\tWhen giving a hex writer")
		{

			var buf bytes.Buffer

			hex := writers.HexToBase64(&buf)

			hex.Write([]byte("welinghton."))
			hex.Close()

			if buf.String() != expected64 {
				t.Fatalf("\t%s\tShould have recieved expect hex data: %s", tests.Failed, buf.String())
			}
			t.Logf("\t%s\tShould have recieved expect hex data: %s", tests.Success, expected64)
		}
	}

}

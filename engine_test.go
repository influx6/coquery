package coquery_test

import (
	"fmt"
	"testing"

	"github.com/influx6/coquery"
)

//==============================================================================

var context = "testing"

//==============================================================================

// logg provides a concrete implementation of a logger.
type logg struct{}

// Log logs all standard log reports.
func (l *logg) Log(context interface{}, name string, message string, data ...interface{}) {
	if testing.Verbose() {
		fmt.Printf("Log : %s : %s : %s", context, name, fmt.Sprintf(message, data...))
	}
}

// Error logs all error reports.
func (l *logg) Error(context interface{}, name string, err error, message string, data ...interface{}) {
	if testing.Verbose() {
		fmt.Printf("Error : %s : %s : %s", context, name, fmt.Sprintf(message, data...))
	}
}

//==============================================================================

// TestCoEngine validates the operation provided by the coquery.Engine.
func TestCoEngine(t *testing.T) {
	t.Logf("Given the need to use a coquery engine for request processor")
	{

		t.Logf("\tWhen giving a mongo document router")
		{

			eos := coquery.New(new(logg))
		}
	}
}

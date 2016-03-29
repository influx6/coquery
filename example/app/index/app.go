package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ardanlabs/kit/log"
	"github.com/influx6/coquery"
	"github.com/influx6/coquery/client"
	"github.com/influx6/coquery/client/js"
	"honnef.co/go/js/dom"
)

func init() {
	log.Init(os.Stdout, func() int { return log.DEV }, log.Ldefault)
}

//==============================================================================

var events eventlog

// logg provides a concrete implementation of a logger.
type eventlog struct{}

// Log logs all standard log reports.
func (l eventlog) Log(context interface{}, name string, message string, data ...interface{}) {
	log.Dev(context, name, message, data...)
}

// Error logs all error reports.
func (l eventlog) Error(context interface{}, name string, err error, message string, data ...interface{}) {
	log.Error(context, name, err, message, data...)
}

//==============================================================================

var context = "example-app"

//==============================================================================

func main() {

	window := dom.GetWindow()
	doc := window.Document()

	clientServo := client.NewServo("http://127.0.0.1:3000", 300*time.Millisecond, js.HTTP)

	all := clientServo.Register("docs.users.findN(-1)")

	all.Listen(func(err error, records coquery.Parameters) {

		if err != nil {
			events.Error(context, "Listen", err, "All query Failed")
			return
		}

		for _, record := range records {
			div := doc.CreateElement("div")
			div.SetInnerHTML(fmt.Sprintf("%+v", record))
			doc.QuerySelector("body").AppendChild(div)
		}
	})

	if err := all.Do(); err != nil {
		events.Error(context, "all.Do", err, "All query Failed")
	}

}

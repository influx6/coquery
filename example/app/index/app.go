package main

import (
	"fmt"
	"time"

	"github.com/influx6/coquery/client"
	"github.com/influx6/coquery/client/js"
	"github.com/influx6/coquery/data"
	"honnef.co/go/js/dom"
)

func init() {
	// log.Init(os.Stdout, func() int { return log.DEV }, log.Ldefault)
}

//==============================================================================

var events eventlog

// logg provides a concrete implementation of a logger.
type eventlog struct{}

// Log logs all standard log reports.
func (l eventlog) Log(context interface{}, name string, message string, data ...interface{}) {
	fmt.Printf("Log: %s : %s : %s : %s\n", context, "DEV", name, fmt.Sprintf(message, data...))
}

// Error logs all error reports.
func (l eventlog) Error(context interface{}, name string, err error, message string, data ...interface{}) {
	fmt.Printf("Error: %s : %s : %s : %s\n", context, "DEV", name, fmt.Sprintf(message, data...))
}

//==============================================================================

var context = "example-app"

//==============================================================================

func main() {

	window := dom.GetWindow()
	doc := window.Document()

	clientServo := client.NewServo("http://127.0.0.1:3000", 300*time.Millisecond, js.HTTP)

	all := clientServo.Register("docs.users.findN(-1)")

	all.Listen(func(err error, records data.Parameters) {

		fmt.Printf("Received: %s -> %+s\n", err, records)
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

	fmt.Printf("Sending: \n")
	if err := all.Do(); err != nil {
		events.Error(context, "all.Do", err, "All query Failed")
	}

}

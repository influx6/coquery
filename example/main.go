package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ardanlabs/kit/log"
	"github.com/influx6/coquery"
	"github.com/influx6/coquery/client"
	"github.com/influx6/coquery/client/web"
	"github.com/influx6/coquery/documents/mongodocs"
	"github.com/influx6/coquery/protocols/cohttp"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/coquery/utils"
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

	var wg sync.WaitGroup
	wg.Add(1)

	diff := coquery.NewExpiringDiffs(events, 1*time.Hour)
	store := storage.NewExpirable("uid", 1*time.Hour)
	app := cohttp.New(events, diff, store)

	app.Route(context, "docs").
		DocumentWith(context, "users", mongodocs.New(mongodocs.DocumentConfig{
		Events:   events,
		Store:    store,
		Workers:  20,
		Wait:     50 * time.Second,
		Host:     "127.0.0.1:27017",
		AuthDB:   "contacts",
		DB:       "contacts",
		QueryDoc: "users",
	}))

	go app.ListenAndServe(context, ":3000")

	clientServo := client.NewServo("http://127.0.0.1:3000", 300*time.Millisecond, web.HTTP)

	all := clientServo.Register("docs.users.findN(-1)")

	all.Listen(func(err error, data coquery.Parameters) {
		defer wg.Done()

		if err != nil {
			events.Error(context, "Listen", err, "All query Failed")
			return
		}

		fmt.Printf("Received All Response: %s\n", utils.Query.QueryIndent(data))
	})

	if err := all.Do(); err != nil {
		events.Error(context, "all.Do", err, "All query Failed")
	}

	wg.Wait()
}

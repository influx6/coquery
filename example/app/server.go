package main

import (
	"os"
	"time"

	"github.com/ardanlabs/kit/log"
	"github.com/influx6/coquery"
	"github.com/influx6/coquery/documents/mongodocs"
	"github.com/influx6/coquery/protocols/cohttp"
	"github.com/influx6/coquery/storage"
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

	diff := coquery.NewExpiringDiffs(events, 1*time.Hour)
	store := storage.NewExpirable("uid", 1*time.Hour)
	app := cohttp.New(events, diff, store)
	app.EnableCORS()

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

	app.ListenAndServe(context, ":3000")

}

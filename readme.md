# Coquery
Coquery is an experiement in providing a different way of access and
querying underline database systems with an approach that gives the client
more control of what they want from the backend than the formal prescribed
approach of defined endpoints(Restful CRUD) that only define routes for
such operations. We are entering into grounds where exact requirements of
parts of the UI are differing and allowing clients to only receive what matters
to them as become as critical as any part of our application behaviour.

## Install

```bash
go get -u github.com/influx/coquery/...
```

## API Design
 Coquery was designed to support a flexible approach in how data is retrieved and
 stored on the backend using a interesting query dsl.

  Example Query Patterns:

```bash

// Retrieve all records and collect only the "name","age" and "address" properties.
docs.users.findN(-1).collects(name,age,address)

// Retrieve 10 records and collect only the "name","age" and "address" properties.
docs.users.findN(10).collects(name,age,address)

// Retrieve 10 records for the first 10 set and collect only the "name","age" and "address" properties.
docs.users.findN(10,10).collects(name,age,address)

// Retrieve 10 records for the first 20 after the first and collect only the "name","age" and "address" properties.
docs.users.findN(10,20).collects(name,age,address)

// Retrieve record with the id=10 and collect only the "name","age" and "address" properties.
docs.users.find(id,10).collects(name,age,address)


// Retrieve record with the id=10 and mutate the "name" property to alex.
docs.user.find(id,0).mutate({name:"alex"})

/* Experiemental ideas not yet implemented

//Retrieve record with the id=10 and mutate the with the details of the
//encoded mapped in hex format
docs.user.find(id,0).mutate(hex("\x32\x4e\x54\x11\x21\x3a"))

//Retrieve record with the id=10 and mutate the with the details of the
//encoded mapped in base64 format
docs.user.find(id,0).mutate(b64("XHg3N1x4NjVceDZjXHg2OVx4NmVceDY3XHg2OFx4NzRceDZmXHg2ZVx4MmU="))

*/

```

## Example

```go
package main

import (
	"net/http"
	"os"
	"time"

	"github.com/ardanlabs/kit/log"
	"github.com/influx6/coquery/documents/mongo"
	"github.com/influx6/coquery/protocols/cohttp"
	"github.com/influx6/coquery/storage"
)

func init() {
	log.Init(os.Stdout, func() int { return log.DEV }, log.Ldefault)
}

//=============================================================================

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

//=============================================================================

var context = "example-app"

//=============================================================================

func main() {

	store := storage.NewExpirable("uid", 1*time.Hour)
	app := cohttp.New(events)

	app.Route(context, "docs").
		DocumentWith(context, "users", mongo.New(mongo.DocumentConfig{
		Events:   events,
		Store:    store,
		Workers:  20,
		Wait:     5 * time.Minute,
		Host:     "127.0.0.1:27017",
		AuthDB:   "contacts",
		DB:       "contacts",
		QueryDoc: "users",
	}))

	// http.ListenAndServe(":3000", app)
	app.ListenAndServe(context, ":3000")
}

```

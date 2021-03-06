# Coquery
Coquery is an experiement in providing a different way of access and
querying underline database systems with an approach that gives the client
more control of what they want from the backend than the formal prescribed
approach of defined endpoints(Restful CRUD) that only define routes for
such operations. We are entering into grounds where exact requirements of
parts of the UI are differing and allowing clients to only receive what matters
to them as become as critical as any part of our application behaviour. It
takes inspiration from [Falcor](https://netflix.github.com/falcor) and other
libraries that provides similiar behaviours.

## Install

```bash
go get -u github.com/influx/coquery/...
```

## API Design
 Coquery was designed to support a flexible approach in how data is retrieved and stored on the backend using a interesting query DSL.

### Query Paths:
  While thinking hard about what the form and look for the query system, I
  I understood, I wanted a system that was familiar and yet robust but not
  become to complicated to become something that needed deep explanations.
  I believe that a complex system that can be expressed as simple as possible
  makes for a great system and Coquery follows that principle at heart.

  Every section of a Query path is an operation that must be performed on the
  previous based on the result of the previous operation which allows the
  system to provide flexibility in the representation of the end result.


```bash

// Retrieve all records and collect only the "name","age" and "address" properties.
docs.users.findN(-1).collects(name,age,address)

// Retrieve 10 records and collect only the "name","age" and "address" properties.
docs.users.findN(10).collects(name,age,address)

// Retrieve 10 records for after the first 10 set and collect only the "name","age" and "address" properties.
docs.users.findN(10,10).collects(name,age,address)

// Retrieve 10 records for after the first 20 after the first and collect only the "name","age" and "address" properties.
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

### API Reply
  Coquery provides a specific reply pattern to all requests which are returned
  from regardless of the query and result, this is set to provide the flexibility
  of including meta details to the responses received from the API.

  *When sending request, the API expects to find the request to be
  a `application/json` content type format.*

  When using the `application/json` content-type for request, the coquery API
  will attempt to load the data into the RequestContext struct, which allows
  controls on how the request will be treated.

  Note: Infact only one attribute `Queries` is required and the rest can be
  left unfield as they only enable specific reply fields.

```go
  type RequestContext struct {
  	RequestID string   `json:"request_id"`
  	Queries   []string `json:"queries"`
  	Diffs     bool     `json:"diffs"`
  	DiffTag   string   `json:"diff_tag"`
  	DiffWatch []string `json:"diff_watch"`
  }
```

#### Request
  When batch query requests to the API are made, it responds with the following json.

  Request Example:

```json
  {
    "queries":["docs.users.find(id,3)", "docs.books.find(uid,30)"]
  }
```

  Request Response:
```JSON
  {
     "record_key": "_id",
     "request_id": "36564-423266-656dA232",
     "delta_id": "36564-423266-656dA232",
     "last_delta_id": "36564-423266-656dA232",
     "batch": true,
     "results": [{"data":[{}] }],
     "total": 20,
     "deltas": [""],
  }
```

The Coquery JSON response will contain standard attributes which provide
information to the client side on the result of the operation. These tags are
as follows:

  - "record_key"
   The `record_key` defines the actual key being used to reference records within
   the server store. This key is the means through which records are organized
   and allows clients to requests updates for these specific records.

  - "last_delta_id"
   The `last_delta_id` is a optional attribute that contains the UUID of the last delta report sent to the client, usually this signifies to the API which delta for record changes it sent last and which the client has last.
   This is included in the client response headers and client cookies.

  - "delta_id"
   The `delta_id` is a optional attribute that contains the UUID of the current delta report sent to the client, usually this signifies to the client which
   delta for record changes is sent to it, this is also included in the response headers and as client cookies.

   - "results"
   The `results` attribute contains the actual result of the query which was
   sent to the API.

   - "total"
   The `total` attribute contains the total result returned from the query which was returned from the backend.

   - "deltas"
   The `deltas` is a optional attribute that contains record IDs which
   were established as changed on the backend and allows the client to make
   requests for this records accordingly to their respective needs.

## Example
  Run example code in the coquery/example folder and send the request URL
  using your browser or curl.

  ```bash
    > cd coquery/example
    > go run main.go
  ```

```go
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
	"github.com/influx6/coquery/data"
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

	all.Listen(func(err error, data data.Parameters) {
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

```

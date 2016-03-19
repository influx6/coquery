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

### API Reply
  Coquery provides a specific reply pattern to all requests which are returned
  from regardless of the query and result, this is set to provide the flexibility of including meta details to the responses received from the
  API.

#### Single Request
  When single query requests to the API are made, it responds with the following json.

  Request Example: docs.users.find(id,3)

```JSON
  [{
     "request_id": "36564-423266-656dA232",
     "last_delta_id": "36564-423266-656dA232",
     "delta_id": "36564-423266-656dA232",
     "batch": false,
     "results": [{}],
     "total": 20,
     "deltas": [""],
  }]
```

#### Batch Request
  When batch query requests to the API are made, it responds with the following json.

  Request Example: [docs.users.find(id,3), docs.books.find(uid,30)]

```JSON
  [{
     "request_id": "36564-423266-656dA232",
     "delta_id": "36564-423266-656dA232",
     "last_delta_id": "36564-423266-656dA232",
     "batch": true,
     "results": [{"data":[{}] }],
     "total": 20,
     "deltas": [""],
  }]
```

The Coquery JSON response will contain standard attributes which provide
information to the client side on the result of the operation. These tags are
as follows:

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

  Depending on the presence of specific headers within the requests, coquery
  returns addition meta information for a requests, this headers include

  - X-CoQuery-WantDelta:
   e.g "X-CoQuery-WantDelta" : "1"

   This sets the coquery API to report record delta changes back to the requests which allows the API to be able to fetch delta changes for records
   watched by the API.

  - X-CoQuery-DUID:

   e.g "X-CoQuery-DUID" : "454HF-F34GH8-4395343G"

   This represents the last generated requests UID that identifies what
   previous delta/record change update where included or alluded to from the
   last request. This is also set as a cookie also. This makes the API add the
   "deltaKeys" field which will contain all records that the API knows has changed, so the requester could make requests for all this records.

  - X-CoQuery-DeltaWatch:

   e.g "X-CoQuery-DeltaWatch" : "[56434, 32231, 32322]"

   This represents a list of records for which the request desires report on
   changes for, this allows all request to cherrypick the delta updates to be
   reported back to them as the "deltas" field within the JSON response.


## Example

```go
package main

import (
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

	diff := coquey.NewExpiringDiffs(events, 1*time.Hour)
	store := storage.NewExpirable("uid", 1*time.Hour)
	app := cohttp.New(events, diff, store)

	app.Route(context, "docs").
		DocumentWith(context, "users", mongo.New(mongo.DocumentConfig{
		Events:   events,
		Store:    store,
		Workers:  20,
		Wait:     20 * time.Second,
		Host:     "127.0.0.1:27017",
		AuthDB:   "contacts",
		DB:       "contacts",
		QueryDoc: "users",
	}))

	app.ListenAndServe(context, ":3000")
}

```

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

  Example Query:

```go

docs.users.find(id,10).collects(name,age,address)

find => {type: find, key:id, value: 0}
collects => {type: collect, keys: [name, age ,address]}

docs.user.find(id,0).mutate({name:"alex"})

find => {type: find, key:id, value: 0}
mutate => {type: mutate, keys: {name: "alex"}}

```

  Example API:

```go
package main

import (
  "github.com/ardanlabs/kit/log"
  "github.com/influx6/coquery"
  dbMongo "github.com/influx6/db/mongo"
  smMongo "github.com/influx6/streams/mongo"
)

// logg provides a concrete implementation of a logger.
type logg struct{}

// Log logs all standard log reports.
func (l *logg) Log(context interface{}, name string, message string, data ...interface{}) {
	fmt.Printf("Log : %s : %s : %s", context, name, fmt.Sprintf(message, data...))
}

// Error logs all error reports.
func (l *logg) Error(context interface{}, name string, err error, message string, data ...interface{}) {
	fmt.Printf("Error : %s : %s : %s", context, name, fmt.Sprintf(message, data...))
}

func main(){

  var context = "example"
  var logger = new(logg)

  var engine = coquery.New(logger)

  mdb, err := dmongo.New(logger,dbMongo.Config{
		Host:     "db.mongohouse.com:5430",
		AuthDB:   "mob",
		DB:       "mob",
		User:     "box",
		Password: "box",
  })

  if err != nil {
    panic(err)
  }

  engine.Route(context,"docs")
  .Document(context,"users",smMongo.New(logger,mdb))
  .Document(context,"admins",smMongo.New(logger,mdb))
  .Document(context,"reports",smMongo.New(logger,mdb))

  http.ListenAndServe(":3000",engine)

}

```

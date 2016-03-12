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

func main(){

  var engine = coquery.NewEngine()
  var logger = log.New(os.Stdout,func() int{ return log.Dev},log.Ldefault)

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

  engine.Route("docs")
  .Document("users",smMongo.New(logger,mdb))
  .Document("admins",smMongo.New(logger,mdb))
  .Document("reports",smMongo.New(logger,mdb))

  http.ListenAndServe(":3000",engine)

}

```

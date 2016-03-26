package mongo

import (
	"time"

	"gopkg.in/mgo.v2"

	"github.com/influx6/coquery"
	"github.com/influx6/coquery/documents/mongo/db"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/coquery/streams"
	"github.com/influx6/faux/sumex"
)

//==============================================================================

// Events defines event logger that allows us to record events for a specific
// action that occured.
type Events interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==============================================================================

// MError provides a custom error message for requests types.
type MError struct {
	Rid    string `json:"rid" bson:"rid"`
	Msg    string `json:"message" bson:"message"`
	IError error  `json:"error" bson:"error"`
}

// Message returns the internal message for this error
func (r MError) Message() string {
	return r.Msg
}

// RequestID returns the response error requestID
func (r MError) RequestID() string {
	return r.Rid
}

// Error returns the error message for this response error.
func (r MError) Error() string {
	if r.IError != nil {
		return r.Rid + " : " + r.Msg + " : " + r.IError.Error()
	}

	return r.Rid + " : " + r.Msg
}

//==============================================================================

// DB provides a interface that provides a execution method for a mongodb DB.
type DB interface {
	New(context interface{}) (*mgo.Database, *mgo.Session, error)
}

//==============================================================================

// DocumentConfig provides a central configuration to initialize the documents
// internal systems.
type DocumentConfig struct {
	Events
	Store storage.Store

	// Stream configuration
	Workers int

	// Wait time for each request.
	Wait time.Duration

	// DB configuration
	Host     string
	AuthDB   string
	DB       string
	User     string
	Password string

	// QueryDoc to set an alternative db.document name for the queries to use.
	QueryDoc string
}

// Document provides a Mongo coquery.DocumentOS which provides the internal
// Mongo OS processor and the query processor. It is mainly created to
// simplify the initialization of the system.
type Document struct {
	*DocumentConfig
	sumex.Streams

	handler coquery.Documents
	query   coquery.QueryProcessor
}

// New returns a new instance of a Document which embodies the initializations
// needed to create a coquery.DocumentOS implementing structure.
func New(config DocumentConfig) *Document {

	streamos := streams.New(streams.Config{
		Log:     config.Events,
		Wait:    config.Wait,
		Workers: config.Workers,
	})

	queries := &coquery.BasicQueries{
		EventLog: config.Events,
		Doc:      config.QueryDoc,
	}

	dc := Document{
		DocumentConfig: &config,
		Streams:        streamos,
		handler:        streamos,
		query:          queries,
	}

	// Initalize the db provider for connecting to the database.
	db := db.Mongnod{
		Events: config.Events,
		Config: db.Config{
			Host:     config.Host,
			AuthDB:   config.AuthDB,
			DB:       config.DB,
			User:     config.User,
			Password: config.Password,
		},
	}

	// Set up the processors for this provider
	dc.Stream(sumex.New(config.Workers, config.Events, &house.Find{
		EventLog: config.Events,
		Mongo:    db,
		Store:    config.Store,
	}))

	dc.Stream(sumex.New(config.Workers, config.Events, &house.Mutate{
		EventLog: config.Events,
		Mongo:    db,
		Store:    config.Store,
	}))

	dc.Stream(sumex.New(config.Workers, config.Events, &house.All{
		EventLog: config.Events,
		Mongo:    db,
		Store:    config.Store,
	}))

	return &dc
}

// Document returns the processor interface for using this document.
func (d *Document) Document() coquery.Documents {
	return d.handler
}

// Queries returns the processor interface for using this document.
func (d *Document) Queries() coquery.QueryProcessor {
	return d.query
}

//==============================================================================

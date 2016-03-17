package mongo

import (
	"time"

	"github.com/influx6/coquery"
	"github.com/influx6/coquery/documents/mongo/db"
	"github.com/influx6/coquery/documents/mongo/house"
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

// DocumentConfig provides a central configuration to initialize the documents
// internal systems.
type DocumentConfig struct {
	Events
	Store storage.Store

	// Stream configuration
	Workers int
	Wait    time.Duration

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

	mg house.Mongo
	qm house.Query
}

// New returns a new instance of a Document which embodies the initializations
// needed to create a coquery.DocumentOS implementing structure.
func New(config DocumentConfig) *Document {

	// Initalize the db provider for connecting to the database.
	db, err := db.New(config.Events, db.Config{
		Host:     config.Host,
		AuthDB:   config.AuthDB,
		DB:       config.DB,
		User:     config.User,
		Password: config.Password,
	})

	// If we fail to connect to the db then panic.
	if err != nil {
		panic(err)
	}

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
		mg:             db,
		qm:             db,
	}

	// Set up the processors for this provider
	dc.Stream(sumex.New(config.Workers, &house.FindProc{
		EventLog: config.Events,
		Mongo:    dc.mg,
		Query:    dc.qm,
		Store:    config.Store,
	}))

	return &dc
}

// Document returns the processor interface for using this document.
func (d *Document) Document() coquery.Documents {
	return d.handler
}

// Query returns the processor interface for using this document.
func (d *Document) Query() coquery.QueryProcessor {
	return d.query
}

//==============================================================================

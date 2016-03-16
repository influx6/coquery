package mongo

import (
	"github.com/influx6/coquery"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/faux/sumex"
	"gopkg.in/mgo.v2"
)

//==========================================================================================

// EventLog defines event logger that allows us to record events for a specific
// action that occured.
type EventLog interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==========================================================================================

// DB provides a interface that provides a execution method for a mongodb DB.
type DB interface {
	ExecuteDB(context interface{}, collectionName string, fn func(*mgo.Collection) error) error
	Shutdown(context interface{})
}

//==========================================================================================

// MgoStream provides a streamer for the mongo service layer in coquery.
type MgoStream struct {
	EventLog
	store storage.Store
	db    DB
	in    sumex.Streams
	out   sumex.Streams
}

// New returns a new MongoStream instance.
func New(e EventLog, db DB, store storage.Store) (*MgoStream, error) {
	ms := MgoStream{
		EventLog: e,
		store:    store,
		db:       db,
	}

	return &ms, nil
}

// Bind connects a sumex.Processor into the mgo streamer.
func (m *MgoStream) Bind(context interface{}, sm sumex.Streams) {

}

// Handle provides processing for the incoming coquery.Requests for the mongo
// db.
func (m *MgoStream) Handle(context interface{}, rq coquery.RecordRequests, rw coquery.ResponseWriter) {
	m.Log(context, "Handle", "Started : Requests %d", len(rq))

	defer m.db.Shutdown(context)

	m.Log(context, "Handle", "Completed")
}

package mongo

import (
	"sync"
	"time"

	"github.com/influx6/coquery/utils"

	"gopkg.in/mgo.v2"
)

//==============================================================================

var mstore struct {
	ml   sync.RWMutex
	list map[string]*mgo.Session
}

// masterListLock provides a mutex for controlling access to the masterList.
var masterListLock sync.RWMutex

// masterList contains a set of session lists of connections that have been
// created
var masterList = make(map[string]*mgo.Session)

//==============================================================================

// Config provides configuration for connecting to a db.
type Config struct {
	Host     string
	AuthDB   string
	DB       string
	User     string
	Password string
}

//==============================================================================

// EventLog defines event logger that allows us to record events for a specific
// action that occured.
type EventLog interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==============================================================================

// Mongnod defines a mongo connection manager that builds off a mongo instance.
type Mongnod struct {
	Config
	EventLog
}

// New connects and initializes the master session for the mongo list.
func (m *Mongnod) New(context interface{}) (*mgo.Database, *mgo.Session, error) {
	m.Log(context, "New", "Started : Config : %s", utils.Query.Query(m.Config))

	key := m.Host + ":" + m.DB

	masterListLock.Lock()
	ms, ok := masterList[key]
	masterListLock.Unlock()

	if ok {
		m.Log(context, "New", "Completed")
		ses := ms.Copy()
		return ses.DB(m.DB), ses, nil
	}

	// If not found, then attemp to connect and add to session master list.
	// We need this object to establish a session to our MongoDB.
	info := mgo.DialInfo{
		Addrs:    []string{m.Host},
		Timeout:  60 * time.Second,
		Database: m.AuthDB,
		Username: m.User,
		Password: m.Password,
	}

	// Create a session which maintains a pool of socket connections
	// to our MongoDB.
	ses, err := mgo.DialWithInfo(&info)
	if err != nil {
		m.Error(context, "New", err, "Completed")
		return nil, nil, err
	}

	ses.SetMode(mgo.Monotonic, true)

	// Add to master list.
	masterListLock.Lock()
	masterList[key] = ses.Copy()
	masterListLock.Unlock()

	m.Log(context, "New", "Completed")
	return ses.DB(m.DB), ses, nil
}

//==========================================================================================

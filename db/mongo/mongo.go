package mongo

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"gopkg.in/mgo.v2"
)

//==============================================================================

// masterListLock provides a mutex for controlling access to the masterList.
var masterListLock sync.Mutex

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
	*Config
	EventLog
	m  *mgo.Session
	rl sync.RWMutex
}

// New returns a new Mongnod instance.
func New(l EventLog, c Config) (*Mongnod, error) {
	m := Mongnod{
		Config:   &c,
		EventLog: l,
	}

	// connect to the mongodb server.
	if err := m.connectDB("New"); err != nil {
		return nil, err
	}

	return &m, nil
}

//==============================================================================

// QueryIndent returns the stringified version of the giving data and indents
// its result. Uses json.Marshal underneath.
func (m *Mongnod) QueryIndent(ms interface{}) string {
	data, err := json.MarshalIndent(ms, "", "\n")
	if err != nil {
		return ""
	}

	return string(data)
}

//==============================================================================

// Query returns a stringified version of the provided argument
// using json.Marshal.
func (m *Mongnod) Query(ms interface{}) string {
	data, err := json.Marshal(ms)
	if err != nil {
		return ""
	}

	return string(data)
}

// Shutdown closes the connection and its internal session provider.
func (m *Mongnod) Shutdown(context interface{}) {
	m.Log(context, "Shutdown", "Started : Db[%s]", m.DB)

	m.rl.RLock()
	m.m.Close()
	m.rl.RUnlock()
	// key := m.Host + ":" + m.DB
	//
	// // Remove this session from list to master list.
	// masterListLock.Lock()
	// delete(masterList, key)
	// masterListLock.Unlock()
	m.rl.Lock()
	m.m = nil
	m.rl.Unlock()

	m.Log(context, "Shutdown", "Completed")
}

//==============================================================================

// ErrCollectionNoExist is returned when the giving collection does not exists
// in the db.
var ErrCollectionNoExist = fmt.Errorf("Collection does not exist")

// ExecuteDB the MongoDB literal function.
func (m *Mongnod) ExecuteDB(context interface{}, collectionName string, f func(*mgo.Collection) error) error {
	m.Log(context, "executeDB", "Started : Db[%s] : Collection[%s]", m.DB, collectionName)

	if m.m == nil {
		if err := m.connectDB(context); err != nil {
			m.Error(context, "executeDB", err, "Completed")
			return err
		}
	}

	m.rl.RLock()
	ses := m.m
	m.rl.RUnlock()

	// If we have a nil session then return an appropriate error.
	if ses == nil {
		err := errors.New("Invalid Session")
		m.Error(context, "executeDB", err, "Completed")
		return err
	}

	// Retrieve the name for the db we wish to use.
	dbName := m.DB

	// Capture the specified collection.
	col := ses.DB(dbName).C(collectionName)
	if col == nil {
		m.Error(context, "executeDB", ErrCollectionNoExist, "Completed")
		return ErrCollectionNoExist
	}

	m.Log(context, "executeDB", "Completed")
	// Execute the MongoDB function and return possible error.
	return f(col)
}

//==========================================================================================

// connectDB connects and initializes the master session for the mongo list.
func (m *Mongnod) connectDB(context interface{}) error {
	m.Log(context, "connectDB", "Started : Config : %s", m.Query(m.Config))

	key := m.Host + ":" + m.DB

	var new bool

	masterListLock.Lock()
	ms, ok := masterList[key]
	masterListLock.Unlock()

	// If not found, then attemp to connect and add to session master list.
	if !ok {

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
			m.Error(context, "connectDB", err, "Completed")
			return err
		}

		ses.SetMode(mgo.Monotonic, true)

		m.rl.Lock()
		m.m = ses
		m.rl.Unlock()

		new = true

		// Add to master list.
		masterListLock.Lock()
		masterList[key] = ses.Copy()
		masterListLock.Unlock()
	}

	if !new {
		m.m = ms.Copy()
	}

	m.Log(context, "connectDB", "Completed")
	return nil
}

//==========================================================================================

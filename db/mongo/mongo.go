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

// Logger provides a logging interface for the coquery package, allowing
// users to provide their own internal logging systems.
type Logger interface {
	User(context interface{}, funcName string, message string, format ...interface{})
	Dev(context interface{}, funcName string, message string, format ...interface{})
	Error(context interface{}, funcName string, err error, message string, format ...interface{})
}

//==============================================================================

// Mongnod defines a mongo connection manager that builds off a mongo instance.
type Mongnod struct {
	*Config
	log Logger
	m   *mgo.Session
}

// New returns a new Mongnod instance.
func New(l Logger, c Config) (*Mongnod, error) {
	m := Mongnod{
		Config: &c,
		log:    l,
	}

	key := c.Host + ":" + c.DB

	var new bool

	masterListLock.Lock()
	ms, ok := masterList[key]
	masterListLock.Unlock()

	// If not found, then attemp to connect and add to session master list.
	if !ok {

		// Attemp to create the mongodb session.
		if err := m.connectDB("mongo.New"); err != nil {
			return nil, err
		}

		new = true

		// Set the session adequately.
		ms = m.m.Copy()

		// Add to master list.
		masterListLock.Lock()
		masterList[key] = ms
		masterListLock.Unlock()
	}

	if !new {
		m.m = ms.Copy()
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

//==============================================================================

// ErrCollectionNoExist is returned when the giving collection does not exists
// in the db.
var ErrCollectionNoExist = fmt.Errorf("Collection does not exist")

// ExecuteDB the MongoDB literal function.
func (m *Mongnod) ExecuteDB(context interface{}, collectionName string, f func(*mgo.Collection) error) error {
	m.log.Dev(context, "executeDB", "Started : Db[%s] : Collection[%s]", m.DB, collectionName)

	ses := m.m.Copy()

	// If we have a nil session then return an appropriate error.
	if ses == nil {
		err := errors.New("Invalid Session")
		m.log.Error(context, "executeDB", err, "Completed")
		return err
	}

	// Retrieve the name for the db we wish to use.
	dbName := m.DB

	// Capture the specified collection.
	col := ses.DB(dbName).C(collectionName)
	if col == nil {
		m.log.Error(context, "executeDB", ErrCollectionNoExist, "Completed")
		return ErrCollectionNoExist
	}

	// Execute the MongoDB function and return possible error.
	return f(col)
}

//==========================================================================================

// connectDB connects and initializes the master session for the mongo list.
func (m *Mongnod) connectDB(context interface{}) error {
	m.log.Dev(context, "connectDB", "Started : Config : %s", m.Query(m.Config))

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
		m.log.Error(context, "connectDB", err, "Completed")
		return err
	}

	ses.SetMode(mgo.Monotonic, true)
	m.m = ses

	m.log.Dev(context, "connectDB", "Completed")
	return nil
}

//==========================================================================================

package providers

import (
	"errors"

	"github.com/influx6/coquery"
	"github.com/influx6/faux/sumex"
)

// MongoProvider provides a central provider for working with mongodb stores.
type MongoProvider struct {
	input   sumex.Streams
	find    sumex.Streams
	collect sumex.Streams
	mutate  sumex.Streams
}

// NewMongoProvider returns a new instance of the MongoProvider.
func NewMongoProvider(workers int) *MongoProvider {
	mp := MongoProvider{
		input: sumex.Identity(workers),
	}

	mp.find = mp.input.Stream(sumex.New(workers, &FindProc{}))

	return &mp
}

// Handle sends a requests into the mongo provider.
func (m *MongoProvider) Handle(req interface{}) {
	m.input.Inject(req)
}

// FindProc provides a find working for handling find requests.
type FindProc struct {
	root MongoProvider
}

// Do performs the necessary tasks passed to FindProc
func (f FindProc) Do(data interface{}, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}

	find, ok := data.(*coquery.Find)
	if !ok {
		return nil, errors.New("Invalid Query Type")
	}

	return f.find(find), nil
}

// find takes a coquery.Find instance and produces the appropriate response.
func (FindProc) find(f *coquery.Find) *coquery.Response {
	return nil
}

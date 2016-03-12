package coquery

//==============================================================================

// BasicQueries provides a base level query processsor for the coquery library.
type BasicQueries struct{ EventLog }

// Generate takes the underline queries and generates the corresponding query
// objects matching the giving functions, if it finds an unrecognized function,
// it returns a ResponseError instead.
func (b *BasicQueries) Generate(context interface{}, queries []string) (RecordRequests, ResponseError) {
	b.Log(context, "BasicQueries.Generate", "Started : Queries : %s", queries)

	b.Log(context, "BasicQueries.Generate", "Completed")
	return nil, nil
}

//==============================================================================

// Find defines a record retrieve request based on the KV query.
type Find struct {
	Doc   string      `json:"doc" bson:"doc"`
	RID   string      `json:"rid" bson:"rid"`
	Key   string      `json:"key" bson:"key"`
	Value interface{} `json:"value" bson:"value"`
}

// RequestName returns the name for the giving request type.
func (f Find) RequestName() string {
	return "find"
}

//==============================================================================

// Collect retrieves specific keyed items from the coquery stores.
type Collect struct {
	RID  string   `json:"rid" bson:"rid"`
	Keys []string `json:"keys" bson:"keys"`
}

// RequestName returns the name for the giving request type.
func (f Collect) RequestName() string {
	return "collect"
}

//==============================================================================

// Mutate provides json data to be saved/augmented into a new version of the
// current document.
type Mutate struct {
	RID  string `json:"rid" bson:"rid"`
	Data []byte `json:"mutate" bson:"mutate"`
}

// RequestName returns the name for the giving request type.
func (f Mutate) RequestName() string {
	return "mutate"
}

//==============================================================================

package coquery

//==============================================================================

// Find defines a record retrieve request based on the KV query.
type Find struct {
	ID    string
	Key   string
	Value interface{}
	Next  []string // the next query to be executed after this query
}

// Collect retrieves specific keyed items from the coquery stores.
type Collect struct {
	ID   string
	Keys []string
	Next []string // the next query to be executed after this query
}

// Mutate provides json data to be saved/augmented into a new version of the
// current document.
type Mutate struct {
	ID   string
	Data []byte
	Next []string // the next query to be executed after this query
}

// Response provides a response struct for replies to coquery requests.
type Response struct {
	Type  string `json:"type" bson:"type"`
	ReqID string `json:"req_id" bson:"req_id"`
	Query string `json:"query" bson:"query"`
	Reply []byte `json:"reply" bson:"reply"`
}

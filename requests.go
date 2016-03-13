package coquery

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/influx6/coquery/parser"
)

//==============================================================================

// BasicQueries provides a base level query processsor for the coquery library.
type BasicQueries struct{ EventLog }

// Generate takes the underline queries and generates the corresponding query
// objects matching the giving functions, if it finds an unrecognized function,
// it returns a ResponseError instead.
func (b *BasicQueries) Generate(context interface{}, reqid string, doc string, queries []string) (RecordRequests, ResponseError) {
	b.Log(context, "BasicQueries.Generate", "Started : Doc[%s] : Queries : %s", doc, queries)

	var reqs RecordRequests

	for _, qs := range queries {

		// Get the request name and its parameters
		method, _, params := parser.SplitQuery(context, qs)

		switch strings.ToLower(method) {
		case "findn":

			switch len(params) {
			case 0:
				reqs = append(reqs, &FindN{
					Doc:    doc,
					RID:    reqid,
					Skip:   0,
					Amount: -1,
				})
				continue
			case 1:
				count, err := strconv.Atoi(params[0])
				if err != nil {
					err := &CoError{
						Rid:    reqid,
						Msg:    fmt.Sprintf("Invalid Integer String"),
						IError: err,
					}

					b.Error(context, "BasicQueries.Generate", err, "Completed")
					return nil, err
				}

				reqs = append(reqs, &FindN{
					Doc:    doc,
					RID:    reqid,
					Amount: count,
					Skip:   0,
				})
				continue
			default:
				count, err := strconv.Atoi(params[0])
				if err != nil {
					err := &CoError{
						Rid:    reqid,
						Msg:    fmt.Sprintf("Invalid Integer String for Count"),
						IError: err,
					}

					b.Error(context, "BasicQueries.Generate", err, "Completed")
					return nil, err
				}

				skip, err := strconv.Atoi(params[1])
				if err != nil {
					err := &CoError{
						Rid:    reqid,
						Msg:    fmt.Sprintf("Invalid Integer String for Skip"),
						IError: err,
					}

					b.Error(context, "BasicQueries.Generate", err, "Completed")
					return nil, err
				}

				reqs = append(reqs, &FindN{
					Doc:    doc,
					RID:    reqid,
					Amount: count,
					Skip:   skip,
				})

				continue
			}

		case "find":

			switch len(params) {
			case 0:
				err := &CoError{
					Rid:    reqid,
					Msg:    fmt.Sprintf("Expected key information"),
					IError: fmt.Errorf("find requires record key as first argument"),
				}

				b.Error(context, "BasicQueries.Generate", err, "Completed")
				return nil, err

			case 1:
				err := &CoError{
					Rid:    reqid,
					Msg:    fmt.Sprintf("Expected value information"),
					IError: fmt.Errorf("find requires record value as second argument"),
				}

				b.Error(context, "BasicQueries.Generate", err, "Completed")
				return nil, err

			default:
				reqs = append(reqs, &Find{
					Doc:   doc,
					RID:   reqid,
					Key:   params[0],
					Value: params[1],
				})
				continue
			}

		case "collect":

			reqs = append(reqs, &Collect{
				RID:  reqid,
				Keys: params,
			})
			continue

		case "mutate":

			if len(params) == 0 {
				err := &CoError{
					Rid:    reqid,
					Msg:    fmt.Sprintf("Expected JSON data"),
					IError: fmt.Errorf("Mutate requires json data as argument"),
				}

				b.Error(context, "BasicQueries.Generate", err, "Completed")
				return nil, err
			}

			pm := make(Parameter)

			if err := json.Unmarshal([]byte(params[0]), &pm); err != nil {
				err := &CoError{
					Rid:    reqid,
					Msg:    fmt.Sprintf("Invalid JSON"),
					IError: err,
				}

				b.Error(context, "BasicQueries.Generate", err, "Completed")
				return nil, err
			}

			reqs = append(reqs, &Mutate{
				RID:       reqid,
				Parameter: pm,
			})

			continue

		default:
			err := &CoError{
				Rid:    reqid,
				Msg:    fmt.Sprintf("Invalid Query Method[%s]", method),
				IError: errors.New("404"),
			}

			b.Error(context, "BasicQueries.Generate", err, "Completed")
			return nil, err
		}
	}

	b.Log(context, "BasicQueries.Generate", "Completed")
	return reqs, nil
}

//==============================================================================

// FindN defines a record request to retrieve data based on a set amount.
type FindN struct {
	Doc    string `json:"doc" bson:"doc"`
	RID    string `json:"rid" bson:"rid"`
	Skip   int    `json:"skip" bson:"skip"`
	Amount int    `json:"limit" bson:"limit"`
}

// RequestName returns the name for the giving request type.
func (f FindN) RequestName() string {
	return "findN"
}

// Example returns a string that showcase a sample of this request.
// In truth this provides a code-level sample information and nothing more.
func (f FindN) Example() []string {
	return []string{"findN(10,20)", "findN(-1)", "findN(10)"}
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

// Example returns a string that showcase a sample of this request.
// In truth this provides a code-level sample information and nothing more.
func (f Find) Example() []string {
	return []string{"find(id,4023)", "find(name,'alex')"}
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

// Example returns a string that showcase a sample of this request.
// In truth this provides a code-level sample information and nothing more.
func (f Collect) Example() []string {
	return []string{"collect(name,age,created_at)"}
}

//==============================================================================

// Mutate provides json data to be saved/augmented into a new version of the
// current document.
type Mutate struct {
	RID       string    `json:"rid" bson:"rid"`
	Parameter Parameter `json:"params" bson:"params"`
}

// RequestName returns the name for the giving request type.
func (f Mutate) RequestName() string {
	return "mutate"
}

// Example returns a string that showcase a sample of this request.
func (f Mutate) Example() []string {
	return []string{"mutate({name:'alex'})"}
}

//==============================================================================
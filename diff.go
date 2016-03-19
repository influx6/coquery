package coquery

// Diffs defines an interface for storing store diffs for the coquery system
// that lets us know which records had changed during a request.
type Diffs interface {
	Put([]string)
	All() ([]string, []string)
	Has(id string) bool
	Get(id string) []string
	Clear()
}

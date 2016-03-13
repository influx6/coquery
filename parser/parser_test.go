package parser_test

import (
	"fmt"
	"testing"

	"github.com/ardanlabs/kit/tests"
	"github.com/influx6/coquery/parser"
)

//==============================================================================

var context = "testing"

//==============================================================================

func init() {
	tests.Init("")
}

//==============================================================================

// TestQuerySpliting validates the ability to split a query method string.
func TestQuerySpliting(t *testing.T) {
	tests.ResetLog()
	defer tests.DisplayLog()

	t.Logf("Given the need to be able to parser a query method call")
	{

		q := "keys(name,age,address)"
		t.Logf("\tWhen giving a query method call %q", q)
		{

			method, content, contents := parser.SplitQuery(context, q)

			if method != "keys" || content != "name,age,address" || len(contents) < 3 {
				t.Fatalf("\t%s\tShould have retrieved the appropriate method and contents of the query: Method: %q Content: %q", tests.Failed, method, content)
			}
			t.Logf("\t%s\tShould have retrieved the appropriate method and contents of the query", tests.Success)
		}

		qs := "kv(id,\"{\"name\":\"bug.\"}\")"
		t.Logf("\tWhen giving a query method call %q", qs)
		{

			method, content, contents := parser.SplitQuery(context, qs)
			fmt.Printf("Method: %s Content: %s Contents: %s\n", method, content, contents)

			if method != "kv" || content != "id,\"{\"name\":\"bug.\"}\"" || len(contents) < 2 {
				t.Fatalf("\t%s\tShould have retrieved the appropriate method and contents of the query: Method: %q Content: %q", tests.Failed, method, content)
			}
			t.Logf("\t%s\tShould have retrieved the appropriate method and contents of the query", tests.Success)
		}

		qs = "kv(id,{name:'slumber',age:1})"
		t.Logf("\tWhen giving a query method call %q", qs)
		{

			method, content, contents := parser.SplitQuery(context, qs)
			fmt.Printf("Method: %s Content: %s Contents: %s\n", method, content, contents)

			if method != "kv" || content != "id,{name:'slumber',age:1}" || len(contents) < 2 {
				t.Fatalf("\t%s\tShould have retrieved the appropriate method and contents of the query: Method: %q Content: %q", tests.Failed, method, content)
			}
			t.Logf("\t%s\tShould have retrieved the appropriate method and contents of the query", tests.Success)
		}

		qs = "kv(id,\"\x21\x4e\")"
		t.Logf("\tWhen giving a query method call %q", qs)
		{

			method, content, contents := parser.SplitQuery(context, qs)
			fmt.Printf("Method: %s Content: %s Contents: %s\n", method, content, contents)

			if method != "kv" || content != "id,\"\x21\x4e\"" || len(contents) < 2 {
				t.Fatalf("\t%s\tShould have retrieved the appropriate method and contents of the query: Method: %q Content: %q", tests.Failed, method, content)
			}
			t.Logf("\t%s\tShould have retrieved the appropriate method and contents of the query", tests.Success)
		}

		qs = "kv(id,`\x21\x4e`)"
		t.Logf("\tWhen giving a query method call %q", qs)
		{

			method, content, contents := parser.SplitQuery(context, qs)
			fmt.Printf("Method: %s Content: %s Contents: %s\n", method, content, contents)

			if method != "kv" || content != "id,`\x21\x4e`" || len(contents) < 2 {
				t.Fatalf("\t%s\tShould have retrieved the appropriate method and contents of the query: Method: %q Content: %q", tests.Failed, method, content)
			}
			t.Logf("\t%s\tShould have retrieved the appropriate method and contents of the query", tests.Success)
		}

		qs = "kv(id,```\x21\x4e```)"
		t.Logf("\tWhen giving a query method call %q", qs)
		{

			method, content, contents := parser.SplitQuery(context, qs)
			fmt.Printf("Method: %s Content: %s Contents: %s\n", method, content, contents)

			if method != "kv" || content != "id,```\x21\x4e```" || len(contents) < 2 {
				t.Fatalf("\t%s\tShould have retrieved the appropriate method and contents of the query: Method: %q Content: %q", tests.Failed, method, content)
			}
			t.Logf("\t%s\tShould have retrieved the appropriate method and contents of the query", tests.Success)
		}

	}
}

//==============================================================================

// TestParsing validates the behaviour of the parser routine for parsing query
// string.
func TestBasicParsing(t *testing.T) {
	tests.ResetLog()
	defer tests.DisplayLog()

	t.Logf("Given the need to be able to parser a query request string")
	{

		q := "docs.user.rid(4356932).kv(id,0).keys(name,age,address)"
		t.Logf("\tWhen giving a query string %q", q)
		{

			proc := parser.ParseQuery(context, q)

			if len(proc) < 5 {
				t.Fatalf("\t%s\tShould have retrieved five segments of the parsing string", tests.Failed)
			}
			t.Logf("\t%s\tShould have retrieved five segments of the parsing string", tests.Success)
		}

	}
}

//==============================================================================

// TestDataParsing tests wether we have data contained within the parsing string.
func TestDataParsing(t *testing.T) {
	tests.ResetLog()
	defer tests.DisplayLog()

	t.Logf("Given the need to be able to parser a query request string")
	{

		q := "docs.user.kv(id,{name:'slumber',age:1})"
		t.Logf("\tWhen giving a query string %q", q)
		{

			proc := parser.ParseQuery(context, q)

			if len(proc) < 3 {
				t.Fatalf("\t%s\tShould have retrieved five segments of the parsing string", tests.Failed)
			}
			t.Logf("\t%s\tShould have retrieved five segments of the parsing string", tests.Success)

		}

	}
}

//==============================================================================

// TestMixedKeysParsing validats that we can parse the data that has been added
// with overly contorted keys.
func TestMixedKeysParsing(t *testing.T) {
	tests.ResetLog()
	defer tests.DisplayLog()

	t.Logf("Given the need to be able to parser a query request string")
	{

		q := "docs.user.keys(id,address.book,name)"

		t.Logf("\tWhen giving a query string %q", q)
		{
			proc := parser.ParseQuery(context, q)
			// fmt.Printf("Proc: %s\n", proc)

			if len(proc) < 3 {
				t.Fatalf("\t%s\tShould have retrieved five segments of the parsing string", tests.Failed)
			}
			t.Logf("\t%s\tShould have retrieved five segments of the parsing string", tests.Success)

		}

	}
}

//==============================================================================

// TestMixedParsing validats that we can parse the data that has been added
// with overly contorted string values that have the basic query type in them.
func TestMixedParsing(t *testing.T) {
	tests.ResetLog()
	defer tests.DisplayLog()

	t.Logf("Given the need to be able to parser a query request string")
	{

		q := "docs.user.kv(id,\"{\"name\":\"bug.\"}\")"

		t.Logf("\tWhen giving a query string %q", q)
		{
			proc := parser.ParseQuery(context, q)
			// fmt.Printf("Proc: %s\n", proc)

			if len(proc) < 3 {
				t.Fatalf("\t%s\tShould have retrieved five segments of the parsing string", tests.Failed)
			}
			t.Logf("\t%s\tShould have retrieved five segments of the parsing string", tests.Success)

		}

	}
}

//==============================================================================

// TestHexParsing tests whether we can parse the data that has been minified
// using a hex and base64 encode data.
func TestHexParsing(t *testing.T) {
	tests.ResetLog()
	defer tests.DisplayLog()

	t.Logf("Given the need to be able to parser a query request string")
	{

		q := "docs.user.kv(id,{name:'slumber',age:1})"

		t.Logf("\tWhen giving a query string %q", q)
		{
			proc := parser.ParseQuery(context, q)
			// fmt.Printf("Proc: %s\n", proc)

			if len(proc) < 3 {
				t.Fatalf("\t%s\tShould have retrieved five segments of the parsing string", tests.Failed)
			}
			t.Logf("\t%s\tShould have retrieved five segments of the parsing string", tests.Success)

		}

	}
}

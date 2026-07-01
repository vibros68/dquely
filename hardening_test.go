package dquely_test

import (
	"strings"
	"testing"

	"github.com/vibros68/dquely"
)

// Bug #2 regression: when the root struct already has a uid, ParseMutation must
// address it by <uid> instead of emitting a blank node (which would create a new
// node instead of updating the existing one).
func TestParseMutationNestedRootUsesUID(t *testing.T) {
	company := &Company{
		Uid:   "0x1",
		Name:  "Acme",
		Owner: &ShortUser{Uid: "0x2"},
	}
	_, mu, err := dquely.ParseMutation(company, true)
	if err != nil {
		t.Fatal(err)
	}
	set := string(mu[0].SetNquads)
	if !strings.Contains(set, `<0x1> <name> "Acme" .`) {
		t.Errorf("expected root addressed by <0x1>, got:\n%s", set)
	}
	if strings.Contains(set, "_:company") {
		t.Errorf("root with uid must not emit _:company blank node, got:\n%s", set)
	}
}

// The upsert helpers must escape literal values just like the query builder and
// ParseMutation, so a value containing a quote cannot break the N-quad syntax.
func TestUpsertEscapesValue(t *testing.T) {
	type upsertDoc struct {
		Uid  string `dquely:"uid"`
		Name string `dquely:"name"`
	}
	got, err := dquely.Upsert(&upsertDoc{Name: `a"b`}, dquely.Eq("id", "1"), "name")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `a\"b`) {
		t.Errorf("expected escaped value in upsert, got:\n%s", got)
	}
	if strings.Contains(got, `"a"b"`) {
		t.Errorf("unescaped value breaks N-quad syntax, got:\n%s", got)
	}
}

// Injection hardening: special characters in filter values must be escaped so a
// value cannot break out of the quoted literal and inject filter syntax.
func TestQueryFilterEscapesInjection(t *testing.T) {
	q := dquely.NewDQL("me").
		Type("Person").
		Eq("name", `x" OR eq(uid, 0x1)`).
		Query()
	if strings.Contains(q, `"x" OR eq(uid, 0x1)"`) {
		t.Errorf("injection not escaped, got:\n%s", q)
	}
	if !strings.Contains(q, `\"`) {
		t.Errorf("expected escaped quote in query, got:\n%s", q)
	}
}

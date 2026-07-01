package dquely

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dgraph-io/dgo/v250/protos/api"
)

func TestEscapeString(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`plain`, `plain`},
		{`a"b`, `a\"b`},
		{`a\b`, `a\\b`},
		{"line1\nline2", `line1\nline2`},
		{"tab\there", `tab\there`},
		{`" OR eq(uid, 0x1) "`, `\" OR eq(uid, 0x1) \"`},
	}
	for _, c := range cases {
		if got := escapeString(c.in); got != c.want {
			t.Errorf("escapeString(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEscapeRegexpDelim(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`abc`, `abc`},
		{`a/b`, `a\/b`},
		{`a\/b`, `a\/b`},    // already escaped: left as-is
		{`a\\/b`, `a\\\/b`}, // escaped backslash then a bare slash
	}
	for _, c := range cases {
		if got := escapeRegexpDelim(c.in); got != c.want {
			t.Errorf("escapeRegexpDelim(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

type execUser struct {
	Uid  string `dquely:"uid"`
	Name string `dquely:"name,unique"`
}

func TestRootUID(t *testing.T) {
	if got := rootUID(&execUser{Uid: "0x1"}); got != "0x1" {
		t.Errorf("rootUID with uid set = %q, want 0x1", got)
	}
	if got := rootUID(&execUser{}); got != "" {
		t.Errorf("rootUID with empty uid = %q, want empty", got)
	}
	if got := rootUID((*execUser)(nil)); got != "" {
		t.Errorf("rootUID(nil) = %q, want empty", got)
	}
}

type mockRunner struct {
	resp   *api.Response
	err    error
	gotReq *api.Request
}

func (m *mockRunner) Do(_ context.Context, req *api.Request) (*api.Response, error) {
	m.gotReq = req
	return m.resp, m.err
}

// Bug #1 regression: updating a node that already has a uid must not be reported
// as ErrDuplicate just because resp.Uids is empty (an update creates no new UID).
func TestExecMutateUpdateSucceedsWithoutBlankNode(t *testing.T) {
	d := &Dgo{}
	runner := &mockRunner{resp: &api.Response{Uids: map[string]string{}}}
	user := &execUser{Uid: "0x1", Name: "Alice"}
	if err := d.execMutate(context.Background(), runner, true, user); err != nil {
		t.Fatalf("update with uid should succeed, got %v", err)
	}
	// SetNquads must address the existing node by <uid>, never a blank node.
	set := string(runner.gotReq.Mutations[0].SetNquads)
	if !strings.Contains(set, "<0x1> <name>") {
		t.Errorf("expected update to target <0x1>, got: %s", set)
	}
	if strings.Contains(set, "_:") {
		t.Errorf("update must not emit a blank node, got: %s", set)
	}
}

// An insert (no uid) whose unique condition matched an existing node yields an
// empty resp.Uids and must surface ErrDuplicate.
func TestExecMutateInsertDuplicate(t *testing.T) {
	d := &Dgo{}
	runner := &mockRunner{resp: &api.Response{Uids: map[string]string{}}}
	user := &execUser{Name: "Alice"}
	err := d.execMutate(context.Background(), runner, true, user)
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

type execChild struct {
	Uid  string `dquely:"uid"`
	Name string `dquely:"name"`
}

type execParent struct {
	Uid  string      `dquely:"uid"`
	Name string      `dquely:"name"`
	Kids []execChild `dquely:"kids"`
}

// execUpdate must write back UIDs created for nested blank-node children (slice
// elements that had no uid) so callers see the newly created nodes.
func TestExecUpdateInjectsNestedUIDs(t *testing.T) {
	d := &Dgo{}
	runner := &mockRunner{resp: &api.Response{Uids: map[string]string{"kids0": "0x7"}}}
	parent := &execParent{Uid: "0x1", Name: "P", Kids: []execChild{{Name: "K"}}}
	if err := d.execUpdate(context.Background(), runner, true, parent, "name", "kids"); err != nil {
		t.Fatalf("update should succeed, got %v", err)
	}
	if parent.Kids[0].Uid != "0x7" {
		t.Errorf("expected nested child uid 0x7 written back, got %q", parent.Kids[0].Uid)
	}
}

// A successful insert returns the new UID keyed by the blank node; it must be
// written back into the struct's uid field.
func TestExecMutateInsertSuccess(t *testing.T) {
	d := &Dgo{}
	user := &execUser{Name: "Alice"}
	blank, err := BlankNodeName(user)
	if err != nil {
		t.Fatal(err)
	}
	runner := &mockRunner{resp: &api.Response{Uids: map[string]string{blank: "0x9"}}}
	if err := d.execMutate(context.Background(), runner, true, user); err != nil {
		t.Fatalf("insert should succeed, got %v", err)
	}
	if user.Uid != "0x9" {
		t.Errorf("expected uid 0x9 written back, got %q", user.Uid)
	}
}

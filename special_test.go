package dquely_test

import (
	"github.com/vibros68/dquely"
	"testing"
)

type Companies struct {
	Uid         string       `json:"uid,omitempty" dquely:"uid"`
	Name        string       `json:"name,omitempty" dquely:"name"`
	Slug        string       `json:"slug,omitempty" dquely:"slug,unique"`
	Memberships []Membership `json:"users,omitempty" dquely:"users"`
}

type Membership struct {
	Uid     string     `json:"uid,omitempty" dquely:"uid"`
	IsOwner bool       `json:"isOwner,omitempty" dquely:"isOwner"`
	User    *User      `json:"user,omitempty" dquely:"user"`
	Company *Companies `json:"company,omitempty" dquely:"company"`
}

func TestSpecialDeepMultiMutation(t *testing.T) {
	var company = Companies{
		Name: "A",
		Slug: "a",
		Memberships: []Membership{
			{
				IsOwner: true,
				User:    &User{Uid: "0x1"},
			},
		},
	}
	query, muConds, err := dquely.ParseMutation(&company, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	const expectedQuery = `{
  v as var(func: type(Companies))
    @filter(eq(slug, "a"))
}
`
	if query != expectedQuery {
		t.Fatalf("expected ParseMutation() to get query be %s, got %s", expectedQuery, query)
	}
	var cond = muConds[0]
	const expectedCond = `@if(eq(len(v), 0))`
	if cond.Cond != expectedCond {
		t.Fatalf("expected ParseMutation() to get Condition be %s, got %s", expectedCond, cond.Cond)
	}
	const expectedSet = `_:companies <name> "A" .
_:companies <slug> "a" .
_:companies <users> _:users0 .
_:companies <dgraph.type> "Companies" .
_:users0 <isOwner> "true" .
_:users0 <user> <0x1> .
_:users0 <dgraph.type> "Membership" .`
	if string(cond.SetNquads) != expectedSet {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", expectedSet, string(cond.SetNquads))
	}
	if string(cond.DelNquads) != "" {
		t.Errorf("expected ParseMutation() to get Mutation be empty, got %s", string(cond.SetNquads))
	}
	// test SetUIDs
	err = dquely.SetUIDs(&company, map[string]string{"company": "0xc352", "memberships0": "0xc353"})
	if err != nil {
		t.Fatalf("dquely.SetUIDs: expect error to be nil got %v", err)
	}
	if company.Uid != "0xc352" {
		t.Errorf("expected SetUIDs to set company.Uid = 0xc352, got %s", company.Uid)
	}
	if company.Memberships[0].Uid != "0xc353" {
		t.Errorf("expected SetUIDs to set company.Memberships[0].Uid = 0xc353, got %s",
			company.Memberships[0].Uid)
	}
}

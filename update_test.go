package dquely_test

import (
	"github.com/vibros68/dquely"
	"testing"
	"time"
)

func TestMutationUserFieldBuilder(t *testing.T) {
	user := &UserFieldBuilder{
		Uid:   "0x1",
		Name:  "Alice",
		Age:   29,
		Roles: map[string]int{"company": 2, "user": 1},
	}
	query, mu, err := dquely.ParseUpdate(user, dquely.FieldAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(mu) != 1 {
		t.Fatalf("expected 1 mutations, got %d", len(mu))
	}
	const expectedQuery = `{
  v as var(func: uid(0x1))
    @filter(type(User))
}`
	if query != expectedQuery {
		t.Errorf("expected query to be %s, got %s", expectedQuery, query)
	}
	const expectedCond = `@if(eq(len(v), 1))`
	if mu[0].Cond != expectedCond {
		t.Errorf("expected cond to be %s, got %s", expectedCond, mu[0].Cond)
	}
	if len(mu[0].DelNquads) != 0 {
		t.Fatalf("expected 0 DelNquads, got %d", len(mu[0].DelNquads))
	}
	if string(mu[0].SetNquads) != userFieldBuilderMutationMock {
		t.Errorf("expected SetNquads to return %s, got %s",
			userFieldBuilderMutationMock, string(mu[0].SetNquads))
	}
}

func TestMutationWithDgraphType(t *testing.T) {
	user := &UserDgraph{Uid: "0x1", Name: "Alice", Age: 29, Email: "alice@example.com"}
	query, mu, err := dquely.ParseUpdate(user, "name", "age")
	if err != nil {
		t.Fatal(err)
	}
	if len(mu) != 1 {
		t.Fatalf("expected 1 mutations, got %d", len(mu))
	}
	const expectedQuery = `{
  v as var(func: uid(0x1))
    @filter(type(User))
}`
	if query != expectedQuery {
		t.Errorf("expected query to be %s, got %s", expectedQuery, query)
	}
	const expectedCond = `@if(eq(len(v), 1))`
	if mu[0].Cond != expectedCond {
		t.Errorf("expected cond to be %s, got %s", expectedCond, mu[0].Cond)
	}
	if len(mu[0].DelNquads) != 0 {
		t.Fatalf("expected 0 DelNquads, got %d", len(mu[0].DelNquads))
	}
	if string(mu[0].SetNquads) != userMutationMock {
		t.Errorf("expected Mutation() to return %s, got %s", userMutationMock, string(mu[0].SetNquads))
	}
}

func TestMutationUserLack(t *testing.T) {
	var amount uint64 = 1000
	user := &UserLack{Uid: "0x1", Name: "Alice", Age: 29, Email: "alice@example.com", Amount: &amount}
	query, mu, err := dquely.ParseUpdate(user, dquely.FieldAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(mu) != 1 {
		t.Fatalf("expected 1 mutations, got %d", len(mu))
	}
	const expectedQuery = `{
  v as var(func: uid(0x1))
    @filter(type(User))
}`
	if query != expectedQuery {
		t.Errorf("expected query to be %s, got %s", expectedQuery, query)
	}
	const expectedCond = `@if(eq(len(v), 1))`
	if mu[0].Cond != expectedCond {
		t.Errorf("expected cond to be %s, got %s", expectedCond, mu[0].Cond)
	}
	if len(mu[0].DelNquads) != 0 {
		t.Fatalf("expected 0 DelNquads, got %d", len(mu[0].DelNquads))
	}
	const expectedSet = `uid(v) <name> "Alice" .
uid(v) <Age> "29" .
uid(v) <Email> "alice@example.com" .
uid(v) <Amount> "1000" .`
	if string(mu[0].SetNquads) != expectedSet {
		t.Errorf("expected Mutation() to return %s, got %s", expectedSet, string(mu[0].SetNquads))
	}
}

func TestUpdateRelationship(t *testing.T) {
	var company = Company{
		Uid:  "0x1",
		Name: "A",
		Owner: &ShortUser{
			Uid: "0x2",
		},
	}
	query, muConds, err := dquely.ParseUpdate(&company, "name", "owner")
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	const expectedQuery = `{
  v as var(func: uid(0x1))
    @filter(type(Company))
}`
	if query != expectedQuery {
		t.Errorf("expected ParseUpdate() to get query %s, got %s", expectedQuery, query)
	}
	var cond = muConds[0]
	const expectedCond = `@if(eq(len(v), 1))`
	if cond.Cond != expectedCond {
		t.Errorf("expected ParseUpdate() to get Condition be %s, got %s", expectedCond, cond.Cond)
	}
	const expectedNquads = `uid(v) <name> "A" .
uid(v) <owner> <0x2> .`
	if string(cond.SetNquads) != expectedNquads {
		t.Errorf("expected ParseUpdate() to get Mutation %s, got %s", expectedNquads,
			string(cond.SetNquads))
	}
	const expectedDelNquads = `uid(v) <owner> * .`
	if string(cond.DelNquads) != expectedDelNquads {
		t.Errorf("expected ParseUpdate() to get Mutation %s, got %s", expectedDelNquads,
			string(cond.DelNquads))
	}
}

func TestUpdateRelationship_2(t *testing.T) {
	var company = Company{
		Uid:  "0x1",
		Name: "A",
		Owner: &ShortUser{
			Uid: "0x2",
		},
		Staffs: []ShortUser{
			{Uid: "0x3"},
			{Uid: "0x4"},
		},
	}
	query, muConds, err := dquely.ParseUpdate(&company, "name", "owner", "staffs")
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	const expectedQuery = `{
  v as var(func: uid(0x1))
    @filter(type(Company))
}`
	if query != expectedQuery {
		t.Errorf("expected ParseUpdate() to get query %s, got %s", expectedQuery, query)
	}
	var cond = muConds[0]
	const expectedCond = `@if(eq(len(v), 1))`
	if cond.Cond != expectedCond {
		t.Errorf("expected ParseUpdate() to get Condition be %s, got %s", expectedCond, cond.Cond)
	}
	const expectedNquads = `uid(v) <name> "A" .
uid(v) <owner> <0x2> .
uid(v) <staffs> <0x3> .
uid(v) <staffs> <0x4> .`
	if string(cond.SetNquads) != expectedNquads {
		t.Errorf("expected ParseUpdate() to get Mutation %s, got %s", expectedNquads,
			string(cond.SetNquads))
	}
	const expectedDelNquads = `uid(v) <owner> * .
uid(v) <staffs> * .`
	if string(cond.DelNquads) != expectedDelNquads {
		t.Errorf("expected ParseUpdate() to get Mutation %s, got %s", expectedDelNquads,
			string(cond.DelNquads))
	}
}

type Product struct {
	Uid       string    `json:"uid,omitempty" dquely:"uid"`
	Name      string    `json:"name,omitempty" dquely:"name"`
	Bio       string    `json:"bio,omitempty" dquely:"bio"`
	CreatedAt time.Time `json:"createdAt,omitempty" dquely:"createdAt"`
	Price     uint64    `json:"price,omitempty" dquely:"price"`
	Medias    []*PMedia `json:"medias,omitempty" dquely:"medias"`
	Stores    []*PStore `json:"stores,omitempty" dquely:"stores"`
	ProductOf *Company  `json:"productOf,omitempty" dquely:"productOf"`
}

type PMedia struct {
	Uid string `json:"uid,omitempty" dquely:"uid"`
}

type PStore struct {
	Uid string `json:"uid,omitempty" dquely:"uid"`
}

func TestUpdateRelationship_3(t *testing.T) {
	var company = Product{
		Uid:   "0xea6e",
		Name:  "Táo Mèo",
		Bio:   "Táo Mèo ăn chua nhưng ngâm siro thì tuyệt cú mèo",
		Price: 5000,
		Medias: []*PMedia{
			{Uid: "0xea6f"},
			{Uid: "0xea70"},
		},
		Stores: []*PStore{
			{Uid: "0xea6a"},
		},
	}
	query, muConds, err := dquely.ParseUpdate(&company, "name", "bio", "price", "medias", "stores")
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	const expectedQuery = `{
  v as var(func: uid(0xea6e))
    @filter(type(Product))
}`
	if query != expectedQuery {
		t.Errorf("expected ParseUpdate() to get query %s, got %s", expectedQuery, query)
	}
	var cond = muConds[0]
	const expectedCond = `@if(eq(len(v), 1))`
	if cond.Cond != expectedCond {
		t.Errorf("expected ParseUpdate() to get Condition be %s, got %s", expectedCond, cond.Cond)
	}
	const expectedNquads = `uid(v) <name> "Táo Mèo" .
uid(v) <bio> "Táo Mèo ăn chua nhưng ngâm siro thì tuyệt cú mèo" .
uid(v) <price> "5000" .
uid(v) <medias> <0xea6f> .
uid(v) <medias> <0xea70> .
uid(v) <stores> <0xea6a> .`
	if string(cond.SetNquads) != expectedNquads {
		t.Errorf("expected ParseUpdate() to get Mutation %s, got %s", expectedNquads,
			string(cond.SetNquads))
	}
	const expectedDelNquads = ``
	if string(cond.DelNquads) != expectedDelNquads {
		t.Errorf("expected ParseUpdate() to get Mutation %s, got %s", expectedDelNquads,
			string(cond.DelNquads))
	}
}

func TestUpdateRelationship_4(t *testing.T) {
	var company = Company{
		Uid:  "0x1",
		Name: "A",
		Owner: &ShortUser{
			Uid: "0x2",
		},
		Staffs: []ShortUser{
			{Name: "ShortUser1", Link: &User{Uid: "0x3"}},
			{Name: "ShortUser2", Link: &User{Uid: "0x4"}},
		},
	}
	query, muConds, err := dquely.ParseUpdate(&company, "name", "owner", "staffs")
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	const expectedQuery = `{
  v as var(func: uid(0x1))
    @filter(type(Company))
}`
	if query != expectedQuery {
		t.Errorf("expected ParseUpdate() to get query %s, got %s", expectedQuery, query)
	}
	var cond = muConds[0]
	const expectedCond = `@if(eq(len(v), 1))`
	if cond.Cond != expectedCond {
		t.Errorf("expected ParseUpdate() to get Condition be %s, got %s", expectedCond, cond.Cond)
	}
	const expectedNquads = `uid(v) <name> "A" .
uid(v) <owner> <0x2> .
uid(v) <staffs> _:shortUser0 .
uid(v) <staffs> _:shortUser1 .
_:shortUser0 <name> "ShortUser1" .
_:shortUser0 <link> <0x3> .
_:shortUser1 <name> "ShortUser2" .
_:shortUser1 <link> <0x4> .`
	if string(cond.SetNquads) != expectedNquads {
		t.Errorf("expected ParseUpdate() to get Mutation %s, got %s", expectedNquads,
			string(cond.SetNquads))
	}
	const expectedDelNquads = `uid(v) <owner> * .
uid(v) <staffs> * .`
	if string(cond.DelNquads) != expectedDelNquads {
		t.Errorf("expected ParseUpdate() to get Mutation %s, got %s", expectedDelNquads,
			string(cond.DelNquads))
	}
}

func TestUpdateRelationship_5(t *testing.T) {
	var amount int64 = 21000
	var company = Order{
		Uid:         "0xeaa5",
		Description: "",
		CreatedAt:   testTime,
		UpdatedAt:   testTime,
		FinishedAt:  &testTime,
		Status:      2,
		Items:       nil,
		Taxes: []TaxItem{
			{Uid: "", RootAmount: 10000, TaxValue: 10, Amount: 5000, Name: "Alice", TaxOf: &Tax{Uid: "0x2"}},
		},
		FinalAmount: &amount,
	}
	query, muConds, err := dquely.ParseUpdate(&company, "status", "updatedAt", "finishedAt", "finalAmount", "taxes")
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	const expectedQuery = `{
  v as var(func: uid(0xeaa5))
    @filter(type(Order))
}`
	if query != expectedQuery {
		t.Errorf("expected ParseUpdate() to get query %s, got %s", expectedQuery, query)
	}
	var cond = muConds[0]
	const expectedCond = `@if(eq(len(v), 1))`
	if cond.Cond != expectedCond {
		t.Errorf("expected ParseUpdate() to get Condition be %s, got %s", expectedCond, cond.Cond)
	}
	const expectedNquads = `uid(v) <updatedAt> "2026-03-07T13:10:31" .
uid(v) <finishedAt> "2026-03-07T13:10:31" .
uid(v) <status> "2" .
uid(v) <taxes> _:taxItem0 .
uid(v) <finalAmount> "21000" .
_:taxItem0 <rootAmount> "10000" .
_:taxItem0 <taxValue> "10" .
_:taxItem0 <amount> "5000" .
_:taxItem0 <name> "Alice" .
_:taxItem0 <taxOf> <0x2> .`
	if string(cond.SetNquads) != expectedNquads {
		t.Errorf("expected ParseUpdate() to get Mutation %s, got %s", expectedNquads,
			string(cond.SetNquads))
	}
	const expectedDelNquads = `uid(v) <taxes> * .`
	if string(cond.DelNquads) != expectedDelNquads {
		t.Errorf("expected ParseUpdate() to get Mutation %s, got %s", expectedDelNquads,
			string(cond.DelNquads))
	}
}

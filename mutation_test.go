package dquely_test

import (
	"testing"

	"github.com/vibros68/dquely"
)

func TestMutation(t *testing.T) {
	user := &User{Name: "Alice", Age: 29, Email: "alice@example.com"}
	result, err := dquely.Mutation(user)
	if err != nil {
		t.Fatal(err)
	}
	if result != userMutationMock {
		t.Errorf("expected Mutation() to return %s, got %s", userMutationMock, result)
	}
}

func TestUpsert(t *testing.T) {
	user := User{Name: "Alice Sayum", Age: 30, Email: "alice@example.com"}
	result, err := dquely.Upsert(user, dquely.Eq("email", "alice@example.com"), "age", "name")
	if err != nil {
		t.Fatal(err)
	}
	if result != userPartialMutationMock {
		t.Errorf("expected Upsert() to return %s, got %s", userPartialMutationMock, result)
	}
}

func TestUpsertWithSetAndDeleteAtSameTime(t *testing.T) {
	q := dquely.NewDQL("").BlockVar("v").Has("age").Select("a as age")
	result := dquely.UpsertBlock("var", q,
		dquely.TripleSetVal("v", "other", "a"),
		dquely.TripleDelete("v", "age"),
	)
	if result != upsertWithSetAndDeleteAtSameTime {
		t.Errorf("expected UpsertBlock() to return %s, got %s", upsertWithSetAndDeleteAtSameTime, result)
	}
}

func TestUpsertToDelete(t *testing.T) {
	result, err := dquely.UpsertDelete("v", dquely.Regexp("email", `.*@company1.io$`), "name", "email", "age")
	if err != nil {
		t.Fatal(err)
	}
	if result != upsertToDeleteMock {
		t.Errorf("expected UpsertDelete() to return %s, got %s", upsertToDeleteMock, result)
	}
}

func TestUpsertWithQuery(t *testing.T) {
	q := dquely.NewDQL("").Func(dquely.Eq("email", "user@company1.io")).Select("v as uid", "name")
	user := User{Name: "first last", Email: "user@company1.io"}
	result, err := dquely.UpsertWithQuery("q", q, "v", user, "name", "email")
	if err != nil {
		t.Fatal(err)
	}
	if result != upsertWithFuncMock {
		t.Errorf("expected UpsertWithQuery() to return %s, got %s", upsertWithFuncMock, result)
	}
}

func TestMutationUserFieldBuilder(t *testing.T) {
	user := &UserFieldBuilder{
		Name:  "Alice",
		Age:   29,
		Roles: map[string]int{"company": 2, "user": 1},
	}
	result, err := dquely.Mutation(user)
	if err != nil {
		t.Fatal(err)
	}
	if result != userFieldBuilderMutationMock {
		t.Errorf("expected Mutation() to return %s, got %s", userFieldBuilderMutationMock, result)
	}
}

func TestMutationUserLack(t *testing.T) {
	user := &UserLack{Name: "Alice", Age: 29, Email: "alice@example.com"}
	result, err := dquely.Mutation(user)
	if err != nil {
		t.Fatal(err)
	}
	if result != userLackMutationMock {
		t.Errorf("expected Mutation() to return %s, got %s", userLackMutationMock, result)
	}
}

func TestMutationWithDgraphType(t *testing.T) {
	user := &UserDgraph{Name: "Alice", Age: 29, Email: "alice@example.com"}
	result, err := dquely.Mutation(user)
	if err != nil {
		t.Fatal(err)
	}
	if result != userMutationMock {
		t.Errorf("expected Mutation() to return %s, got %s", userMutationMock, result)
	}
}

func TestMutationWithUniqueTag(t *testing.T) {
	user := &UserWithUnique{UserName: "alice", Email: "alice@example.com", Age: 29}
	result, err := dquely.Mutation(user)
	if err != nil {
		t.Fatal(err)
	}
	if result != userWithUniqueMutationFullMock {
		t.Errorf("expected Mutation() to return %s, got %s", userWithUniqueMutationFullMock, result)
	}
}

func TestUniqueFields(t *testing.T) {
	user := UserWithUnique{UserName: "alice", Email: "alice@example.com", Age: 29}
	fields, err := dquely.UniqueFields(user)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 unique fields, got %d", len(fields))
	}
	if fields[0].Predicate != "userName" || fields[0].Value != "alice" {
		t.Errorf("unexpected first unique field: %+v", fields[0])
	}
	if fields[1].Predicate != "email" || fields[1].Value != "alice@example.com" {
		t.Errorf("unexpected second unique field: %+v", fields[1])
	}
}

func TestUniqueFieldsPointer(t *testing.T) {
	user := &UserWithUnique{UserName: "bob", Email: "bob@example.com"}
	fields, err := dquely.UniqueFields(user)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 unique fields, got %d", len(fields))
	}
	if fields[0].Predicate != "userName" || fields[0].Value != "bob" {
		t.Errorf("unexpected first unique field: %+v", fields[0])
	}
	if fields[1].Predicate != "email" || fields[1].Value != "bob@example.com" {
		t.Errorf("unexpected second unique field: %+v", fields[1])
	}
}

func TestUniqueFieldsNone(t *testing.T) {
	user := User{Name: "Alice", Email: "alice@example.com"}
	fields, err := dquely.UniqueFields(user)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 0 {
		t.Errorf("expected 0 unique fields for struct with no unique tags, got %d", len(fields))
	}
}

func TestMutationNotPointer(t *testing.T) {
	user := User{Name: "Alice", Age: 29, Email: "alice@example.com"}
	_, err := dquely.Mutation(user)
	if err == nil {
		t.Error("expected error when passing a non-pointer to Mutation")
	}
}

func TestMutationNoUidField(t *testing.T) {
	node := &UserNoUID{Name: "Alice", Email: "alice@example.com"}
	_, err := dquely.Mutation(node)
	if err == nil {
		t.Error("expected error when struct has no dquely:\"uid\" field")
	}
}

func TestBlankNodeName(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{&User{}, "user"},
		{&UserWithUnique{}, "user"}, // DgraphType() returns "User"
		{&UserDgraph{}, "user"},     // DgraphType() returns "User"
	}
	for _, tc := range tests {
		got, err := dquely.BlankNodeName(tc.input)
		if err != nil {
			t.Fatalf("BlankNodeName(%T) unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("BlankNodeName(%T) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSetUID(t *testing.T) {
	user := &User{}
	if err := dquely.SetUID(user, "0x1234"); err != nil {
		t.Fatal(err)
	}
	if user.Uid != "0x1234" {
		t.Errorf("expected Uid to be 0x1234, got %s", user.Uid)
	}
}

func TestSetUIDNotPointer(t *testing.T) {
	user := User{}
	if err := dquely.SetUID(user, "0x1234"); err == nil {
		t.Error("expected error when passing a non-pointer to SetUID")
	}
}

func TestSetUIDNoField(t *testing.T) {
	node := &UserNoUID{Name: "Alice"}
	if err := dquely.SetUID(node, "0x1234"); err == nil {
		t.Error("expected error when struct has no dquely:\"uid\" field")
	}
}

func TestSimpleParseMutation(t *testing.T) {
	var user = UserLack{
		Uid:   "",
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   29,
	}
	query, muConds, err := dquely.ParseMutation(&user)
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	if query != "" {
		t.Errorf("expected ParseMutation() to get query be empty, got %s", query)
	}
	var cond = muConds[0]
	if cond.Cond != "" {
		t.Errorf("expected ParseMutation() to get Condition be empty, got %s", cond.Cond)
	}
	if string(cond.SetNquads) != userLackMutationMock {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", userLackMutationMock,
			string(cond.SetNquads))
	}
}

func TestSingleParseMutation(t *testing.T) {
	var user = UserWithUnique{
		Uid:      "",
		UserName: "alice",
		Email:    "alice@example.com",
		Age:      29,
	}
	query, muConds, err := dquely.ParseMutation(&user)
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	if query != userUniqueSingleQuery {
		t.Errorf("expected ParseMutation() to get query %s, got %s", userUniqueSingleQuery, query)
	}
	var cond = muConds[0]
	if cond.Cond != userUniqueCondMock {
		t.Errorf("expected ParseMutation() to get Condition %s, got %s", userUniqueCondMock, cond.Cond)
	}
	if string(cond.SetNquads) != userWithUniqueMutationMock {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", userWithUniqueMutationMock,
			string(cond.SetNquads))
	}
}

func TestSingleLackingParseMutation(t *testing.T) {
	var user = UserWithUnique{
		Uid:      "",
		UserName: "alice",
		Email:    "",
		Age:      29,
	}
	query, muConds, err := dquely.ParseMutation(&user)
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	if query != userUniqueLackingSingleQuery {
		t.Errorf("expected ParseMutation() to get query %s, got %s", userUniqueLackingSingleQuery, query)
	}
	var cond = muConds[0]
	if cond.Cond != userUniqueCondMock {
		t.Errorf("expected ParseMutation() to get Condition %s, got %s", userUniqueCondMock, cond.Cond)
	}
	if string(cond.SetNquads) != userWithUniqueLackingMutationMock {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", userWithUniqueLackingMutationMock,
			string(cond.SetNquads))
	}
}

func TestErrorParseMutation(t *testing.T) {
	var user = UserWithUnique{
		Uid:      "",
		UserName: "alice",
		Email:    "",
		Age:      29,
	}
	_, _, err := dquely.ParseMutation(user)
	if err == nil {
		t.Error("expected error when passing a non-pointer to SetUID")
	}
}

func TestErrorNoUidParseMutation(t *testing.T) {
	var user = UserNoUID{
		Name:  "alice",
		Email: "alice@example.com",
	}
	_, _, err := dquely.ParseMutation(&user)
	if err == nil {
		t.Error("expected error when passing a non-uid to SetUID")
	}
}

func TestUidSingleParseMutation(t *testing.T) {
	var user = UserWithUnique{
		Uid:      "0x1",
		UserName: "alice",
		Email:    "alice@example.com",
		Age:      29,
	}
	query, muConds, err := dquely.ParseMutation(&user)
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	if query != userUniqueSingleWithUidQuery {
		t.Errorf("expected ParseMutation() to get query %s, got %s", userUniqueSingleWithUidQuery, query)
	}
	var cond = muConds[0]
	if cond.Cond != userUniqueWithUidCondMock {
		t.Errorf("expected ParseMutation() to get Condition %s, got %s", userUniqueWithUidCondMock, cond.Cond)
	}
	if string(cond.SetNquads) != userUniqueWithUidSquads {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", userUniqueWithUidSquads,
			string(cond.SetNquads))
	}
}

func TestUidLackingSingleParseMutation(t *testing.T) {
	var user = UserWithUnique{
		Uid:      "0x1",
		UserName: "alice",
		Email:    "alice@example.com",
		Age:      0,
	}
	query, muConds, err := dquely.ParseMutation(&user)
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	if query != userUniqueSingleWithUidQuery {
		t.Errorf("expected ParseMutation() to get query %s, got %s", userUniqueSingleWithUidQuery, query)
	}
	var cond = muConds[0]
	if cond.Cond != userUniqueWithUidCondMock {
		t.Errorf("expected ParseMutation() to get Condition %s, got %s", userUniqueWithUidCondMock, cond.Cond)
	}
	if string(cond.SetNquads) != userUniqueLackingWithUidSquads {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", userUniqueLackingWithUidSquads,
			string(cond.SetNquads))
	}
	if string(cond.DelNquads) != `<0x1> <age> * .` {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", `<0x1> <age> * .`,
			string(cond.SetNquads))
	}
}

func TestUidLackingMultiParseMutation(t *testing.T) {
	var user = UserWithUnique{
		Uid:      "0x1",
		UserName: "alice",
		Email:    "",
		Age:      0,
	}
	query, muConds, err := dquely.ParseMutation(&user)
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	if query != userUniqueLackingSingleWithUidQuery {
		t.Errorf("expected ParseMutation() to get query %s, got %s", userUniqueLackingSingleWithUidQuery, query)
	}
	var cond = muConds[0]
	if cond.Cond != userUniqueWithUidCondMock {
		t.Errorf("expected ParseMutation() to get Condition %s, got %s", userUniqueWithUidCondMock, cond.Cond)
	}
	if string(cond.SetNquads) != `<0x1> <userName> "alice" .` {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", `<0x1> <userName> "alice" .`,
			string(cond.SetNquads))
	}
	if string(cond.DelNquads) != userUniqueDelMultiSquads {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", userUniqueDelMultiSquads,
			string(cond.SetNquads))
	}
}

func TestUndeepMutation(t *testing.T) {
	var company = Company{
		Name: "A",
		Owner: &ShortUser{
			Name: "U",
		},
	}
	query, muConds, err := dquely.ParseMutation(&company)
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	if query != "" {
		t.Fatalf("expected ParseMutation() to get query be empty, got %s", query)
	}
	var cond = muConds[0]
	if cond.Cond != "" {
		t.Fatalf("expected ParseMutation() to get Condition be empty, got %s", cond.Cond)
	}
	const expectedSet = `_:company <name> "A" .
_:company <dgraph.type> "Company" .`
	if string(cond.SetNquads) != expectedSet {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", expectedSet, string(cond.SetNquads))
	}
	if string(cond.DelNquads) != "" {
		t.Errorf("expected ParseMutation() to get Mutation be empty, got %s", string(cond.SetNquads))
	}
}

func TestDeepMutation(t *testing.T) {
	var company = Company{
		Name: "A",
		Owner: &ShortUser{
			Name: "U",
		},
	}
	query, muConds, err := dquely.ParseMutation(&company, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	if query != "" {
		t.Fatalf("expected ParseMutation() to get query be empty, got %s", query)
	}
	var cond = muConds[0]
	if cond.Cond != "" {
		t.Fatalf("expected ParseMutation() to get Condition be empty, got %s", cond.Cond)
	}
	const expectedSet = `_:company <name> "A" .
_:company <dgraph.type> "Company" .
_:company <owner> _:owner .
_:owner <name> "U" .
_:owner <dgraph.type> "ShortUser" .`
	if string(cond.SetNquads) != expectedSet {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", expectedSet, string(cond.SetNquads))
	}
	if string(cond.DelNquads) != "" {
		t.Errorf("expected ParseMutation() to get Mutation be empty, got %s", string(cond.SetNquads))
	}
}

func TestDeepMultiMutation(t *testing.T) {
	var company = Company{
		Name: "A",
		Owner: &ShortUser{
			Name: "U",
		},
		Staffs: []ShortUser{
			{Name: "S1"},
			{Name: "S2"},
		},
	}
	query, muConds, err := dquely.ParseMutation(&company, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(muConds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(muConds))
	}
	if query != "" {
		t.Fatalf("expected ParseMutation() to get query be empty, got %s", query)
	}
	var cond = muConds[0]
	if cond.Cond != "" {
		t.Fatalf("expected ParseMutation() to get Condition be empty, got %s", cond.Cond)
	}
	const expectedSet = `_:company <name> "A" .
_:company <dgraph.type> "Company" .
_:company <owner> _:owner .
_:company <staffs> _:staffs0 .
_:company <staffs> _:staffs1 .
_:owner <name> "U" .
_:owner <dgraph.type> "ShortUser" .
_:staffs0 <name> "S1" .
_:staffs0 <dgraph.type> "ShortUser" .
_:staffs1 <name> "S2" .
_:staffs1 <dgraph.type> "ShortUser" .`
	if string(cond.SetNquads) != expectedSet {
		t.Errorf("expected ParseMutation() to get Mutation %s, got %s", expectedSet, string(cond.SetNquads))
	}
	if string(cond.DelNquads) != "" {
		t.Errorf("expected ParseMutation() to get Mutation be empty, got %s", string(cond.SetNquads))
	}
}

package dquely_test

import (
	"testing"

	"github.com/vibros68/dquely"
)

func TestMutation(t *testing.T) {
	user := User{Name: "Alice", Age: 29, Email: "alice@example.com"}
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
	q := dquely.New().BlockVar("v").Has("age").Select("a as age")
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
	q := dquely.New().Func(dquely.Eq("email", "user@company1.io")).Select("v as uid", "name")
	user := User{Name: "first last", Email: "user@company1.io"}
	result, err := dquely.UpsertWithQuery("q", q, "v", user, "name", "email")
	if err != nil {
		t.Fatal(err)
	}
	if result != upsertWithFuncMock {
		t.Errorf("expected UpsertWithQuery() to return %s, got %s", upsertWithFuncMock, result)
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

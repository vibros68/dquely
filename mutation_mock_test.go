package dquely_test

const userMutationMock = `{
  set {
    _:user <name> "Alice" .
    _:user <email> "alice@example.com" .
    _:user <age> "29" .
    _:user <dgraph.type> "User" .
  }
}`

const userPartialMutationMock = `upsert {
  query {
    user as var(func: eq(email, "alice@example.com"))
  }
  mutation {
    set {
      uid(user) <age> "30" .
      uid(user) <name> "Alice Sayum" .
    }
  }
}`

const upsertWithFuncMock = `upsert {
  query {
    q(func: eq(email, "user@company1.io")) {
      v as uid
      name
    }
  }

  mutation {
    set {
      uid(v) <name> "first last" .
      uid(v) <email> "user@company1.io" .
    }
  }
}`

const upsertToDeleteMock = `upsert {
  query {
    v as var(func: regexp(email, /.*@company1.io$/))
  }

  mutation {
    delete {
      uid(v) <name> * .
      uid(v) <email> * .
      uid(v) <age> * .
    }
  }
}`

// we copy the values from the old predicate with set statement
// and we delete the old predicate with delete statement
const upsertWithSetAndDeleteAtSameTime = `upsert {
  query {
    v as var(func: has(age)) {
      a as age
    }
  }

  mutation {
    set {
      uid(v) <other> val(a) .
    }
    delete {
      uid(v) <age> * .
    }
  }
}`

type User struct {
	Uid   string `dquely:"uid"`
	Name  string `dquely:"name"`
	Age   int    `dquely:"age"`
	Email string `dquely:"email"`
}

type UserDgraph struct {
	Uid   string `dquely:"uid"`
	Name  string `dquely:"name"`
	Age   int    `dquely:"age"`
	Email string `dquely:"email"`
}

const userLackMutationMock = `{
  set {
    _:user <name> "Alice" .
    _:user <Email> "alice@example.com" .
    _:user <Age> "29" .
    _:user <dgraph.type> "User" .
  }
}`

type UserLack struct {
	Uid   string `dquely:"uid"`
	Name  string `dquely:"name"`
	Age   int
	Email string
}

func (u *UserLack) DgraphType() string {
	return "User"
}

func (u *UserDgraph) DgraphType() string {
	return "User"
}

type UserFieldBuilder struct {
	Uid   string `dquely:"uid"`
	Name  string `dquely:"name"`
	Age   int
	Roles map[string]int `dquely:"roles,json"`
}

func (u *UserFieldBuilder) DgraphType() string {
	return "User"
}

const userFieldBuilderMutationMock = `{
  set {
    _:user <name> "Alice" .
    _:user <roles> "{\"company\":2,\"user\":1}" .
    _:user <Age> "29" .
    _:user <dgraph.type> "User" .
  }
}`

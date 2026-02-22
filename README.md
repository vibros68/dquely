# dquely

A Go library for building [DGraph Query Language (DQL)](https://dgraph.io/docs/dql/) queries and mutations programmatically.

## Installation

```bash
go get github.com/vibros68/dquely
```

## Query builder

### Basic query

```go
q := dquely.New().
    Type("Person").
    Select("uid", "name", "friend")

fmt.Println(q.Query("people"))
```

```
{
  people(func: type(Person)) {
    uid
    name
    friend
  }
}
```

### Filters

Single filter:

```go
dquely.New().
    Type("Person").
    Filter(dquely.Gt("age", 18)).
    Select("uid", "name")
```

Multiple filters (AND):

```go
dquely.New().
    Has("description").
    Ngram("description", "brown fox").
    Filter(dquely.Eq("status", "published"), dquely.Gt("score", 10)).
    Select("uid", "description", "status", "score")
```

OR groups inside AND:

```go
dquely.New().
    Has("description").
    Ngram("description", "brown fox").
    Or(dquely.Eq("status", "published"), dquely.Eq("status", "review")).
    Select("uid", "description", "status")
```

NOT:

```go
dquely.New().
    Has("description").
    Ngram("description", "brown fox").
    Filter(dquely.Not(dquely.Eq("status", "archived"))).
    Select("uid", "description", "status")
```

### Available filter functions

| Function | DQL |
|---|---|
| `Eq(key, value)` | `eq(key, value)` |
| `Gt(key, value)` | `gt(key, value)` |
| `Ge(key, value)` | `ge(key, value)` |
| `Le(key, value)` | `le(key, value)` |
| `Lt(key, value)` | `lt(key, value)` |
| `Has(field)` | `has(field)` |
| `Uid(values...)` | `uid(v1, v2)` |
| `UidIn(pred, values...)` | `uid_in(pred, value)` |
| `Between(field, from, to)` | `between(field, from, to)` |
| `Regexp(field, pattern, flags...)` | `regexp(field, /pattern/flags)` |
| `AllOfTerms(key, value)` | `allofterms(key, value)` |
| `AnyOfTerms(key, value)` | `anyofterms(key, value)` |
| `AllOfText(key, value)` | `alloftext(key, value)` |
| `AnyOfText(key, value)` | `anyoftext(key, value)` |
| `Ngram(key, value)` | `ngram(key, value)` |
| `Not(expr)` | `NOT expr` |
| `Val(varName)` | `val(varName)` |
| `Count(predicate)` | `count(predicate)` |

### Nested selects

```go
dquely.New().
    AllOfTerms("name@en", "jones indiana").
    Select(
        "name@en",
        dquely.New().Select("name@en").As("genre"),
    ).
    Query("me")
```

```
{
  me(func: allofterms(name@en, "jones indiana")) {
    name@en
    genre {
      name@en
    }
  }
}
```

### Pagination

```go
dquely.New().
    AllOfTerms("name@en", "Hark Tsui").
    Select(
        "name@zh",
        dquely.New().
            Order("name@en", dquely.ASC).
            First(6).
            Offset(4).
            Select("name@en", "name@zh").
            As("director.film"),
    )
```

```
{
  me(func: allofterms(name@en, "Hark Tsui")) {
    name@zh
    director.film(orderasc: name@en, first: 6, offset: 4) {
      name@en
      name@zh
    }
  }
}
```

### Ordering

```go
Order(expr string, dir dquely.OrderDir)
```

```go
dquely.ASC   // orderasc
dquely.DESC  // orderdesc
```

### Variables

Assign a variable to a nested block:

```go
dquely.New().As("performance.film").Assign("F").Select("name@en")
// → F as performance.film { name@en }
```

Block-level variable (`var` block):

```go
dquely.NewVar().
    AllOfTerms("name@en", "Taraji Henson").
    Select(
        dquely.New().Select(
            dquely.New().Assign("F").Select("G as count(genre)").As("performance.film"),
        ).As("actor.film"),
    )
// → var(func: allofterms(...)) { actor.film { F as performance.film { G as count(genre) } } }
```

Block variable prefix with `BlockVar`:

```go
dquely.New().BlockVar("ID").
    AllOfTerms("name@en", "Steven").
    Has("director.film")
// → ID as var(func: allofterms(...)) @filter(has(director.film)) { ... }
```

### Multi-query

```go
q1 := dquely.New().Has("description").Ngram("description", "brown fox").As("me")
q2 := dquely.New().Regexp("name@en", "^Steven Sp.*$").As("directors")

fmt.Println(dquely.Build(q1, q2))
```

```
{
  me(func: has(description)) @filter(ngram(description, "brown fox")) {
    ...
  }

  directors(func: regexp(name@en, /^Steven Sp.*$/)) {
    ...
  }
}
```

### Condition blocks

```go
dquely.NewCondition("getJeunet", "q").
    Func(dquely.Eq("name@fr", "Jean-Pierre Jeunet"))
// → getJeunet as q(func: eq(name@fr, "Jean-Pierre Jeunet"))
```

### Directives

**`@cascade`** — removes nodes that don't have all queried predicates:

```go
dquely.New().AllOfTerms("name@en", "Harry Potter").Cascade().Select(...)
```

**`@groupby`** — groups results by a predicate:

```go
dquely.New().Select(
    dquely.New().GroupBy("genre").Select("a as count(uid)").As("director.film"),
)
```

### expand(_all_)

`dquely.ExpandAll` is a constant holding the `"expand(_all_)"` predicate supported by DGraph:

```go
q.Select(dquely.ExpandAll)
```

To create an inline `expand(_all_) { fields... }` block:

```go
dquely.ExpandAllBlock("u as uid")
// → expand(_all_) { u as uid }
```

---

## Mutations

### Full insert

`Mutation` serializes a struct to an RDF N-Quad `set` block. Fields are mapped by the `dquely` struct tag. Zero-value fields are skipped. String fields are emitted before numeric fields.

```go
type User struct {
    Uid   string `dquely:"uid"`
    Name  string `dquely:"name"`
    Age   int    `dquely:"age"`
    Email string `dquely:"email"`
}

user := User{Name: "Alice", Age: 29, Email: "alice@example.com"}
result, err := dquely.Mutation(user)
```

```
{
  set {
    _:user <name> "Alice" .
    _:user <email> "alice@example.com" .
    _:user <age> "29" .
    _:user <dgraph.type> "User" .
  }
}
```

**Custom type name** — implement `DgraphMutation` to override the blank node and `dgraph.type`:

```go
func (u *User) DgraphType() string { return "Member" }
// blank node becomes _:member, dgraph.type becomes "Member"
```

### Upsert (partial update)

`Upsert` queries a node using any `FilterExpr` and updates the specified fields from a struct, in the order given:

```go
user := User{Name: "Alice Sayum", Age: 30, Email: "alice@example.com"}
result, err := dquely.Upsert(user, dquely.Eq("email", "alice@example.com"), "age", "name")
```

```
upsert {
  query {
    user as var(func: eq(email, "alice@example.com"))
  }
  mutation {
    set {
      uid(user) <age> "30" .
      uid(user) <name> "Alice Sayum" .
    }
  }
}
```

### Upsert with named query block

`UpsertWithQuery` uses a full named query block with a body, referencing a variable assigned inside:

```go
q := dquely.New().Func(dquely.Eq("email", "user@company.io")).Select("v as uid", "name")
user := User{Name: "first last", Email: "user@company.io"}
result, err := dquely.UpsertWithQuery("q", q, "v", user, "name", "email")
```

```
upsert {
  query {
    q(func: eq(email, "user@company.io")) {
      v as uid
      name
    }
  }

  mutation {
    set {
      uid(v) <name> "first last" .
      uid(v) <email> "user@company.io" .
    }
  }
}
```

### Upsert delete

`UpsertDelete` deletes specific predicates (wildcard `*`) from matched nodes:

```go
result, err := dquely.UpsertDelete("v", dquely.Regexp("email", `.*@company.io$`), "name", "email", "age")
```

```
upsert {
  query {
    v as var(func: regexp(email, /.*@company.io$/))
  }

  mutation {
    delete {
      uid(v) <name> * .
      uid(v) <email> * .
      uid(v) <age> * .
    }
  }
}
```

### UpsertBlock — combined set and delete

`UpsertBlock` is the low-level primitive that handles any combination of set and delete triples. The query block is a `*DQuely`:

- If `BlockVar` is set on the query → rendered as `blockVar as var(func: ...)`
- Otherwise → rendered as `queryName(func: ...)`

**Triple constructors:**

| Constructor | Renders as |
|---|---|
| `TripleSet(varRef, pred, value)` | `uid(v) <pred> "value" .` |
| `TripleSetVal(varRef, pred, valVar)` | `uid(v) <pred> val(a) .` |
| `TripleDelete(varRef, pred)` | `uid(v) <pred> * .` |

**Example — copy a predicate value then delete the old one:**

```go
q := dquely.New().BlockVar("v").Has("age").Select("a as age")
result := dquely.UpsertBlock("var", q,
    dquely.TripleSetVal("v", "other", "a"),
    dquely.TripleDelete("v", "age"),
)
```

```
upsert {
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
}
```

`UpsertBlock` can also express the patterns covered by `UpsertWithQuery` and `UpsertDelete`:

```go
// equivalent to UpsertDelete
q := dquely.New().BlockVar("v").Regexp("email", `.*@company.io$`)
dquely.UpsertBlock("var", q,
    dquely.TripleDelete("v", "name"),
    dquely.TripleDelete("v", "email"),
)

// equivalent to UpsertWithQuery
q := dquely.New().Func(dquely.Eq("email", "user@company.io")).Select("v as uid", "name")
dquely.UpsertBlock("q", q,
    dquely.TripleSet("v", "name", "first last"),
    dquely.TripleSet("v", "email", "user@company.io"),
)
```

# dquely — Usage Guide

## Table of Contents

- [Struct Tags](#struct-tags)
- [Query Builder](#query-builder)
  - [Basic Query](#basic-query)
  - [Filters](#filters)
  - [Nested Selects](#nested-selects)
  - [Pagination & Ordering](#pagination--ordering)
  - [Variables](#variables)
  - [Multi-query](#multi-query)
  - [Directives](#directives)
- [Mutations](#mutations)
  - [Mutation](#mutation)
  - [ParseMutation](#parsemutation)
  - [Deep Mutation](#deep-mutation)
  - [Upsert](#upsert)
  - [UpsertWithQuery](#upsertwithquery)
  - [UpsertDelete](#upsertdelete)
  - [UpsertBlock](#upsertblock)
- [UID Helpers](#uid-helpers)
- [Client](#client)

---

## Struct Tags

Fields are mapped to DGraph predicates via the `dquely` struct tag.

```go
type User struct {
    Uid   string `dquely:"uid"`           // marks the UID field
    Name  string `dquely:"name"`          // predicate "name"
    Email string `dquely:"email,unique"`  // predicate "email", enforced unique
    Roles map[string]int `dquely:"roles,json"` // serialized as JSON string
}
```

| Option | Effect |
|--------|--------|
| `dquely:"uid"` | Marks the field as the UID receiver |
| `dquely:"predicate"` | Maps field to the given predicate name |
| `dquely:",unique"` | Flags the field for duplicate-prevention in `ParseMutation` |
| `dquely:",json"` | Serializes the field value as a JSON string |
| `dquely:"-"` | Skips the field entirely |

If no tag is provided, the Go field name is used as the predicate.

**Custom DGraph type** — implement `DgraphMutation` to override the blank-node name and `dgraph.type`:

```go
func (u *User) DgraphType() string { return "Member" }
// _:member ... <dgraph.type> "Member" .
```

---

## Query Builder

### Basic Query

```go
q := dquely.NewDQL("people").
    Type("Person").
    Select("uid", "name", "email")

fmt.Println(q.Query())
```

```
{
  people(func: type(Person)) {
    uid
    name
    email
  }
}
```

### Filters

> **Security note:** string *values* passed to filters and mutations are escaped
> automatically, so untrusted user input is safe as a value. Predicate/key names
> (e.g. the `"age"` in `Gt("age", 18)`) are interpolated verbatim and are **not**
> escaped — always supply trusted schema names for keys, never raw user input.

**Single filter:**

```go
dquely.NewDQL("").
    Type("Person").
    Filter(dquely.Gt("age", 18)).
    Select("uid", "name")
```

**Multiple filters (AND):**

```go
dquely.NewDQL("").
    Has("description").
    Ngram("description", "brown fox").
    Filter(dquely.Eq("status", "published"), dquely.Gt("score", 10)).
    Select("uid", "description", "status", "score")
```

**OR inside AND:**

```go
dquely.NewDQL("").
    Has("description").
    Or(dquely.Eq("status", "published"), dquely.Eq("status", "review")).
    Select("uid", "description", "status")
```

**NOT:**

```go
dquely.NewDQL("").
    Has("description").
    Filter(dquely.Not(dquely.Eq("status", "archived"))).
    Select("uid", "description")
```

**Available filter functions:**

| Function | DQL output |
|----------|-----------|
| `Eq(key, value)` | `eq(key, value)` |
| `Gt(key, value)` | `gt(key, value)` |
| `Ge(key, value)` | `ge(key, value)` |
| `Le(key, value)` | `le(key, value)` |
| `Lt(key, value)` | `lt(key, value)` |
| `Has(field)` | `has(field)` |
| `Uid(values...)` | `uid(v1, v2, ...)` |
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

### Nested Selects

Pass a `*DQuely` as an element to `Select` to create a nested block. Use `.As(name)` to set the predicate name for the block.

```go
dquely.NewDQL("").
    AllOfTerms("name@en", "jones indiana").
    Select(
        "name@en",
        dquely.NewDQL("").Select("name@en").As("genre"),
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

### Pagination & Ordering

```go
dquely.NewDQL("").
    AllOfTerms("name@en", "Hark Tsui").
    Select(
        "name@zh",
        dquely.NewDQL("").
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

| Method | DQL |
|--------|-----|
| `Order(expr, dquely.ASC)` | `orderasc: expr` |
| `Order(expr, dquely.DESC)` | `orderdesc: expr` |
| `First(n)` | `first: n` |
| `Offset(n)` | `offset: n` |

### Variables

**Assign a variable to a nested block:**

```go
dquely.NewDQL("").As("performance.film").Assign("F").Select("name@en")
// F as performance.film { name@en }
```

**Block-level var block:**

```go
dquely.NewVar().
    AllOfTerms("name@en", "Taraji Henson").
    Select(
        dquely.NewDQL("").Select(
            dquely.NewDQL("").Assign("F").Select("G as count(genre)").As("performance.film"),
        ).As("actor.film"),
    )
```

**BlockVar prefix:**

```go
dquely.NewDQL("").BlockVar("ID").
    AllOfTerms("name@en", "Steven").
    Has("director.film")
// ID as var(func: allofterms(...)) @filter(has(director.film))
```

### Multi-query

```go
q1 := dquely.NewDQL("").Has("description").Ngram("description", "brown fox").As("me")
q2 := dquely.NewDQL("").Regexp("name@en", `^Steven Sp.*$`).As("directors")

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

### Directives

**`@cascade`** — removes nodes missing any queried predicate:

```go
dquely.NewDQL("").AllOfTerms("name@en", "Harry Potter").Cascade().Select("uid", "name@en")
```

**`@groupby`** — groups results:

```go
dquely.NewDQL("").Select(
    dquely.NewDQL("").GroupBy("genre").Select("a as count(uid)").As("director.film"),
)
```

**`expand(_all_)`:**

```go
q.Select(dquely.ExpandAll)

// inline expand block:
dquely.ExpandAllBlock("u as uid")
// → expand(_all_) { u as uid }
```

### Scopes

`Scopes` applies one or more `BindFunc` (`func(*DQuely) *DQuely`) in sequence,
letting you factor reusable query fragments and compose them into a chain:

```go
onlyActive := func(q *dquely.DQuely) *dquely.DQuely { return q.Eq("active", true) }
recent := func(q *dquely.DQuely) *dquely.DQuely { return q.Order("createdAt", dquely.DESC) }

dquely.NewDQL("me").Type("Person").Scopes(onlyActive, recent).Select("uid", "name")
```

---

## Mutations

### Mutation

`Mutation` serializes a struct pointer to a DGraph `{ set { … } }` N-Quad block. String fields are emitted before numeric fields. Zero-value fields are skipped.

```go
type User struct {
    Uid   string `dquely:"uid"`
    Name  string `dquely:"name"`
    Age   int    `dquely:"age"`
    Email string `dquely:"email"`
}

user := &User{Name: "Alice", Age: 29, Email: "alice@example.com"}
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

> `input` must be a non-nil pointer with a `dquely:"uid"` field.

### ParseMutation

`ParseMutation` is the primary mutation builder. It returns a query string and a slice of `*api.Mutation` ready for use in a DGraph upsert request. Behaviour depends on the struct's unique fields and uid:

**No unique fields — plain insert:**

```go
type Post struct {
    Uid   string `dquely:"uid"`
    Title string `dquely:"title"`
    Body  string `dquely:"body"`
}

query, mutations, err := dquely.ParseMutation(&Post{Title: "Hello", Body: "World"})
// query == ""
// mutations[0].SetNquads == raw N-quads (no condition)
```

**Unique fields, uid empty — conditional insert:**

Builds a query that counts matching nodes; the mutation only runs when no duplicate exists (`@if(eq(len(v), 0))`).

```go
type User struct {
    Uid      string `dquely:"uid"`
    UserName string `dquely:"userName,unique"`
    Email    string `dquely:"email,unique"`
    Age      int    `dquely:"age"`
}

query, mutations, err := dquely.ParseMutation(&User{
    UserName: "alice",
    Email:    "alice@example.com",
    Age:      29,
})
```

```
// query:
{
  v as var(func: type(User))
    @filter(eq(userName, "alice") OR eq(email, "alice@example.com"))
}

// mutations[0].Cond:  @if(eq(len(v), 0))
// mutations[0].SetNquads:
_:user <userName> "alice" .
_:user <email> "alice@example.com" .
_:user <age> "29" .
_:user <dgraph.type> "User" .
```

**Unique fields, uid set — conditional update:**

Builds a two-variable query (`u` = current node, `v` = potential duplicates). Updates non-zero fields and deletes zero-value fields. Runs only when there is no duplicate and the node exists (`@if(eq(len(v), 0) AND eq(len(u), 1))`).

```go
query, mutations, err := dquely.ParseMutation(&User{
    Uid:      "0x1",
    UserName: "alice",
    Email:    "alice@example.com",
    Age:      29,
})
```

### Deep Mutation

Pass `true` as the second argument to recursively include nested struct fields.

```go
type ShortUser struct {
    Uid  string `dquely:"uid"`
    Name string `dquely:"name"`
}

type Company struct {
    Uid    string      `dquely:"uid"`
    Name   string      `dquely:"name"`
    Owner  *ShortUser  `dquely:"owner"`
    Staffs []ShortUser `dquely:"staffs"`
}

query, mutations, err := dquely.ParseMutation(&Company{
    Name:  "Acme",
    Owner: &ShortUser{Name: "Bob"},
    Staffs: []ShortUser{
        {Name: "Alice"},
        {Name: "Charlie"},
    },
}, true)
```

```
_:company <name> "Acme" .
_:company <owner> _:owner .
_:company <staffs> _:staffs0 .
_:company <staffs> _:staffs1 .
_:company <dgraph.type> "Company" .
_:owner <name> "Bob" .
_:owner <dgraph.type> "ShortUser" .
_:staffs0 <name> "Alice" .
_:staffs0 <dgraph.type> "ShortUser" .
_:staffs1 <name> "Charlie" .
_:staffs1 <dgraph.type> "ShortUser" .
```

**Existing nested nodes** (with uid) are referenced by `<uid>` and their content is not re-emitted:

```go
Company{
    Name:  "Acme",
    Owner: &ShortUser{Uid: "0x1"},  // already exists
}
// → _:company <owner> <0x1> .   (no ShortUser block)
```

**Deep mutation with unique fields** — when the root struct has unique fields, the deduplication query and condition are still generated:

```go
type Companies struct {
    Uid         string       `dquely:"uid"`
    Name        string       `dquely:"name"`
    Slug        string       `dquely:"slug,unique"`
    Memberships []Membership `dquely:"users"`
}

query, mutations, err := dquely.ParseMutation(&Companies{
    Name: "Acme",
    Slug: "acme",
    Memberships: []Membership{{IsOwner: true, User: &User{Uid: "0x1"}}},
}, true)
// query contains @filter(eq(slug, "acme"))
// mutations[0].Cond == "@if(eq(len(v), 0))"
// mutations[0].SetNquads contains the full deep N-quads
```

### Upsert

`Upsert` queries a node using any `FilterExpr` and updates only the specified fields:

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

### UpsertWithQuery

Uses a full named query block where a variable is assigned inside the body:

```go
q := dquely.NewDQL("").Func(dquely.Eq("email", "user@company.io")).Select("v as uid", "name")
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

### UpsertDelete

Deletes specific predicates (wildcard `*`) from all matched nodes:

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

### UpsertBlock

Low-level primitive for any combination of set and delete triples. The query block is a `*DQuely`:

- If `BlockVar` is set → rendered as `blockVar as var(func: ...)`
- Otherwise → rendered as `queryName(func: ...)`

**Triple constructors:**

| Constructor | Renders as |
|-------------|-----------|
| `TripleSet(varRef, pred, value)` | `uid(v) <pred> "value" .` |
| `TripleSetVal(varRef, pred, valVar)` | `uid(v) <pred> val(a) .` |
| `TripleDelete(varRef, pred)` | `uid(v) <pred> * .` |

**Example — migrate a predicate value:**

```go
q := dquely.NewDQL("").BlockVar("v").Has("age").Select("a as age")
result := dquely.UpsertBlock("var", q,
    dquely.TripleSetVal("v", "yearsOld", "a"),
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
      uid(v) <yearsOld> val(a) .
    }
    delete {
      uid(v) <age> * .
    }
  }
}
```

---

## UID Helpers

### BlankNodeName

Returns the blank-node key that `Mutation` / `ParseMutation` generates for a struct. This matches the key in `api.Response.Uids` after a successful mutation.

```go
name, err := dquely.BlankNodeName(&User{})  // → "user"
```

### SetUID

Writes a UID string into the `dquely:"uid"` field of a struct pointer:

```go
user := &User{}
err := dquely.SetUID(user, "0x1234")
// user.Uid == "0x1234"
```

### SetUIDs

Distributes multiple UIDs (e.g. from `api.Response.Uids` after a deep mutation) into a struct and its nested fields. Keys are matched by each field's `dquely` tag predicate (the part before any comma), falling back to the exact field name when the tag has no predicate:

- Root struct — any key not matching a nested field is applied to the root's uid
- Nested `*Struct` field — key = `predicate` (e.g. `dquely:"owner"` → `owner`)
- Nested `[]Struct` element at index j — key = `predicate + strconv.Itoa(j)` (e.g. `dquely:"staffs"` → `staffs0`, `staffs1`, …)

```go
company := &Company{
    Name:  "Acme",
    Owner: &ShortUser{Name: "Bob"},
    Staffs: []ShortUser{{Name: "Alice"}},
}
dquely.ParseMutation(company, true)

err := dquely.SetUIDs(company, map[string]string{
    "acme":    "0x10",   // → company.Uid
    "owner":   "0x11",   // → company.Owner.Uid
    "staffs0": "0x12",   // → company.Staffs[0].Uid
})
```

### UniqueFields

Returns all fields tagged with `,unique` as `[]UniqueField{Predicate, Value}`:

```go
fields, err := dquely.UniqueFields(&User{UserName: "alice", Email: "alice@example.com"})
// fields[0] == {Predicate: "userName", Value: "alice"}
// fields[1] == {Predicate: "email",    Value: "alice@example.com"}
```

---

## Client

### Connecting

```go
client, err := dquely.NewClient(dquely.Config{
    DNS:       "localhost:9080",
    Username:  "groot",
    Password:  "password",
    Namespace: 0,
})
defer client.Close()
```

For a local or development cluster without ACL enabled, use `NewClientInsecure` (username/password are not required):

```go
client, err := dquely.NewClientInsecure(dquely.Config{DNS: "localhost:9080"})
```

Set `Debug` to log the generated query and N-quads. By default debug output goes to
`fmt.Printf`; assign `Logf` to route it through your own logger:

```go
client.Debug = true
client.Logf = log.Printf // any func(format string, args ...any)
```

### Applying a Schema

```go
err := client.SetSchema(ctx, `
    name:  string @index(exact) .
    email: string @index(exact) .
    age:   int .
    type User {
        name
        email
        age
    }
`)
```

### Mutate

`Mutate` runs `ParseMutation`, executes the DGraph request, and writes the generated UIDs back into the struct via `SetUIDs`:

```go
user := &User{Name: "Alice", Age: 29, Email: "alice@example.com"}
err := client.Mutate(ctx, user)
// user.Uid is now set to the UID assigned by DGraph

// deep mutation:
err = client.Mutate(ctx, company, true)
// company.Uid, company.Owner.Uid, company.Staffs[0].Uid, … all populated
```

Returns the sentinel `ErrDuplicate` when the conditional insert is rejected (duplicate detected via unique fields). Test for it with `errors.Is(err, dquely.ErrDuplicate)`.

### Querying

`Model[T]` returns a typed query builder. `First` executes the query and returns the first matching node:

```go
user, err := dquely.Model[User](client).First(ctx,
    dquely.NewDQL("user").Func(dquely.Eq("email", "alice@example.com")),
)
// The block name passed to NewDQL ("user") is also used as the JSON key when
// decoding the response, so it must be non-empty.
```

`First` forces `first: 1` on the query (any `First`/`Offset` on the filter is
overridden) and returns `ErrNotFound` when nothing matches:

```go
if errors.Is(err, dquely.ErrNotFound) {
    // no matching node
}
```

`Find` returns all matching nodes; `FindAndCount` also returns the total count
(pair the filter with `QueryAndCount` semantics for pagination):

```go
users, err := dquely.Model[User](client).Find(ctx, filter)
users, total, err := dquely.Model[User](client).FindAndCount(ctx, filter)
```

### Update

`Update` builds a uid-based conditional mutation (via `ParseUpdate`) and applies it.
Pass predicate names to limit the update, or `dquely.FieldAll` for every non-zero field:

```go
err := client.Update(ctx, &User{Uid: "0x1", Name: "Bob"}, "name")
err = client.Update(ctx, user, dquely.FieldAll)
```

`Mutate` also handles updates: when the struct's `uid` field is set it addresses the
existing node by `<uid>` (no blank node), so it does not return `ErrDuplicate` on a
successful update.

### Raw query

For cases the builder does not cover, run raw DQL and get the JSON response bytes:

```go
jsonBytes, err := client.Query(ctx, `{ q(func: type(User)) { uid name } }`)
```

### Transactions

Group multiple mutations atomically with `DoTxn` (auto commit/rollback) or manage a
`Txn` directly:

```go
err := client.DoTxn(ctx, func(txn *dquely.Txn) error {
    if err := txn.Mutate(ctx, user); err != nil {
        return err
    }
    return txn.Update(ctx, company, "name")
})
```

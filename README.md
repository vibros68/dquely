# dquely

A Go library for building [DGraph Query Language (DQL)](https://docs.dgraph.io/dql) queries and mutations programmatically.

## Installation

```bash
go get github.com/vibros68/dquely
```

## Features

- **Query builder** — type-safe, composable DQL queries with filters, pagination, ordering, variables, nested selects, and multi-query support
- **Struct-based mutations** — serialize Go structs to RDF N-Quads for insert, upsert, and deep (nested) mutations
- **Unique-field deduplication** — tag fields with `dquely:",unique"` to generate conditional upsert queries automatically
- **Deep mutations** — recursively build N-Quad sets for nested structs and slices in a single atomic request
- **UID injection** — `SetUIDs` distributes DGraph response UIDs back into a struct and all its nested fields
- **DGraph client** — connection management (ACL or insecure), one-call `Mutate`/`Update`, typed queries via `Model[T]` (`First`/`Find`/`FindAndCount`), transactions (`DoTxn`), raw DQL, and a pluggable debug logger
- **Typed errors** — `ErrDuplicate` (unique-constraint conflict) and `ErrNotFound`, both `errors.Is`-comparable
- **Injection-safe** — string values in queries and mutations are escaped before being emitted

## Documentation

For full usage details — struct tags, all query builder methods, mutation functions, UID helpers, and client usage — see **[INSTRUCT.md](./INSTRUCT.md)**.

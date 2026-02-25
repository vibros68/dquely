# dquely

A Go library for building [DGraph Query Language (DQL)](https://dgraph.io/docs/dql/) queries and mutations programmatically.

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
- **DGraph client wrapper** — thin wrapper around `dgo` with connection management and one-call `Mutate`

## Documentation

For full usage details — struct tags, all query builder methods, mutation functions, UID helpers, and client usage — see **[INSTRUCT.md](./INSTRUCT.md)**.

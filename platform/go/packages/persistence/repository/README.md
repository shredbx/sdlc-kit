# repository

A storage-agnostic `Repository[M]` interface, a typed query tree, list and search options, and the errors a backend returns.

```bash
go test ./...    # this module's tests
```

## Overview

`repository` is the vocabulary that domain code uses to talk to storage without importing a database driver. It holds no storage code. The PostgreSQL implementation is the nested module `postgres/` in this folder, which has its own `go.mod`.

What it defines:

- **`Repository[M]`**: `Get`, `List`, `Create`, `Update` and `Delete` for a model type `M`. `List` returns the items and the total count, for pagination.
- **Errors**: `ErrNotFound`, `ErrConflict` (a duplicate key) and `ErrValidation`. A backend wraps its own errors into these so callers can test with `errors.Is`. `NewNotFoundError(entity, id)` builds one that names the entity and the id and still matches `ErrNotFound`.
- **Queries**: a `Query` is a tree of `And`, `Or` and single `Predicate`s, and an empty query matches everything. Predicates come from a typed column descriptor, `Field[T]` (`Eq`, `Neq`, `Gt`, `Gte`, `Lt`, `Lte`, `In`, `Like`, `IsNull`, `IsNotNull`), so comparing an `int64` field with a string does not compile; `ArrayField.Contains` makes an array-containment predicate for a Postgres `TEXT[]` column. `And` and `Or` take predicates; to nest one inside another, fill in the `Query` struct directly.
- **`ListOptions`**: a `Filter`, a `Sort` (built with `Asc` and `DescSort`), a `Limit` (0 means the backend's default), an `Offset`, and `OnlyDeleted`, which lists soft-deleted rows instead of live ones.
- **Optional abilities**, which a backend may or may not have, so a caller type-asserts and does without: `GroupCounter` (a count per value of one field), `RangeCounter` (a count per `RangeBucket`, in one query) and `Searcher[M]` (ranked full-text search with `SearchOptions`; the result says which tier matched: prefix, fuzzy or ILIKE substring).

A predicate is data, not SQL: a logical field name, an operator and a value. Turning it into SQL, and logical names into columns, is the backend's job. The counters carry one contract: the field names they are given end up as column names in the statement, so they must be constants or checked against an allow-list, never request input.

## Install

`repository` is a Go module in the `platform/go` workspace and is not published. Inside the workspace it is listed in `platform/go/go.work`, so there is nothing to install: import it.

From another module, require it and point the `replace` at this folder (the path is relative to your `go.mod`):

```
require github.com/shredbx/sbx-core/pkg/repository v0.0.0

replace github.com/shredbx/sbx-core/pkg/repository => <path to platform/go/packages/persistence/repository>
```

The import path keeps its original name, `github.com/shredbx/sbx-core/pkg/repository`, until the naming pass; only the folder was regrouped.

## Usage

Build a filtered, sorted list request, and test an error the way domain code does. No backend is needed, because the request is only data:

```go
package main

import (
	"errors"
	"fmt"

	"github.com/shredbx/sbx-core/pkg/repository"
)

// Fields names a model's columns once, with their Go types.
var Fields = struct {
	Price     repository.Field[int64]
	Published repository.Field[bool]
	Tags      repository.ArrayField
}{
	Price:     repository.Field[int64]{Name: "price_amount"},
	Published: repository.Field[bool]{Name: "is_published"},
	Tags:      repository.ArrayField{Name: "tags"},
}

func main() {
	opts := repository.ListOptions{
		Filter: repository.And(
			Fields.Published.Eq(true),
			Fields.Price.Gte(500000),
			Fields.Tags.Contains([]string{"pool", "sea-view"}),
		),
		Sort:  []repository.SortField{repository.DescSort("created_at")},
		Limit: 20,
	}

	// A predicate is data: a logical field name, an operator and a value.
	// Turning it into SQL is the backend's job.
	for _, q := range opts.Filter.And {
		fmt.Println(q.Pred.FieldName, q.Pred.Value)
	}
	fmt.Println("second is >=:", opts.Filter.And[1].Pred.Op == repository.OpGte)
	fmt.Println("empty filter matches everything:", repository.ListOptions{}.Filter.IsEmpty())

	// Backends return these; callers test with errors.Is.
	err := repository.NewNotFoundError("property", "p-42")
	fmt.Println(err, "| is ErrNotFound:", errors.Is(err, repository.ErrNotFound))
}
```

It prints:

```
is_published true
price_amount 500000
tags [pool sea-view]
second is >=: true
empty filter matches everything: true
property with id "p-42" not found | is ErrNotFound: true
```

## Configuration

None. `repository` has no settings and no dependencies.

## Tests

`go test ./...` in this folder, or `go -C platform/go/packages/persistence/repository test ./...` from the repository root. The nested `postgres/` module is separate and is tested from its own folder.

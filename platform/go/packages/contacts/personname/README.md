# personname

The `PersonName` value type: a title, a given name, an optional middle name and a surname, with validation and two display forms.

```bash
go test ./...    # this module's tests
```

## Overview

`personname` keeps a person's name as four fields, `Title`, `GivenName`, `MiddleName` and `Surname`, rather than one string, so the caller chooses how to show it. Only the given name and the surname are required: `Validate` refuses a value that is empty or only whitespace and names the field (`given_name must not be empty`, `surname must not be empty`). The title and the middle name are not checked.

There are two display forms. `FullName` is the given name, the middle name (left out when empty) and the surname, and it never includes the title. `FormalName` is the title and the surname (`Mr. Jaidee`), or just the surname when there is no title. `IsZero` is true when all four fields are empty, which tells "no name given" apart from "an invalid name".

The type is a plain struct with `json` and `yaml` tags (`title`, `given_name`, `middle_name`, `surname`; the title and the middle name are left out when empty). Every method has a value receiver and none changes the name.

## Install

`personname` is a Go module in the `platform/go` workspace and is not published. Inside the workspace it is listed in `platform/go/go.work`, so there is nothing to install: import it.

From another module, require it and point the `replace` at this folder (the path is relative to your `go.mod`):

```
require github.com/shredbx/sbx-core/pkg/personname v0.0.0

replace github.com/shredbx/sbx-core/pkg/personname => <path to platform/go/packages/contacts/personname>
```

The import path keeps its original name, `github.com/shredbx/sbx-core/pkg/personname`, until the naming pass; only the folder was regrouped.

## Usage

Build a name, show it both ways, and validate it:

```go
package main

import (
	"fmt"

	"github.com/shredbx/sbx-core/pkg/personname"
)

func main() {
	name := personname.PersonName{
		Title:      "Dr.",
		GivenName:  "Anna",
		MiddleName: "Marie",
		Surname:    "Mueller",
	}
	fmt.Println("full:", name.FullName())
	fmt.Println("formal:", name.FormalName())

	// The given name and the surname are required; a blank one counts as empty.
	bad := personname.NewPersonName("Anna", "  ")
	fmt.Println("invalid:", bad.Validate())

	fmt.Println("unset:", personname.PersonName{}.IsZero(), "| set:", name.IsZero())
}
```

It prints:

```
full: Anna Marie Mueller
formal: Dr. Mueller
invalid: surname must not be empty
unset: true | set: false
```

## Configuration

None. `personname` is a pure value type with no settings and no dependencies.

## Tests

`go test ./...` in this folder, or `go -C platform/go/packages/contacts/personname test ./...` from the repository root.

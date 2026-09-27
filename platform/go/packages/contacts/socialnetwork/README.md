# socialnetwork

The `SocialNetwork` value type and its `List`: a reference to a social-media profile, and a list of them that is stored as one nullable JSON column.

```bash
go test ./...    # this module's tests
```

## Overview

`SocialNetwork` is a reference to a profile on a social platform: a `Platform` (`instagram`, `line`, …), a `Handle` and an optional `URL`. `Validate` requires a platform and a handle that are not blank. The platform is free text: nothing checks it against a list. `HasURL` reports whether the URL is set, and `IsZero` whether all three fields are empty.

`List` is a `[]SocialNetwork` that round-trips through one nullable JSON column. `Value` returns SQL `NULL` for an empty list. `Scan` accepts `[]byte` or `string` and turns `NULL`, an empty value or the JSON `null` into a nil list rather than an error; any other source type is an error. `List.Validate` checks every entry and reports the first failure with its position (`social network entry index 1: handle must not be empty`); an empty list is valid. The package comment calls it the one list wrapper for social profiles, to be reused rather than redefined per entity.

The single `SocialNetwork` has no `Value` or `Scan`. The package comment says a consuming entity gets three columns (`{attr}_platform`, `{attr}_handle` and `{attr}_url`); nothing in this module creates them, so that mapping belongs to the consumer. To store a set of profiles, use `List`.

## Install

`socialnetwork` is a Go module in the `platform/go` workspace and is not published. Inside the workspace it is listed in `platform/go/go.work`, so there is nothing to install: import it.

From another module, require it and point the `replace` at this folder (the path is relative to your `go.mod`):

```
require github.com/shredbx/sbx-core/pkg/socialnetwork v0.0.0

replace github.com/shredbx/sbx-core/pkg/socialnetwork => <path to platform/go/packages/contacts/socialnetwork>
```

The import path keeps its original name, `github.com/shredbx/sbx-core/pkg/socialnetwork`, until the naming pass; only the folder was regrouped.

## Usage

Validate a list of profiles, store it as one JSON value, and read it back:

```go
package main

import (
	"fmt"

	"github.com/shredbx/sbx-core/pkg/socialnetwork"
)

func main() {
	profiles := socialnetwork.List{
		socialnetwork.NewSocialNetwork("line", "@example_villas"),
		{Platform: "instagram", Handle: "example_villas", URL: "https://instagram.com/example_villas"},
	}
	fmt.Println("valid:", profiles.Validate() == nil)

	// The list is stored as one JSON value.
	stored, _ := profiles.Value()
	fmt.Println("stored:", string(stored.([]byte)))

	var back socialnetwork.List
	_ = back.Scan(stored)
	fmt.Println("round trip:", len(back), "profiles, second has a URL:", back[1].HasURL())

	// An empty list is stored as SQL NULL.
	none, _ := socialnetwork.List{}.Value()
	fmt.Println("empty stores:", none)

	// A bad entry is reported with its position.
	bad := socialnetwork.List{profiles[0], {Platform: "tiktok"}}
	fmt.Println("invalid:", bad.Validate())
}
```

It prints:

```
valid: true
stored: [{"platform":"line","handle":"@example_villas"},{"platform":"instagram","handle":"example_villas","url":"https://instagram.com/example_villas"}]
round trip: 2 profiles, second has a URL: true
empty stores: <nil>
invalid: social network entry index 1: handle must not be empty
```

## Configuration

None. `socialnetwork` is a pure value type with no settings and no third-party dependencies.

## Tests

`go test ./...` in this folder, or `go -C platform/go/packages/contacts/socialnetwork test ./...` from the repository root.

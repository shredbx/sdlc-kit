# phonenumber

The `PhoneNumber` value type: an international phone number kept as a country code, a number and an optional type.

```bash
go test ./...    # this module's tests
```

## Overview

`phonenumber` keeps an international phone number in three parts: `CountryCode` (with its leading `+`), `Number` and an optional `PhoneType` such as `mobile`, so a caller never has to parse one free-form string. `Validate` stops at the first of three failures: a blank country code, a country code that does not start with `+`, or a blank number. It does not check that the country code exists or that the number has the right length or digits, and `PhoneType` is free text.

`Format` returns the international form: the country code and the number, separated by one space (`+66 812345678`). `IsZero` is true when all three fields are empty.

The type is a plain struct with `json` and `yaml` tags (`country_code`, `number`, `phone_type`; the type is left out when empty). It has no `driver.Valuer` or `sql.Scanner`, so it is not stored as one value. The package comment says a consuming entity gets three columns (`{attr}_country_code`, `{attr}_number` and `{attr}_type`); nothing in this module creates them, so that mapping belongs to the consumer.

## Install

`phonenumber` is a Go module in the `platform/go` workspace and is not published. Inside the workspace it is listed in `platform/go/go.work`, so there is nothing to install: import it.

From another module, require it and point the `replace` at this folder (the path is relative to your `go.mod`):

```
require github.com/shredbx/sbx-core/pkg/phonenumber v0.0.0

replace github.com/shredbx/sbx-core/pkg/phonenumber => <path to platform/go/packages/contacts/phonenumber>
```

The import path keeps its original name, `github.com/shredbx/sbx-core/pkg/phonenumber`, until the naming pass; only the folder was regrouped.

## Usage

Build a number, format it, and validate it:

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/shredbx/sbx-core/pkg/phonenumber"
)

func main() {
	phone := phonenumber.NewPhoneNumber("+66", "812345678")
	phone.PhoneType = "mobile"

	fmt.Println("valid:", phone.Validate() == nil)
	fmt.Println("format:", phone.Format())

	// The country code must start with "+"; nothing else about it is checked.
	bad := phonenumber.NewPhoneNumber("66", "812345678")
	fmt.Println("no plus:", bad.Validate())

	// Three JSON fields, not one string.
	raw, _ := json.Marshal(phone)
	fmt.Println("json:", string(raw))
}
```

It prints:

```
valid: true
format: +66 812345678
no plus: country_code must start with '+', got "66"
json: {"country_code":"+66","number":"812345678","phone_type":"mobile"}
```

## Configuration

None. `phonenumber` is a pure value type with no settings and no dependencies.

## Tests

`go test ./...` in this folder, or `go -C platform/go/packages/contacts/phonenumber test ./...` from the repository root.

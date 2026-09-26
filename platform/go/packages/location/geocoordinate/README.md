# geocoordinate

A WGS84 latitude/longitude value object with bounds validation and great-circle distance.

```bash
go test ./...    # this module's tests
```

## Overview

`geocoordinate` is a small, immutable value object for a GPS position in decimal degrees (WGS84). `Validate` checks the bounds (latitude in [-90, 90], longitude in [-180, 180]), `IsZero` tells an unset position from a real one, and `DistanceTo` returns the Haversine distance in kilometres, using a mean Earth radius of 6371 km.

It is meant to be stored as two typed columns per coordinate (`<attr>_latitude` and `<attr>_longitude`), the same way a `money` amount is stored as two columns, and it reads and writes JSON and YAML as `latitude` and `longitude`.

## Install

`geocoordinate` is a Go module in the `platform/go` workspace and is not published. Inside the workspace it is listed in `platform/go/go.work`, so there is nothing to install: import it.

From another module, require it and point the `replace` at this folder (the path is relative to your `go.mod`):

```
require github.com/shredbx/sbx-core/pkg/geocoordinate v0.0.0

replace github.com/shredbx/sbx-core/pkg/geocoordinate => <path to platform/go/packages/location/geocoordinate>
```

The import path keeps its original name, `github.com/shredbx/sbx-core/pkg/geocoordinate`, until the naming pass; only the folder was regrouped.

## Usage

Create two positions, validate one, and measure the distance:

```go
package main

import (
	"fmt"

	"github.com/shredbx/sbx-core/pkg/geocoordinate"
)

func main() {
	bangkok := geocoordinate.NewGeoCoordinate(13.7563, 100.5018)
	chiangMai := geocoordinate.NewGeoCoordinate(18.7883, 98.9853)

	fmt.Println("valid:", bangkok.Validate() == nil)
	fmt.Printf("distance: %.0f km\n", bangkok.DistanceTo(chiangMai))

	bad := geocoordinate.NewGeoCoordinate(91, 0)
	fmt.Println("out of range:", bad.Validate())
}
```

It prints:

```
valid: true
distance: 582 km
out of range: latitude must be between -90 and 90, got 91.000000
```

## Configuration

None. `geocoordinate` is a pure value type with no settings and no dependencies.

## Tests

`go test ./...` in this folder, or `go -C platform/go/packages/location/geocoordinate test ./...` from the repository root.

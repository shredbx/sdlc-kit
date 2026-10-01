# video

A read-only, keyless video-link engine: recognize a pasted YouTube, TikTok or Instagram URL, parse it, and resolve its display metadata.

```bash
go test ./...    # this module's tests
```

## Overview

Given a pasted social-video URL, `video` recognizes the platform, parses a canonical URL, a stable external id and an orientation (all without network I/O), and resolves a title, an author and a thumbnail from the platform's keyless oEmbed endpoint. It never hosts or transcodes media; it links out to the original. It uses only the Go standard library.

Everything goes through a `Registry`, built with `NewRegistry(client, cache)`. It holds one resolver per platform behind the `Resolver` interface (`Match`, `Parse`, `Resolve`) and asks each in turn; a URL that no resolver recognizes gives an error wrapping `ErrUnsupportedURL`.

| Platform | Recognized | Orientation | `Resolve` |
|---|---|---|---|
| YouTube | `youtube.com`, `m.youtube.com` and `youtu.be`: watch, short link, `/shorts/` and `/embed/` | portrait for `/shorts/`, otherwise landscape | YouTube oEmbed |
| TikTok | any `tiktok.com` host; the canonical form is `/@user/video/<id>` | portrait | TikTok oEmbed |
| Instagram | `instagram.com` `/reel/` and `/reels/` (portrait), `/p/` (landscape) | as listed | an empty `Meta` and no error: Instagram has no keyless metadata |

Three details:

- A YouTube Short keeps its `/shorts/` form as the canonical URL, and every other YouTube link becomes `https://www.youtube.com/watch?v=<id>`; the query string and the timestamp are dropped.
- TikTok `vm.` and `vt.` short links are recognized but cannot be parsed, because no redirect is followed. Expand them to the canonical URL first.
- `PlatformFacebook` is a valid value with no resolver, so a Facebook URL is `ErrUnsupportedURL`.

A resolved `Meta` is cached for 24 hours, keyed by platform and external id, in a `Cache` (`Get` and `Set` with a TTL). `NewMemoryCache` is an in-memory one, and a nil cache means that; a production consumer passes its own. A failed lookup is never cached, so it can be retried.

`Platform`, `Orientation`, `Placement` and `Status` are typed values instead of raw strings, each with `Parse…`, `Valid` and `String`. Their comments cite dictionary files; in this module the value sets are the Go constants. `Placement` and `Status` are for the consumer's own storage; the `Registry` does not use them. `ValidatePlacement(orientation, placement)` enforces one rule: a landscape video cannot go in the `shorts_rail`. Portrait fits `shorts_rail` and `tours_grid`, and landscape fits `tours_grid`.

## Install

`video` is a Go module in the `platform/go` workspace and is not published. Inside the workspace it is listed in `platform/go/go.work`, so there is nothing to install: import it.

From another module, require it and point the `replace` at this folder (the path is relative to your `go.mod`):

```
require github.com/shredbx/sbx-core/pkg/video v0.0.0

replace github.com/shredbx/sbx-core/pkg/video => <path to platform/go/packages/media/video>
```

The import path keeps its original name, `github.com/shredbx/sbx-core/pkg/video`, until the naming pass; only the folder was regrouped.

## Usage

Parse two links, resolve one twice, and check a placement. The registry is given a fake oEmbed client, so nothing leaves the machine:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/shredbx/sbx-core/pkg/video"
)

// fakeOEmbed answers every request with one canned oEmbed body and counts the
// calls, so this example never touches the network.
type fakeOEmbed struct{ calls int }

func (f *fakeOEmbed) Do(*http.Request) (*http.Response, error) {
	f.calls++
	body := `{"title":"Harbour walk-through","author_name":"Example Tours","thumbnail_url":"https://img.example.com/hq.jpg"}`
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
}

func main() {
	api := &fakeOEmbed{}
	reg := video.NewRegistry(api, nil) // a nil cache means an in-memory one

	// Parsing needs no network: host, id, canonical URL and orientation.
	watch, _ := reg.Parse("https://youtu.be/abc123XYZ_-?t=42")
	fmt.Println(watch.Platform, watch.ExternalID, watch.Orientation, watch.CanonicalURL)
	short, _ := reg.Parse("https://www.youtube.com/shorts/abc123XYZ_-")
	fmt.Println(short.Orientation, short.CanonicalURL)

	// Resolving fetches the title, author and thumbnail; the second call is cached.
	ctx := context.Background()
	meta, _ := reg.Resolve(ctx, watch)
	_, _ = reg.Resolve(ctx, watch)
	fmt.Println(meta.Title, "by", meta.AuthorName)
	fmt.Println("upstream calls after two resolves:", api.calls)

	_, err := reg.Parse("https://example.com/watch?v=1")
	fmt.Println("unknown host:", errors.Is(err, video.ErrUnsupportedURL))

	// A landscape video does not belong in the vertical shorts rail.
	fmt.Println(video.ValidatePlacement(video.OrientationLandscape, video.PlacementShortsRail))
}
```

It prints:

```
youtube abc123XYZ_- landscape https://www.youtube.com/watch?v=abc123XYZ_-
portrait https://www.youtube.com/shorts/abc123XYZ_-
Harbour walk-through by Example Tours
upstream calls after two resolves: 1
unknown host: true
video: landscape video cannot be placed in shorts_rail
```

## Configuration

`DefaultHTTPClient()` is an `*http.Client` with a 10 second timeout (`DefaultResolveTimeout`), and a nil client in `NewRegistry` means that one. The only outbound requests are GETs to `https://www.youtube.com/oembed` and `https://www.tiktok.com/oembed`, sent with a browser-like `User-Agent` and a 1 MB cap on the response. To replace the client, pass any value with a `Do(*http.Request) (*http.Response, error)` method: a stub in tests, or a client with your own transport. The cache lifetime and the `User-Agent` are fixed.

## Tests

`go test ./...` in this folder, or `go -C platform/go/packages/media/video test ./...` from the repository root. The tests use stub clients and the oEmbed samples in `testdata/`, so they need no network.

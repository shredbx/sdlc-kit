module github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go

go 1.26

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-chi/cors v1.2.1
	github.com/shredbx/sbx-core/pkg/auth v0.0.0
	github.com/shredbx/sbx-core/pkg/httputil v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.1 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/redis/go-redis/v9 v9.18.0 // indirect
	github.com/shredbx/sbx-core/pkg/money v0.0.0 // indirect
	github.com/shredbx/sbx-core/pkg/repository v0.0.0 // indirect
	github.com/shredbx/sbx-core/pkg/user v0.0.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)

// replace does not propagate: the whole in-repo closure of auth and httputil is listed here (D24).
replace (
	github.com/shredbx/sbx-core/pkg/auth => ../../packages/identity/auth
	github.com/shredbx/sbx-core/pkg/httputil => ../../packages/http/httputil
	github.com/shredbx/sbx-core/pkg/money => ../../packages/money
	github.com/shredbx/sbx-core/pkg/repository => ../../packages/persistence/repository
	github.com/shredbx/sbx-core/pkg/user => ../../packages/identity/user
)

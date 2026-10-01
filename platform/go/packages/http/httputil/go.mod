module github.com/shredbx/sbx-core/pkg/httputil

go 1.26

require (
	github.com/go-playground/validator/v10 v10.30.2
	github.com/google/uuid v1.6.0
	github.com/shredbx/sbx-core/pkg/money v0.0.0
)

require (
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)

replace github.com/shredbx/sbx-core/pkg/money => ../../money

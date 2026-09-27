module github.com/shredbx/sbx-core/pkg/scheduler/schedcli

go 1.26

require (
	github.com/shredbx/sbx-core/pkg/database v0.0.0
	github.com/shredbx/sbx-core/pkg/scheduler v0.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.1 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/shredbx/sbx-core/pkg/feed v0.0.0 // indirect
	github.com/shredbx/sbx-core/pkg/rss v0.0.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/shredbx/sbx-core/pkg/database => ../../../persistence/database
	github.com/shredbx/sbx-core/pkg/feed => ../../../news/feed
	github.com/shredbx/sbx-core/pkg/rss => ../../../news/rss
	github.com/shredbx/sbx-core/pkg/scheduler => ../
)

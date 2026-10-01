module github.com/shredbx/sbx-core/pkg/faq

go 1.26

require (
	github.com/shredbx/sbx-core/pkg/repository v0.0.0
	github.com/shredbx/sbx-core/pkg/repository/postgres v0.0.0
	github.com/shredbx/sbx-core/pkg/seo v0.0.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.1 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/shredbx/sbx-core/pkg/database v0.0.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)

replace (
	github.com/shredbx/sbx-core/pkg/database => ../persistence/database
	github.com/shredbx/sbx-core/pkg/repository => ../persistence/repository
	github.com/shredbx/sbx-core/pkg/repository/postgres => ../persistence/repository/postgres
	github.com/shredbx/sbx-core/pkg/seo => ../seo
)

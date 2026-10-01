module github.com/shredbx/sbx-core/pkg/property

go 1.26

require (
	github.com/google/uuid v1.6.0
	github.com/gosimple/slug v1.15.0
	github.com/jackc/pgx/v5 v5.9.1
	github.com/shredbx/sbx-core/pkg/address v0.0.0
	github.com/shredbx/sbx-core/pkg/money v0.0.0
	github.com/shredbx/sbx-core/pkg/repository v0.0.0
	github.com/shredbx/sbx-core/pkg/repository/postgres v0.0.0
	github.com/shredbx/sbx-core/pkg/seo v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/shredbx/sbx-core/pkg/database v0.0.0 // indirect
	github.com/shredbx/sbx-core/pkg/geocoordinate v0.0.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)

replace (
	github.com/shredbx/sbx-core/pkg/address => ../../location/address
	github.com/shredbx/sbx-core/pkg/database => ../../persistence/database
	github.com/shredbx/sbx-core/pkg/geocoordinate => ../../location/geocoordinate
	github.com/shredbx/sbx-core/pkg/money => ../../money
	github.com/shredbx/sbx-core/pkg/repository => ../../persistence/repository
	github.com/shredbx/sbx-core/pkg/repository/postgres => ../../persistence/repository/postgres
	github.com/shredbx/sbx-core/pkg/seo => ../../seo
)

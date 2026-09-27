module github.com/shredbx/sbx-core/pkg/contact/vcard

go 1.26

require (
	github.com/shredbx/sbx-core/pkg/contact v0.0.0
	github.com/shredbx/sbx-core/pkg/socialnetwork v0.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.1 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/shredbx/sbx-core/pkg/address v0.0.0 // indirect
	github.com/shredbx/sbx-core/pkg/database v0.0.0 // indirect
	github.com/shredbx/sbx-core/pkg/geocoordinate v0.0.0 // indirect
	github.com/shredbx/sbx-core/pkg/personname v0.0.0 // indirect
	github.com/shredbx/sbx-core/pkg/phonenumber v0.0.0 // indirect
	github.com/shredbx/sbx-core/pkg/repository v0.0.0 // indirect
	github.com/shredbx/sbx-core/pkg/repository/postgres v0.0.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)

replace (
	github.com/shredbx/sbx-core/pkg/address => ../../../location/address
	github.com/shredbx/sbx-core/pkg/contact => ../
	github.com/shredbx/sbx-core/pkg/database => ../../../persistence/database
	github.com/shredbx/sbx-core/pkg/geocoordinate => ../../../location/geocoordinate
	github.com/shredbx/sbx-core/pkg/personname => ../../personname
	github.com/shredbx/sbx-core/pkg/phonenumber => ../../phonenumber
	github.com/shredbx/sbx-core/pkg/repository => ../../../persistence/repository
	github.com/shredbx/sbx-core/pkg/repository/postgres => ../../../persistence/repository/postgres
	github.com/shredbx/sbx-core/pkg/socialnetwork => ../../socialnetwork
)

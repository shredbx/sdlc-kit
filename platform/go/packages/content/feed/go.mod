module github.com/shredbx/sbx-core/pkg/feed

go 1.26

require (
	github.com/google/uuid v1.6.0
	github.com/shredbx/sbx-core/pkg/rss v0.0.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace github.com/shredbx/sbx-core/pkg/rss => ../rss

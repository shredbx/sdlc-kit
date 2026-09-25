package postgres

import "testing"

func TestBuildPrefixTsquery(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty input", "", ""},
		{"whitespace only", "   ", ""},
		{"single token", "Agricul", "Agricul:*"},
		{"two tokens", "agricul land", "agricul:* & land:*"},
		{"three tokens with multiple spaces", "luxury  pool   villa", "luxury:* & pool:* & villa:*"},
		{"strips ampersand", "foo&bar", "foobar:*"},
		{"strips pipe and bang", "foo|bar!baz", "foobarbaz:*"},
		{"strips parens and colon", "foo(bar):baz", "foobarbaz:*"},
		{"strips quotes and backslash", `foo"bar'\\baz`, "foobarbaz:*"},
		{"operator-only token dropped", "valid & word", "valid:* & word:*"},
		{"all-operator input returns empty", "&|():", ""},
		{"mixed case preserved", "TEST Property", "TEST:* & Property:*"},
		{"unicode preserved", "เกษตร phuket", "เกษตร:* & phuket:*"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPrefixTsquery(tt.input)
			if got != tt.want {
				t.Errorf("buildPrefixTsquery(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

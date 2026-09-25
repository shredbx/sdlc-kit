package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shredbx/sbx-core/pkg/auth"
)

// TC-M1: schema validator rejects SQL-injection payloads.
func TestMustValidateSchema(t *testing.T) {
	valid := []string{
		"bestierealestate",
		"my_schema",
		"_underscore_first",
		"a",
		"schema123",
	}
	for _, s := range valid {
		t.Run("valid:"+s, func(t *testing.T) {
			assert.NotPanics(t, func() { auth.MustValidateSchema(s) })
		})
	}

	invalid := []string{
		"",
		"Public",                            // uppercase
		"1startswithdigit",                  // leading digit
		"has-dash",                          // hyphen
		"has space",                         // space
		"public; DROP TABLE users--",        // injection
		"public\nDROP TABLE users",          // newline injection
		"a" + string(make([]byte, 63)) + "z", // too long
	}
	for _, s := range invalid {
		t.Run("invalid:"+s, func(t *testing.T) {
			assert.Panics(t, func() { auth.MustValidateSchema(s) })
		})
	}
}

// TC-M1: postgres constructors panic on invalid schema.
func TestNewPostgresSessionRepo_PanicsOnInjection(t *testing.T) {
	assert.Panics(t, func() {
		auth.NewPostgresSessionRepo(nil, "public; DROP TABLE users--")
	})
}

func TestNewPostgresMagicLinkRepo_PanicsOnInjection(t *testing.T) {
	assert.Panics(t, func() {
		auth.NewPostgresMagicLinkRepo(nil, "evil-schema")
	})
}

func TestNewPostgresResetRequestRepo_PanicsOnInjection(t *testing.T) {
	assert.Panics(t, func() {
		auth.NewPostgresResetRequestRepo(nil, "")
	})
}

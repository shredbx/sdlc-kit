// Package bosyaml reads a bos app's bos.yaml, the one file both halves of the app take their
// settings from, and turns it into the environment each half reads. The API's variables are the
// ones package config reads; the web's are the ones the web framework reads.
//
// The two halves cannot disagree because neither states the other's address: the API's allowed
// origin is derived from the web's port, and the web's API address from the API's port.
package bosyaml

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// File is the content of a bos.yaml.
type File struct {
	// Prefix scopes the product-named environment variables (with "ACME", ACME_CORS_ORIGINS).
	Prefix string `yaml:"prefix"`
	// SiteName is the name the web shows.
	SiteName string `yaml:"site_name"`
	// Environment is the API's ENVIRONMENT; empty means "dev".
	Environment string `yaml:"environment"`
	API         API    `yaml:"api"`
	Web         Web    `yaml:"web"`
}

// API is the Go half's part of the file.
type API struct {
	Port int `yaml:"port"`
	// DatabaseURL empty means no database: the API still starts and /health reports it disconnected.
	DatabaseURL string `yaml:"database_url"`
}

// Web is the SvelteKit half's part of the file.
type Web struct {
	Port int `yaml:"port"`
}

// Var is one environment variable of a half. Quoted values are written in double quotes.
type Var struct {
	Name   string
	Value  string
	Quoted bool
}

var prefixPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// Load reads and checks a bos.yaml. An unknown key is an error, so a misspelt setting is not
// silently ignored.
func Load(path string) (File, error) {
	fh, err := os.Open(path)
	if err != nil {
		return File{}, err
	}
	defer fh.Close()

	var f File
	dec := yaml.NewDecoder(fh)
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		if errors.Is(err, io.EOF) {
			return File{}, fmt.Errorf("%s: the file is empty", path)
		}
		return File{}, fmt.Errorf("%s: %w", path, err)
	}
	if f.Environment == "" {
		f.Environment = "dev"
	}
	if err := f.validate(); err != nil {
		return File{}, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

func (f File) validate() error {
	switch {
	case !prefixPattern.MatchString(f.Prefix):
		return fmt.Errorf("prefix %q must be upper case letters, digits and underscores, starting with a letter", f.Prefix)
	case strings.TrimSpace(f.SiteName) == "" || strings.ContainsAny(f.SiteName, "\r\n"):
		return errors.New("site_name must be one non-empty line")
	case strings.ContainsAny(f.Environment, " \t\r\n\"'$`\\"):
		return fmt.Errorf("environment %q must be a single plain word", f.Environment)
	case strings.ContainsAny(f.API.DatabaseURL, "\r\n"):
		return errors.New("api.database_url must be one line")
	case f.API.Port < 1 || f.API.Port > 65535:
		return fmt.Errorf("api.port %d is not a port", f.API.Port)
	case f.Web.Port < 1 || f.Web.Port > 65535:
		return fmt.Errorf("web.port %d is not a port", f.Web.Port)
	case f.API.Port == f.Web.Port:
		return fmt.Errorf("api.port and web.port are both %d", f.API.Port)
	}
	return nil
}

// APIEnv is the environment of the API, in a fixed order. jwtSecret is the caller's to choose
// because it is generated once and kept, never written into bos.yaml.
func (f File) APIEnv(jwtSecret string) []Var {
	return []Var{
		{Name: "ENVIRONMENT", Value: f.Environment},
		{Name: "PORT", Value: fmt.Sprint(f.API.Port)},
		{Name: "DATABASE_URL", Value: f.API.DatabaseURL},
		{Name: "JWT_SECRET", Value: jwtSecret},
		{Name: f.Prefix + "_CORS_ORIGINS", Value: fmt.Sprintf("http://localhost:%d", f.Web.Port)},
	}
}

// WebEnv is the environment of the web, in a fixed order.
func (f File) WebEnv() []Var {
	return []Var{
		{Name: "PORT", Value: fmt.Sprint(f.Web.Port)},
		{Name: "PUBLIC_API_URL", Value: fmt.Sprintf("http://localhost:%d", f.API.Port)},
		{Name: "PUBLIC_SITE_NAME", Value: f.SiteName, Quoted: true},
	}
}

// BundleEnv is the .env of the docker compose bundle, in a fixed order: the two ports the
// services publish on the host, and the values the two services read. The ports the services
// listen on inside their containers are fixed by their images, so only the published ones come
// from the file.
func (f File) BundleEnv(jwtSecret string) []Var {
	return []Var{
		{Name: "API_PORT", Value: fmt.Sprint(f.API.Port)},
		{Name: "WEB_PORT", Value: fmt.Sprint(f.Web.Port)},
		{Name: "ENVIRONMENT", Value: f.Environment},
		{Name: "DATABASE_URL", Value: f.API.DatabaseURL},
		{Name: "JWT_SECRET", Value: jwtSecret},
		{Name: f.Prefix + "_CORS_ORIGINS", Value: fmt.Sprintf("http://localhost:%d", f.Web.Port)},
		{Name: "PUBLIC_SITE_NAME", Value: f.SiteName, Quoted: true},
	}
}

// ComposeLines renders variables as the NAME=value lines of a .env file that docker compose
// reads. Compose expands $ inside double quotes, so a value that needs quoting is written in
// single quotes, which it keeps literal. A single quote or a line break in such a value cannot
// be written that way, and is an error rather than a silently different value.
func ComposeLines(vars []Var) (string, error) {
	var b strings.Builder
	for _, v := range vars {
		b.WriteString(v.Name)
		b.WriteByte('=')
		if v.Quoted || needsQuote(v.Value) {
			if strings.ContainsAny(v.Value, "'\r\n") {
				return "", fmt.Errorf("%s: a value with a single quote or a line break cannot be written to a compose .env file", v.Name)
			}
			b.WriteString("'" + v.Value + "'")
		} else {
			b.WriteString(v.Value)
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// Lines renders variables as the NAME=value lines of an env file that a POSIX shell can source.
// A value is quoted when it was asked to be, and also when it holds a character the shell would
// read as syntax.
func Lines(vars []Var) string {
	var b strings.Builder
	for _, v := range vars {
		b.WriteString(v.Name)
		b.WriteByte('=')
		if v.Quoted || needsQuote(v.Value) {
			b.WriteString(doubleQuote(v.Value))
		} else {
			b.WriteString(v.Value)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func needsQuote(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune("_@%+=:,./-", r):
		default:
			return true
		}
	}
	return false
}

// doubleQuote wraps s in double quotes, escaping what a shell still expands inside them.
func doubleQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		if strings.ContainsRune("\\\"$`", r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}

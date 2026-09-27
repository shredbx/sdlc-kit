// Command bos is the command line of a bos app. Today it has two commands:
//
//	bos version
//	bos env [-f bos.yaml] [-o .bos/env]
//
// env writes the two halves' environment files from bos.yaml, api.env and web.env, so the API and
// the web read their settings from one place and cannot disagree. The API's JWT secret is
// generated the first time and kept on every later run.
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/core/bosyaml"
)

const version = "0.0.0"

const usage = `usage:
  bos version
  bos env [-f bos.yaml] [-o .bos/env]
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(errw, usage)
		return 2
	}
	switch args[0] {
	case "version":
		fmt.Fprintf(out, "bos %s\n", version)
		return 0
	case "env":
		return runEnv(args[1:], out, errw)
	default:
		fmt.Fprintf(errw, "bos: unknown command %q\n%s", args[0], usage)
		return 2
	}
}

func runEnv(args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("bos env", flag.ContinueOnError)
	fs.SetOutput(errw)
	file := fs.String("f", "bos.yaml", "the app's bos.yaml")
	dir := fs.String("o", filepath.Join(".bos", "env"), "the folder the env files are written to")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	f, err := bosyaml.Load(*file)
	if err != nil {
		fmt.Fprintf(errw, "bos env: %v\n", err)
		return 1
	}

	apiPath := filepath.Join(*dir, "api.env")
	webPath := filepath.Join(*dir, "web.env")

	secret := existingSecret(apiPath)
	if secret == "" {
		if secret, err = newSecret(); err != nil {
			fmt.Fprintf(errw, "bos env: %v\n", err)
			return 1
		}
	}

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fmt.Fprintf(errw, "bos env: %v\n", err)
		return 1
	}
	// api.env holds the secret, so only its owner may read it.
	if err := writeFile(apiPath, bosyaml.Lines(f.APIEnv(secret)), 0o600); err != nil {
		fmt.Fprintf(errw, "bos env: %v\n", err)
		return 1
	}
	if err := writeFile(webPath, bosyaml.Lines(f.WebEnv()), 0o644); err != nil {
		fmt.Fprintf(errw, "bos env: %v\n", err)
		return 1
	}
	fmt.Fprintf(out, "wrote %s and %s\n", apiPath, webPath)
	return 0
}

// existingSecret returns the JWT_SECRET already in an api.env, or "" when there is none.
func existingSecret(path string) string {
	fh, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer fh.Close()
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "JWT_SECRET="); ok && v != "" {
			return v
		}
	}
	return ""
}

// newSecret returns 32 random bytes as 64 hex characters.
func newSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// writeFile writes to a temporary file in the same folder and renames it into place, so a reader
// never sees half a file, and an existing file's mode is replaced by mode.
func writeFile(path, content string, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".bos-env-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name) // a no-op after a successful rename
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

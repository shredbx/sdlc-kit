// Command bos is the command line of a bos app. Today it has two commands:
//
//	bos version
//	bos env [-f bos.yaml] [-o .bos/env] [-p dev|bundle]
//
// env writes the settings of the app's two halves from bos.yaml, so the API and the web read
// them from one place and cannot disagree. Profile dev (the default) writes the environment
// files of a native run, api.env and web.env. Profile bundle writes the one .env that docker
// compose reads for the app's bundle. The API's JWT secret is generated the first time and kept
// on every later run.
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
  bos env [-f bos.yaml] [-o .bos/env] [-p dev|bundle]
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
	profile := fs.String("p", "dev", "dev: api.env and web.env for a native run; bundle: the one .env docker compose reads")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *profile != "dev" && *profile != "bundle" {
		fmt.Fprintf(errw, "bos env: unknown profile %q (dev or bundle)\n%s", *profile, usage)
		return 2
	}

	f, err := bosyaml.Load(*file)
	if err != nil {
		fmt.Fprintf(errw, "bos env: %v\n", err)
		return 1
	}

	// The secret lives in the file that holds it: api.env for a native run, .env for the bundle.
	secretFile := "api.env"
	if *profile == "bundle" {
		secretFile = ".env"
	}
	secret := existingSecret(filepath.Join(*dir, secretFile))
	if secret == "" {
		if secret, err = newSecret(); err != nil {
			fmt.Fprintf(errw, "bos env: %v\n", err)
			return 1
		}
	}

	// Every file's content is made before any is written, so a refused value leaves nothing behind.
	// A file with the secret in it is readable by its owner only.
	type envFile struct {
		path, content string
		mode          os.FileMode
	}
	var files []envFile
	if *profile == "bundle" {
		content, err := bosyaml.ComposeLines(f.BundleEnv(secret))
		if err != nil {
			fmt.Fprintf(errw, "bos env: %s: %v\n", *file, err)
			return 1
		}
		files = []envFile{{filepath.Join(*dir, ".env"), content, 0o600}}
	} else {
		files = []envFile{
			{filepath.Join(*dir, "api.env"), bosyaml.Lines(f.APIEnv(secret)), 0o600},
			{filepath.Join(*dir, "web.env"), bosyaml.Lines(f.WebEnv()), 0o644},
		}
	}

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fmt.Fprintf(errw, "bos env: %v\n", err)
		return 1
	}
	paths := make([]string, len(files))
	for i, ef := range files {
		if err := writeFile(ef.path, ef.content, ef.mode); err != nil {
			fmt.Fprintf(errw, "bos env: %v\n", err)
			return 1
		}
		paths[i] = ef.path
	}
	fmt.Fprintf(out, "wrote %s\n", strings.Join(paths, " and "))
	return 0
}

// existingSecret returns the JWT_SECRET already in an env file, or "" when there is none.
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

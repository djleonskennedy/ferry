// Package scan discovers env-style files in a project root.
//
// No git dependency. The walk skips well-known noise directories and matches
// filename globs; the user is the final filter (they edit ferry.toml after init).
package scan

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultGlobs are filename patterns considered env files.
//
// Coverage:
//   - .env              → ".env" and "*.env" (the latter also catches dev.env,
//                         prod.env, staging.env, app.env, etc.)
//   - .env.local, .env.development, .env.production → ".env.*"
//   - .envrc (direnv)   → ".envrc"
var DefaultGlobs = []string{".env", ".env.*", "*.env", ".envrc"}

// SkipDirs are directory names pruned during the walk (not descended into).
var SkipDirs = map[string]struct{}{
	".git":          {},
	".ferry":        {},
	".idea":         {},
	".vscode":       {},
	"node_modules":  {},
	"vendor":        {},
	"dist":          {},
	"build":         {},
	"out":           {},
	".next":         {},
	".nuxt":         {},
	".svelte-kit":   {},
	"target":        {},
	".venv":         {},
	"venv":          {},
	"__pycache__":   {},
	".cache":        {},
	".parcel-cache": {},
	".turbo":        {},
	"coverage":      {},
}

// Candidate is an env-style file found by Discover.
type Candidate struct {
	RelPath string // forward-slash relative to root
	Size    int64
}

// Discover walks root and returns matching candidates.
// If extraGlobs is non-nil, it replaces DefaultGlobs.
func Discover(root string, extraGlobs []string) ([]Candidate, error) {
	globs := DefaultGlobs
	if extraGlobs != nil {
		globs = extraGlobs
	}
	var out []Candidate
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p == root {
				return nil
			}
			if _, skip := SkipDirs[d.Name()]; skip {
				return fs.SkipDir
			}
			return nil
		}
		if !matchesAny(d.Name(), globs) {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, Candidate{
			RelPath: filepath.ToSlash(rel),
			Size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelPath < out[j].RelPath })
	return out, nil
}

func matchesAny(name string, globs []string) bool {
	for _, g := range globs {
		ok, err := filepath.Match(g, name)
		if err == nil && ok {
			return true
		}
	}
	return false
}

// HasGlobChars reports whether s contains characters that filepath.Match would
// treat specially. Useful for callers that want to validate user-provided globs.
func HasGlobChars(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

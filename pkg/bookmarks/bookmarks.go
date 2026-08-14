// Package bookmarks loads and saves named MongoDB connection strings as
// TOML files under a directory (by default ~/.gonosequel/bookmarks/),
// so a URL doesn't have to be retyped or kept in shell history.
package bookmarks

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Bookmark is a single named connection.
type Bookmark struct {
	Name string `toml:"-"`
	URL  string `toml:"url"`
}

// DefaultDir returns ~/.gonosequel/bookmarks, creating no
// directories itself — callers create it on first Save.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".gonosequel", "bookmarks"), nil
}

// Load reads a single bookmark by name from dir.
func Load(dir, name string) (*Bookmark, error) {
	path := filepath.Join(dir, name+".toml")
	var b Bookmark
	if _, err := toml.DecodeFile(path, &b); err != nil {
		return nil, fmt.Errorf("load bookmark %q: %w", name, err)
	}
	b.Name = name
	return &b, nil
}

// Save writes a bookmark to dir, creating the directory if needed.
func Save(dir string, b Bookmark) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create bookmarks directory: %w", err)
	}
	path := filepath.Join(dir, b.Name+".toml")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open bookmark file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(b); err != nil {
		return fmt.Errorf("write bookmark %q: %w", b.Name, err)
	}
	return nil
}

// List returns every bookmark saved under dir, sorted by name. A missing
// directory is not an error — it just means there are no bookmarks yet.
func List(dir string) ([]Bookmark, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read bookmarks directory: %w", err)
	}

	var out []Bookmark
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".toml")
		b, err := Load(dir, name)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

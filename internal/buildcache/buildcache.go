// Package buildcache persists parsed pages between builds so an incremental
// build can skip re-parsing unchanged content. The cache is keyed by a header
// (binary identity, resolved config, theme tree) plus a per-file content hash;
// any header mismatch forces a full rebuild, so the cache can never produce
// output that differs from a clean build.
package buildcache

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofuego/fuego/core"
)

// cacheVersion is bumped when the on-disk format changes; a mismatch is
// treated as a cache miss (full rebuild).
const cacheVersion = 1

func init() {
	// Envelope values are arbitrary YAML scalars/containers held in any.
	gob.Register(map[string]any{})
	gob.Register([]any{})
	gob.Register(time.Time{})
}

// Header identifies the build environment the cache was produced under. If any
// field changes, the whole cache is invalid.
type Header struct {
	Version    int
	BinaryID   string
	ConfigHash string
	ThemeHash  string
}

// ParsedPage is the post-PARSE state of a page, restored on a cache hit.
type ParsedPage struct {
	ContentHash string
	Envelope    core.Envelope
	Nodes       []core.Node
	Type        string
	Layout      string
	IsRaw       bool
}

// Cache is the on-disk build cache.
type Cache struct {
	Header  Header
	Pages   map[string]ParsedPage // keyed by content RelPath
	Outputs []string              // page output relpaths from the last build
}

// New returns an empty cache for the given header, stamped with the current
// cache version.
func New(h Header) *Cache {
	h.Version = cacheVersion
	return &Cache{Header: h, Pages: map[string]ParsedPage{}}
}

// Load reads the cache at path. A missing or unreadable cache returns an empty
// cache and ok=false — callers treat that as "no usable cache" rather than an
// error, so corruption simply triggers a full rebuild.
func Load(path string) (*Cache, bool) {
	f, err := os.Open(path)
	if err != nil {
		return &Cache{Pages: map[string]ParsedPage{}}, false
	}
	defer f.Close()

	var c Cache
	if err := gob.NewDecoder(f).Decode(&c); err != nil {
		return &Cache{Pages: map[string]ParsedPage{}}, false
	}
	if c.Pages == nil {
		c.Pages = map[string]ParsedPage{}
	}
	return &c, true
}

// Save writes the cache to path, creating parent directories.
func Save(path string, c *Cache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating cache file: %w", err)
	}
	defer f.Close()
	if err := gob.NewEncoder(f).Encode(c); err != nil {
		return fmt.Errorf("encoding cache: %w", err)
	}
	return nil
}

// Valid reports whether the cache was produced by the same engine binary,
// resolved config, and theme — i.e. whether its parsed pages may be reused.
// The version field of h is ignored; the cache's own version must match.
func (c *Cache) Valid(h Header) bool {
	return c.Header.Version == cacheVersion &&
		c.Header.BinaryID == h.BinaryID &&
		c.Header.ConfigHash == h.ConfigHash &&
		c.Header.ThemeHash == h.ThemeHash
}

// HashBytes returns the hex SHA-256 of b.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// BinaryID hashes the running executable so a rebuilt engine (new parsers,
// hooks, or pack code) invalidates the cache. Under `go run` the binary is a
// throwaway, so each run gets a fresh id and rebuilds fully — which is correct.
func BinaryID() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		return "", err
	}
	return HashBytes(data), nil
}

// ThemeHash hashes the user theme directory and every pack theme FS, so a
// template change invalidates the cache. Inputs are sorted for determinism.
func ThemeHash(themeDir string, packThemes []fs.FS) string {
	h := sha256.New()
	addFS := func(prefix string, fsys fs.FS) {
		var entries []string
		fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			entries = append(entries, p)
			return nil
		})
		sort.Strings(entries)
		for _, p := range entries {
			b, err := fs.ReadFile(fsys, p)
			if err != nil {
				continue
			}
			fmt.Fprintf(h, "%s/%s\x00", prefix, p)
			h.Write(b)
			h.Write([]byte{0})
		}
	}

	if info, err := os.Stat(themeDir); err == nil && info.IsDir() {
		addFS("user", os.DirFS(themeDir))
	}
	for i, t := range packThemes {
		if t != nil {
			addFS(fmt.Sprintf("pack%d", i), t)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// OrphanedOutputs returns the entries in old that are not in current — output
// files that should be deleted because their page no longer exists.
func OrphanedOutputs(old, current []string) []string {
	keep := make(map[string]bool, len(current))
	for _, p := range current {
		keep[p] = true
	}
	var orphans []string
	for _, p := range old {
		if !keep[p] {
			orphans = append(orphans, p)
		}
	}
	sort.Strings(orphans)
	return orphans
}

// PruneEmptyDirs removes now-empty directories under root left behind by
// deleted outputs, walking deepest-first. root itself is never removed.
func PruneEmptyDirs(root string, removed []string) {
	seen := map[string]bool{}
	var dirs []string
	for _, rel := range removed {
		dir := filepath.Dir(filepath.Join(root, rel))
		for dir != root && len(dir) > len(root) && !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
			dir = filepath.Dir(dir)
		}
	}
	// Deepest paths first.
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		if entries, err := os.ReadDir(d); err == nil && len(entries) == 0 {
			os.Remove(d)
		}
	}
}

// OutputRelPath returns the output file path (relative to the output dir) for a
// page URL — the same mapping the render phase uses to write index.html.
func OutputRelPath(url string) string {
	rel := path.Join(filepath.ToSlash(url), "index.html")
	return strings.TrimPrefix(rel, "/")
}

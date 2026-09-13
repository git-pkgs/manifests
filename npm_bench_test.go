package manifests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func npmLockHistory(tb testing.TB, format, count int) []byte {
	tb.Helper()
	entries := make(map[string]any, count)
	for i := range count {
		name := fmt.Sprintf("@scope/package-%d", i)
		key := name
		if format == 3 {
			key = "node_modules/" + name
		}
		entries[key] = map[string]any{
			"version":   "1.2.3",
			"resolved":  "https://registry.npmjs.org/" + name + "/-/package-1.2.3.tgz",
			"integrity": "sha512-Zml4dHVyZS1wYWNrYWdlLWludGVncml0eQ==",
			"license":   "MIT", "dev": i%2 == 0,
		}
	}
	document := map[string]any{"name": "benchmark-project", "version": "1.0.0", "lockfileVersion": format}
	if format == 3 {
		entries[""] = map[string]any{"dependencies": map[string]string{"@scope/package-1": "^1.2.0"}}
		document["packages"] = entries
	} else {
		document["dependencies"] = entries
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		tb.Fatal(err)
	}
	return data
}

func TestNPMLargeLockfile(t *testing.T) {
	const count = 10000
	for _, format := range []int{1, 3} {
		t.Run(fmt.Sprintf("v%d", format), func(t *testing.T) {
			result, err := Parse("package-lock.json", npmLockHistory(t, format, count))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Dependencies) != count {
				t.Fatalf("got %d dependencies, want %d", len(result.Dependencies), count)
			}
			seen := make(map[string]bool, count)
			for _, dep := range result.Dependencies {
				if seen[dep.Name] {
					t.Fatalf("duplicate dependency: %s", dep.Name)
				}
				seen[dep.Name] = true
				if dep.Version != "1.2.3" || dep.PURL != "pkg:npm/%40"+strings.TrimPrefix(dep.Name, "@")+"@1.2.3" || dep.Integrity != "sha512-Zml4dHVyZS1wYWNrYWdlLWludGVncml0eQ==" {
					t.Fatalf("unexpected dependency: %+v", dep)
				}
				if dep.Direct != (format == 3 && dep.Name == "@scope/package-1") {
					t.Fatalf("unexpected direct flag: %+v", dep)
				}
			}
		})
	}
}

func BenchmarkNPMLockfile(b *testing.B) {
	for _, format := range []int{1, 3} {
		for _, count := range []int{10, 1000, 10000} {
			b.Run(fmt.Sprintf("v%d/packages=%d", format, count), func(b *testing.B) {
				data := npmLockHistory(b, format, count)
				b.ReportAllocs()
				b.SetBytes(int64(len(data)))
				b.ResetTimer()
				for range b.N {
					result, err := Parse("package-lock.json", data)
					if err != nil {
						b.Fatal(err)
					}
					if len(result.Dependencies) != count {
						b.Fatalf("got %d dependencies, want %d", len(result.Dependencies), count)
					}
				}
			})
		}
	}
}

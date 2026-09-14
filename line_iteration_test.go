package manifests

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLineParserEndings(t *testing.T) {
	for _, path := range []string{"golang/go.mod", "golang/go.sum", "golang/go.graph", "pypi/requirements.txt", "pypi/pip-resolved-dependencies.txt", "nuget/paket.lock"} {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile("testdata/" + path)
			if err != nil {
				t.Fatal(err)
			}
			name := path[strings.LastIndexByte(path, '/')+1:]
			lf := bytes.TrimRight(bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n")), "\n")
			want, err := Parse(name, lf)
			if err != nil {
				t.Fatal(err)
			}
			if len(want.Dependencies) == 0 {
				t.Fatal("fixture has no dependencies")
			}
			variants := [][]byte{append(bytes.Clone(lf), '\n'), append(bytes.Clone(lf), '\n', '\n'), bytes.ReplaceAll(lf, []byte("\n"), []byte("\r\n"))}
			for _, data := range variants {
				got, err := Parse(name, data)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("line endings changed %s output", name)
				}
			}
		})
	}
}

func lineBenchmarkInput(name string, count int) []byte {
	var s strings.Builder
	switch name {
	case "go.mod":
		s.WriteString("module example.com/demo\n\ngo 1.26\nrequire (\n")
	case "paket.lock":
		s.WriteString("NUGET\n  remote: https://api.nuget.org/v3/index.json\n  specs:\n")
	}
	for i := range count {
		switch name {
		case "go.mod":
			fmt.Fprintf(&s, "example.com/package%d v1.2.3 // indirect\n", i)
		case "go.sum":
			fmt.Fprintf(&s, "example.com/package%d v1.2.3 h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n", i)
		case "go.graph":
			fmt.Fprintf(&s, "example.com/demo example.com/package%d@v1.2.3\n", i)
		case "requirements.txt", "pip-resolved-dependencies.txt":
			fmt.Fprintf(&s, "package%d==1.2.3\n", i)
		case "paket.lock":
			fmt.Fprintf(&s, "    Package%d (1.2.3)\n", i)
		}
	}
	if name == "go.mod" {
		s.WriteString(")\n")
	}
	return []byte(s.String())
}
func BenchmarkLineParsers(b *testing.B) {
	for _, name := range []string{"go.mod", "go.sum", "go.graph", "requirements.txt", "pip-resolved-dependencies.txt", "paket.lock"} {
		b.Run(name, func(b *testing.B) {
			for _, count := range []int{100, 10000} {
				b.Run(fmt.Sprint(count), func(b *testing.B) {
					content := lineBenchmarkInput(name, count)
					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						result, err := Parse(name, content)
						if err != nil {
							b.Fatal(err)
						}
						if len(result.Dependencies) != count {
							b.Fatalf("got %d dependencies, want %d", len(result.Dependencies), count)
						}
					}
				})
			}
		})
	}
}

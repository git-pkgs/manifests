package manifests

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func TestNPMV2LockfileCRLF(t *testing.T) {
	content, err := os.ReadFile("testdata/npm/npm-lockfile-version-2/package-lock.json")
	if err != nil {
		t.Fatal(err)
	}
	lf := bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	crlf := bytes.ReplaceAll(lf, []byte("\n"), []byte("\r\n"))
	for _, filename := range []string{"package-lock.json", "npm-shrinkwrap.json"} {
		t.Run(filename, func(t *testing.T) {
			want, err := Parse(filename, lf)
			if err != nil {
				t.Fatal(err)
			}
			if len(want.Dependencies) != 3 {
				t.Fatalf("LF fixture has %d dependencies, want 3", len(want.Dependencies))
			}
			got, err := Parse(filename, crlf)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("CRLF result differs from LF: got %d dependencies, want %d", len(got.Dependencies), len(want.Dependencies))
			}
		})
	}
}

package rebar

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestRebarLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/rebar/rebar.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &rebarLockParser{}
	res, err := parser.Parse("rebar.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	expected := map[string]struct {
		version   string
		integrity string
	}{
		"hex_core": {
			"0.10.3",
			"sha256-4516C43F1CD22C9E4ABBBC3C4F5EBBC84EF24F7D8BF0D3D6BF0F3D5818F8612D",
		},
		"verl": {
			"1.1.1",
			"sha256-36B66CC8E9B6C30BCAEB35A4674ABAF9A4C4E5E30C8E7A0E4FC3E4F4F4E4E4F4",
		},
		"ssl_verify_fun": {
			"1.1.7",
			"sha256-BDB0D2471F453C88FF3908E7686F86F9BE327D065CC1EC16FA4540197EA04680",
		},
	}

	for name, want := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != want.version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, want.version)
		}
		if dep.Integrity != want.integrity {
			t.Errorf("%s integrity = %q, want %q", name, dep.Integrity, want.integrity)
		}
	}
}

func TestRebarLockFallsBackToLegacyPackageHash(t *testing.T) {
	content := []byte(`{"1.1.0",
[{<<"hex_core">>,{pkg,<<"hex_core">>,<<"0.10.3">>},0}]}.
[
{pkg_hash,[
 {<<"hex_core">>, <<"21E84B3AB21EEE6A1EAA56B69624E0D7D82F61F148B4C7441B4692FA2C48E0C1">>}]}
].`)

	res, err := (&rebarLockParser{}).Parse("rebar.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(res.Dependencies))
	}

	want := "sha256-21E84B3AB21EEE6A1EAA56B69624E0D7D82F61F148B4C7441B4692FA2C48E0C1"
	if got := res.Dependencies[0].Integrity; got != want {
		t.Errorf("integrity = %q, want %q", got, want)
	}
}

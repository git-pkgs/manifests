module github.com/git-pkgs/manifests

go 1.25.6

require (
	github.com/BurntSushi/toml v1.6.0
	github.com/bazelbuild/buildtools v0.0.0-20260121081817-bbf01ec6cb49
	github.com/git-pkgs/purl v0.1.3
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/git-pkgs/vers v0.2.1 // indirect
	github.com/package-url/packageurl-go v0.1.3 // indirect
)

replace github.com/package-url/packageurl-go => github.com/git-pkgs/packageurl-go v0.0.0-20260115093137-a0c26f7ee19e

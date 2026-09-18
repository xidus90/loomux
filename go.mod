module github.com/xidus90/loomux

go 1.25.0

toolchain go1.27.0

require (
	github.com/BurntSushi/toml v1.6.0
	golang.org/x/sys v0.18.0
	gopkg.in/yaml.v3 v3.0.1
)

require golang.org/x/text v0.38.0

replace github.com/BurntSushi/toml => ./third_party/toml

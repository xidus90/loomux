module github.com/xidus90/loomux

go 1.26.0

toolchain go1.27.4

require (
	github.com/BurntSushi/toml v1.6.0
	golang.org/x/sys v0.50.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/fsnotify/fsnotify v1.10.1
	github.com/modelcontextprotocol/go-sdk v1.8.0
	github.com/odvcencio/gotreesitter v0.57.0
	golang.org/x/term v0.48.0
	golang.org/x/text v0.44.0
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.38.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/time v0.16.0 // indirect
)

replace github.com/BurntSushi/toml => ./third_party/toml

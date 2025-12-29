module github.com/SeungKang/memshonk

go 1.24.0

require (
	github.com/buger/goterm v1.0.4
	github.com/desertbit/grumble v1.2.0
	github.com/desertbit/readline v1.5.1
	github.com/ebitengine/purego v0.8.3
	github.com/fatih/color v1.18.0
	github.com/mitchellh/go-ps v1.0.0
	golang.org/x/sys v0.39.0
	golang.org/x/term v0.38.0
)

require (
	github.com/desertbit/closer/v3 v3.7.5 // indirect
	github.com/desertbit/columnize v2.1.0+incompatible // indirect
	github.com/desertbit/go-shlex v0.1.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
)

replace github.com/desertbit/grumble => ./internal/vendored/grumble

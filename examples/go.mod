module github.com/sethgrid/kverr/examples

go 1.23.1

replace github.com/sethgrid/kverr => ../

require (
	github.com/rs/zerolog v1.34.0
	github.com/sethgrid/kverr v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.27.1
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

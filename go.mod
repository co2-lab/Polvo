module github.com/co2-lab/polvo

go 1.26.0

tool (
	github.com/jandelgado/gcov2lcov
	github.com/jstemmer/go-junit-report/v2
)

require (
	github.com/charmbracelet/x/vt v0.1.0
	github.com/creack/pty v1.1.24
	golang.org/x/crypto v0.58.0
	golang.org/x/net v0.61.0
	golang.org/x/term v0.47.0
)

require (
	github.com/charmbracelet/colorprofile v0.4.2 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260303162955-0b88c25f3fff // indirect
	github.com/charmbracelet/x/ansi v0.11.7 // indirect
	github.com/charmbracelet/x/exp/ordered v0.1.0 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/jandelgado/gcov2lcov v1.1.1 // indirect
	github.com/jstemmer/go-junit-report/v2 v2.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/mattn/go-runewidth v0.0.23 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace github.com/charmbracelet/x/vt => ./third_party/charmbracelet-x-vt

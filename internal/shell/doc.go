// Package shell implements a custom shell using the following libraries:
//
//   - mvdan.cc/sh
//   - github.com/chzyer/readline
//   - github.com/fatih/color
//
// claude was (sadly) very useful in jamming these libraries together.
// Having looked at the readline library, I was concerned about some
// assumptions it had about the stdin/out/err being real file descriptors
// attached to a terminal. claude created a table describing some of these
// issues, which I am summarizing here for future reference:
//
// - Global Stdin/Stdout/Stderr (readline/std.go:10-14)
//   - Always provide explicit I/O in Config
//
// - Global singleton instance (readline/std.go:16-29)
//   - Never use Line(), Password(), etc. - always NewEx()
//
// - SIGWINCH race condition (readline/utils_unix.go:62-83)
//   - Provide custom FuncOnWidthChanged per session
//
// - RawMode hardcodes FD 0 (readline/utils.go:264-278)
//   - Provide no-op FuncMakeRaw/FuncExitRaw
//
// - DefaultIsTerminal checks FD 0 (readline/utils_unix.go:52-54)
//   - Provide custom FuncIsTerminal returning true
//
// I hate that I used claude for this, but this code is an absolute pain
// to write and think about.
package shell

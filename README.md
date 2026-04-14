# memshonk

memshonk is an experimental command-line debugger companion that tries to
fill the functionality gaps between debuggers. Think of it as a cross
between `gdb`, `rizin`, and Cheat Engine. It is not meant to replace
a debugger, but supplement it.

Please note that memshonk is in its very early stages of development.
There are bugs and missing functionality.

## Demo

Here is a demo of memshonk in video form: https://youtu.be/Z5twBlVa9R4

Please find a text-based demo of memshonk below:

> Note: In git-bash shell on Windows, running memshonk using
> `winpty memshonk ARGS...` improves terminal rendering.

Start a session using a project file:

```console
$ memshonk -p examples/vim.txt
(short-seal) $
```

Attach to the target program identified by the project file. The process'
PID appears in the shell prompt once attached:

```console
(short-seal) $ attach
attached to "vim.exe", pid: 49564, base addr: 0x100400000
(short-seal) [49564] $
```

Search for a string in memory. The result is the address where it was found:

```console
(short-seal) [49564] $ scan -d string hello
searching..............................................................
0x10079ed7c
```

Read and overwrite memory at that address:

```console
(short-seal) [49564] $ readm -a 0x10079ed7c -s 5 -d raw
000000010079ed81   68 65 6c 6c  6f                                      |hello           |

(short-seal) [49564] $ writem -a 0x10079ed7c -d raw -v world

(short-seal) [49564] $ readm -a 0x10079ed7c -s 5 -d raw
000000010079ed81   77 6f 72 6c  64                                      |world           |
```

## Features

- Read, write, and watch memory in real time
- View process memory mappings and permissions
- Search memory based on data type or byte patterns
- Run side-by-side with your favorite debugger on Linux and Windows
  - Supports switching between `ptrace` and `procfs` memory management modes
    on Linux to allow side-by-side use with tools like `gdb` and `pwndbg`
- Text-based UI via sh-like shell provides access to external programs (e.g.
  grep, cat, ls), pipes, shell history, reverse search, and tab completion
- Multi-session support allows multiple clients. Great for providing
  multiple windows or debugging with friends
- Client-daemon architecture allows for long-running debugging sessions and
  protection against accidental exits or lack of tools like tmux
- Project files make it easy to attach to a program by its executable file
  name, set pre-defined variables, and automatically load plugins
- Plugin support via shared libraries (`.so`, `.dll` files)
  - A Rust library named [`mskit`](plugin-api/mskit) is provided
    as a building block
  - Users can specify optional automation to run when reloading plugins,
    making it easy to, for example, recompile a plugin from source
- Scripting interface via `mrun` command provides access to memshonk
  commands using a POSIX shell syntax

## Supported systems

memshonk supports the following operating systems:

- FreeBSD
- Linux
- Windows

Support for other Unix-like OSes is definitely possible. We just have not
had time to work on that.

## Installation

Prebuilt executables are not currently provided. To build from source,
refer to the [Development document](./docs/development/README.md).

## Documentation

- [Commands](./docs/commands/README.md)
  - For a full list of commands and help topics, run `help` or `help [TOPIC]`
    in memshonk
- [Configuration file examples](./examples/)
- [Plugins](./docs/plugins/README.md)
- [Design and security model](./docs/design/README.md)
- [Development documentation](./docs/development/README.md)
- [Future plans, limitations, and known issues](./TODO.md)

## Special thanks

We would like to acknowledge and thank the following people and projects
for their work on various libraries and code that memshonk depends on.
memshonk would not be possible without their awesome work:

- [awgh](https://github.com/awgh) for their work on a Go-based PE file
  parser which enabled us to parse PE file symbols using the Go standard
  library. Our plugin system relies on exported library symbols. Without
  awgh's code, we would not be able to parse plugin symbols on Windows
- [ChenYe](https://github.com/chzyer) and
  [Daniel Martí](https://github.com/mvdan) for respectively developing the
  `github.com/chzyer/readline` and `github.com/mvdan/sh` libraries, which
  enabled us to build a very powerful shell with minimal dependencies
- [Grumble project](https://github.com/desertbit/grumble) for providing
  an easy-to-use shell / TUI library that allowed us to get memshonk
  started. memshonk would not be where it is today without grumble
- [Igor Café](https://github.com/igorcafe) for their `xx` library
  which became the basis for our `internal/hexdump` library
- [Leonid Bugaev](https://github.com/buger) for their `goterm` library
  which we have forked into `internal/vendored/goterm`
- [Mahmud "hjr265" Ridwan](https://github.com/hjr265) for their `ptrace`
  Go library work which served as the basis for our `internal/ptrace`
  library
- [Nominal Animal](https://stackoverflow.com/a/18603766) for their very
  detailed explanation of `ptrace(2)` and its many byzantine rules
- [purego project](https://github.com/ebitengine/purego) for enabling
  use of shared libraries in Go and giving us an opportunity to build
  a really neat plugin system

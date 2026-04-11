# memshonk

memshonk is an experimental command-line debugger companion that tries to
fill the functionality gaps between debuggers. Think of it as a cross
between `gdb`, `rizin`, and CheatEngine. It is not meant to replace
a debugger, but supplement it.

Please note that memshonk is in its very early stages of development.
There are bugs and missing functionality.

## Features

- Read, write, and watch memory in real time
- View process memory mappings and permissions
- Search memory based on data type or byte patterns
- Run side-by-side with your favorite debugger on Linux and Windows
  - Supports switching between `ptrace` and `procfs` memory management modes
    on Linux to allow side-by-side use with tools like `gdb` and `pwndbg`
- Text-based UI
- Full sh-like shell with access to external programs (e.g. grep, cat, ls),
  pipes, shell history, reverse search, and tab completion
- Multi-session support allows multiple clients. Great for providing
  multiple windows or debugging with friends
- Client-daemon architecture allows for long-running debugging sessions and
  protection against accidental exits or lack of tools like tmux
- Project files make it easy to attach to a program by its executable file
  name, set pre-defined variables, and automatically load plugins
- Plugin support via dynamically-loaded libraries
  - A Rust library named [`mskit`](plugin-api/mskit) is provided as
    a building block. Refer to the [examples](examples/plugins) for
    details
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

## Demo

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
(short-seal) [49564] $ find -d string hello
searching..............................................................
0x10079ed7c
```

Read and overwrite memory at that address:

```console
(short-seal) [49564] $ readm -a 0x10079ed7c -s 5 -d raw
000000010079ed81   68 65 6c 6c  6f                                      |hello           |

(short-seal) [49564] $ writem -a 0x10079ed7c -d world

(short-seal) [49564] $ readm -a 0x10079ed7c -s 5 -d raw
000000010079ed81   77 6f 72 6c  64                                      |world           |
```

## Commands

Refer to the [Commands document](./docs/commands/README.md) for a full
list of commands and topics. Within memshonk, type `help` or `help [TOPIC]`.

## Project file syntax

Refer to the [example configuration files](./examples/).

## Plugins

Refer to the [Plugin document](./docs/plugins/README.md).

## Development documentation

Refer to the [Development document](./docs/development/README.md).

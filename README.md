# memshonk

TODO: needs to reflect more of it being a companion tool and fix some grammer stuff

memshonk is a command-line debugger for dynamic analysis of process memory.
Using a client-daemon architecture, it provides an interactive shell
with POSIX shell syntax, pipes, job control, and execution of both external
programs and built-in memshonk commands. The daemon is a background process
that does the actual memory analysis work, while clients act as interactive
frontends that send commands and display results.

memshonk is like Wireshark, but for process memory. It provides
an interactive shell that supports POSIX shell syntax, pipes,
job control, and execution of external programs and internal
memshonk commands.

## Features

- Read, write, and watch memory
- Search memory based on data type or byte patterns with the `find` command
- Client-daemon architecture for long-running debugging sessions
- Plugin support for additional functionality
- Multi-session support (multiple clients connecting to the same daemon)
- Switchable between ptrace and procfs memory management modes (Linux only)
- Attach to a program by PID, file path, or configuration file
- Full shell with access to external programs (e.g. grep, cat, echo), shell history, reverse search, and tab completion
- Run automation scripts with `mrun`, using the same shell syntax and built-in commands as the interactive shell

## Installation

To build from source, see the [Development document](./docs/development/README.md).

## Commands

See the [Commands document](./docs/commands/commands.md) for a full list of commands and topics.
Within memshonk, type `help` or `help [TOPIC]` for reference.

## Supported Systems

- Windows
- Linux
- FreeBSD

## Development and Building from Source

Refer to the [Development document](./docs/development/README.md).

## Example Configuration File

Refer to the [Example configuration file](./examples/vim-windows.txt).

## Example: Running Memshonk

> Note: Running with `winpty` can help with terminal rendering on Windows.

View usage:

```console
$ winpty ./memshonk.exe -h
SYNOPSIS
  memshonk -h
  memshonk [options] -e EXECUTABLE-FILE-PATH
  memshonk [options] -p PROJECT-FILE-PATH

DESCRIPTION

OPTIONS
  -S id
        Use a custom session id
  -e path
        Load the specified executable by its path and use an empty project
  -h    Display this information
  -p path
        Load a project file by its path
```

Start a session using a configuration file and a custom session ID:

```console
$ winpty ./memshonk.exe -p examples/vim-windows.txt -S short-seal
```

Attach to the target program. The PID appears in the shell prompt once attached:

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
(short-seal) [49564] $ readm -s 5 -d raw 0x10079ed7c
000000010079ed81   68 65 6c 6c  6f                                      |hello           |

(short-seal) [49564] $ writem -addr 0x10079ed7c -data world

(short-seal) [49564] $ readm -s 5 -d raw 0x10079ed7c
000000010079ed81   77 6f 72 6c  64                                      |world           |
```

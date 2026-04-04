# memshonk

memshonk is a command-line debugger for dynamic analysis of process memory.
Useful for reverse engineering, debugging closed-source software, and similar
tasks. Using a client-daemon architecture, it provides an interactive shell
with POSIX shell syntax, pipes, job control, and execution of both external
programs and built-in memshonk commands.

`TODO INTRODUCE WHAT DAEMON MEANS. MODEL OF CLIENT AND DAEMON, BACKGROUND PROCESS THAT IS ACTUALLY DOING THE WORK.`
Multiple clients can connect to the same daemon simultaneously, each with their own I/O and command history.

## Features

- Read, write, and watch memory
`TODO CHANGE THE NAME OF FIND, TO SCAN OR SOMETHING ELSE`
- Search memory based on data type or byte patterns with the `find` command
- Client-daemon architecture for long-running debugging sessions
- Plugin support for additional functionality
- Multi-session support (multiple clients connecting to the same daemon)
- Switchable between ptrace and procfs memory management modes (Linux only)
- Attach to a program by PID, file path, or configuration file
- Full shell with access to external programs (e.g. grep, cat, echo), shell history, reverse search, and tab completion

## Installation

To build from source, see the [Development document](./docs/development/README.md).

## Commands

Type `help` in memshonk to list all commands, or `help [TOPIC]` for details on a specific topic.

TOPICS

```
datatypes - Data types usable with various memory manipulation commands
formats   - Supported data formatting (encoding) options
pattern   - Pattern string format used in the "find" command and other commands
```

Use `[COMMAND] -h` to list full options for a specific command.

COMMANDS

> Note: `readm` and `writem` are named to avoid shadowing the POSIX shell
> built-in `read` command.

```
attach   - attach to the process
daemon   - manage the server daemon
detach   - detach from the process
find     - find data in a process's memory
help     - list available commands and help topics
plugins  - manage plugins
readm    - read data from an address
session  - manage session (list, inspect, or remove sessions)
shonkset - set configuration options
vmmap    - view the process's memory regions
watch    - watch data at an address for changes
writem   - write value to an address
```

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

```
SYNOPSIS
  memshonk -h
  memshonk [options] EXECUTABLE-PATH
  memshonk [options] -p PROJECT-FILE-PATH

OPTIONS
  -h          display help
  -p PATH     use a project file instead of specifying an executable path
  -S ID       use a custom session ID
```

```sh
# Run memshonk with configuration file vim-windows.txt and session ID short-seal
winpty ./memshonk.exe -p examples/vim-windows.txt -S short-seal

# attach to program vim
(short-seal) $ attach
attached to "vim.exe", pid: 49564, base addr: 0x100400000

# pid shows in shell prompt
(short-seal) [49564] $

# search for string hello in program
# 0x10079ed7c is the address of the result
(short-seal) [49564] $ find -d string hello
searching..............................................................
0x10079ed7c

# read 5 bytes of data from address 0x10079ed7c
(short-seal) [49564] $ readm -s 5 -d raw 0x10079ed7c
000000010079ed81   68 65 6c 6c  6f                                      |hello           |

# overwrite the string at that address with "world"
(short-seal) [49564] $ writem -addr 0x10079ed7c -data world

# confirm the write
(short-seal) [49564] $ readm -s 5 -d raw 0x10079ed7c
000000010079ed81   77 6f 72 6c  64                                      |world           |
```

# TODO

## next stopping point

- readme
- support stdin reading for each command including supporting io.closer for cancellation
- command output support (access the result of previous commands)
- allow overwriting of executable mapped object name
  - error: failed to attach to process 220785 ("vim") - attach failure - failed to get mapped object for exe - failed to find a match for an object named: "vim" (searched through: ["vim.basic" "locale-archive" "libpthread.so.0" "libpcre2-8.so.0.11.2" "libc.so.6" "libgpm.so.2" "libacl.so.1.1.2301" "libsodium.so.23.3.0" "libselinux.so.1" "libtinfo.so.6.4" "libm.so.6" "ld-linux-x86-64.so.2"])

## diff

- diff the output of 2 commands

## multi session support

- figure out better way of doing copyAndAddBackslashRLoop()
- instant messaging :)

## plugin

- add context as arguments to plugin.load and plugin.unload
- improve plugin user experience when unloading / reloading
  (i.e., be able to load a previously-unloaded plugin using
  only its name - maybe we can have a name -> plugin info
  cache?)

## mskit

- helper function for reading a pointer from process
- helper function for reading data from process using a pointer to a `Vec<u8>`
- investigate rust macro to generate ffi functions

## vmmap

- fix the permissions are not showing up correctly on windows

## progctl

- Need to implement Suspend and Resume methods for WindowsProcess
- Search for process using its path (we are currently limited to
  searching by its name)

## memory

- Implement a Reader object for a process that knows its bounds based on
  mapped objects
- BufferedReader: Implement constructor-like functions that either constrain
  the range based on an arbitrary range or base and end addrs of a mapped
  object

## command ideas

- `command addr number_of_pointers` tries to determine if there are pointers at
  this addr
- outputs command
- command performance measuring

## find

- "*" support for super wildcard pattern search, maybe not at the end
- add configurable logging for when error occurs
- improve find performance (increase size read, or with start/end address)

## kernel32

- make Read/Write process memory behave like io interfaces, return the number of bytes read/written

## events

- consider making log messages available to events to logging logic

## shell

- memshonk exec mode to run a memshonk command (allow you to run memshonk from shell script)
- test scripting within memshonk shell
- need to consider tab completion for external program
- disable color (needs to be a Session-level setting)
- allow execution of external programs to be disabled
- replace shlex code in `internal/shell/completer.go` (`splitWords`)

## projects

- Create project file based on current settings (i.e., serialize current
  project.Project object to ini)

## bugs

- hanging when running "winpty go run -tags plugins_execonreload main.go -p examples/mass-effect-3.txt" stuck at "connecting to daemon..."
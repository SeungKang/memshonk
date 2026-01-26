# TODO

## next stopping point

~~- unix support~~
- command output support (access the result of previous commands)
- plugin command for me3
~~- don't make ctrl + c exit~~
~~- retain shell command history~~
- allow overwriting of executable mapped object name
  - error: failed to attach to process 220785 ("vim") - attach failure - failed to get mapped object for exe - failed to find a match for an object named: "vim" (searched through: ["vim.basic" "locale-archive" "libpthread.so.0" "libpcre2-8.so.0.11.2" "libc.so.6" "libgpm.so.2" "libacl.so.1.1.2301" "libsodium.so.23.3.0" "libselinux.so.1" "libtinfo.so.6.4" "libm.so.6" "ld-linux-x86-64.so.2"])
- instant messaging :)
- 2026/01/28 20:30:25 failed to accept client - failed to create new session - session id already in use (":1")

## diff

- diff the output of 2 commands

## multi session support

- more than one grumble shell interacting with each other
- grumble exits process before cleanup code can run (example: socket file not being removed)
- fix history file creation location
- figure out better way of doing copyAndAddBackslashRLoop()

## plugin

- support for custom plugin commands
  - ex. lineup - makes all enemies lineup in front of player
  - ex. coords - prints x,y,z coords of all enemies
- add context as arguments to plugin.load and plugin.unload
- investigate order of unload and load events being out of order
- improve plugin user experience when unloading / reloading
  (i.e., be able to load a previously-unloaded plugin using
  only its name - maybe we can have a name -> plugin info
  cache?)

### mskit

- helper function for reading a pointer from process
- helper function for reading data from process using a pointer to a Vec<u8>
- investigate rust macro to generate ffi functions

## parser

~~- check if we are attached to a process before running a parser~~
~~- fix assumption of user supplying absolute address in parser~~

## vmmap

~~- `vmmap object_Name` shows object with that name and regions under it~~
~~- code needs to be cleaned up~~
- fix the permissions are not showing up correctly

## progctl

~~- when MappedObjects is called go and actually ask windows~~
- Support for exitMonitor on Unix-like systems
- Need to implement Suspend and Resume methods for WindowsProcess

## memory

~~- fix the MappedObjects to be a slice instead of map, handle duplicate dlls~~
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
- detach command
- pid in prompt when attached

## find

- "*" support for super wildcard pattern search, maybe not at the end
- add configurable logging for when error occurs
- improve find performance (increase size read, or with start/end address)

## kernel32

- make Read/Write process memory behave like io interfaces, return the number of bytes read/written

## events

- move event and grsh into app or session and implement state for those events
- consider making log messages available to events to logging logic

## shell

- reverse search (ctrl+r) and selecting an entry does not save to history
- disable color (needs to be a Session-level setting)

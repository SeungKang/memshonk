# TODO

## next stopping point

- command output support (access the result of previous commands)
- allow overwriting of executable mapped object name
  - error: failed to attach to process 220785 ("vim") - attach failure - failed to get mapped object for exe - failed to find a match for an object named: "vim" (searched through: ["vim.basic" "locale-archive" "libpthread.so.0" "libpcre2-8.so.0.11.2" "libc.so.6" "libgpm.so.2" "libacl.so.1.1.2301" "libsodium.so.23.3.0" "libselinux.so.1" "libtinfo.so.6.4" "libm.so.6" "ld-linux-x86-64.so.2"])
- processWriter needs to offset itself automatically like os.File
- Provide short usage explanation (how project files work w/ daemon)

## documentation

- export in-app documentation to Markdown files
- add documentation for procfs mode

## diff

- diff the output of 2 commands

## multi session support

- instant messaging :)

## plugin

- add context as arguments to plugin.load and plugin.unload
- improve plugin user experience when unloading / reloading
  (i.e., be able to load a previously-unloaded plugin using
  only its name - maybe we can have a name -> plugin info
  cache?)
- add ResolvePointer function (i.e., allow plugins to resolve pointer strings)

## mskit

- helper function for reading a pointer from process
- helper function for reading data from process using a pointer to a `Vec<u8>`
- investigate rust macro to generate ffi functions

## progctl

- Need to implement Suspend and Resume methods for Windows
- Search for process using its path (we are currently limited to
  searching by its name)
- Switch `Read*` methods to behave more like io.Reader (pass in
  a []byte to read data into, rather than allocating a new one)

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
- From [Amy](https://github.com/tobert/): memory-map a process' region / object
  to a file so things outside of memshonk can play with it

## readm command

- Get input data from a file / pipe

## scan

- "*" support for super wildcard pattern search, maybe not at the end
- add configurable logging for when error occurs
- pattern: add ability to specify n `??` (like: `??x10`)

## kernel32

- make Read/Write process memory behave like io interfaces, return the number of bytes read/written

## events

- consider making log messages available to events to logging logic

## shell

- support command output piping / referencing (access results of previous commands)
- memshonk exec mode to run a memshonk command (allow you to run memshonk from shell script)
- test scripting within memshonk shell
- need to consider tab completion for external program
- disable color (needs to be a Session-level setting)
- allow execution of external programs to be disabled
- support stdin reading for each command including supporting io.closer for cancellation
- tab completion should add a space (" ") character if it completes
  and there are no other options (e.g., tab completing "foob" when
  nothing else matches should become "foobar " and not "foobar")

## projects

- Create project file based on current settings (i.e., serialize current
  project.Project object to ini)

## limitations

- No breakpoints or disassembly (we would like to implement this though!
  Refer to the `dissect` branch for our initial work towards that)

## hexdump

- Switch hexdump styles via shonkset and/or main configuration file

## known issues (bugs)

- intermittent hanging at "connecting to daemon..." when running:

  ```
  winpty go run -tags plugins_execonreload main.go -p examples/mass-effect-3.txt
  ```

- Handling of terminal cursor when it wraps on to the next line (there
  needs to be ~half the terminal rows filled for this to happen)
- Shell reverse search (ctrl+r) does not save the selected item to shell
  history (this is likely a limitation of the `readline` library)
- Interactive programs like `vim` and `less` do not currently work (this
  may be a simple fix, but we have not looked into it deeply as of yet)

## hacks / workarounds

- Try to remove Unix socket startup workaround in main.go
  - Everything but color works :/
- figure out better way of doing copyAndAddBackslashRLoop() (adding `\r\n`)

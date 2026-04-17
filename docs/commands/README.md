
# Commands

This is a placeholder for now.

Here is what the `help` command provides:

```
OVERVIEW
  memshonk is an experimental command-line debugger companion that tries to
  fill the functionality gaps between debuggers. Think of it as a cross
  between gdb, rizin, and Cheat Engine. It is not meant to replace
  a debugger, but supplement it.

TOPICS
  address   - How memshonk handles memory addresses (pointer chains)
  datatypes - Data types usable with various memory manipulation commands
  formats   - Supported data formatting (encoding) options
  pattern   - Pattern string format used in the scan command and potentially
              other commands

COMMANDS
  attach   - attach to the process
  daemon   - manage the server daemon
  detach   - detach from the process
  help     - list available commands and help topics
  jobs     - manage background jobs
  mrun     - run a memshonk script
  plugin   - manage plugins
  prev     - list and retrieve outputs of previously-run commands
  quit     - exit the current memshonk session
  readm    - read data from process memory
  scan     - search process memory for values or byte patterns
  session  - manage session
  shonkset - set configuration options
  vmmap    - view the process's memory regions
  watch    - watch data at an address for changes
  writem   - write data to process memory
```

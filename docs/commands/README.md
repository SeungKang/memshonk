
# Commands

This is a placeholder for now.

Here is what the `help` command provides:

```
OVERVIEW
memshonk is like Wireshark, but for process memory. It provides
an interactive shell that supports POSIX shell syntax, pipes,
job control, and execution of external programs and internal
memshonk commands.

TOPICS
datatypes - Data types usable with various memory manipulation commands
formats   - Supported data formatting (encoding) options
pattern   - Pattern string format used in the "find" command and
potentially other commands

COMMANDS
attach   - attach to the process
daemon   - manage the server daemon
detach   - detach from the process
find     - find data in a process' memory
help     - list available commands and help topics
jobs     - manage background jobs
mrun     - run a memshonk script
plugins  - manage plugins
quit     - exit the current memshonk session
readm    - read data from process memory
session  - manage session
shonkset - set configuration options
vmmap    - view the process's memory regions
watch    - watch data at an address for changes
writem   - write data to process memory
```
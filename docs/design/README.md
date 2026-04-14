# Design / architecture

This document discusses topics related to memshonk's design patterns
and architecture.

## Security

tl;dr - Only use plugins if you absolutely trust the source of the plugin
code. Do not copy paste random code from the Internet into memshonk or
execute memshonk scripts that you do not understand. memshonk is an
inherently privileged program with limited security controls.

Security is, unfortunately, something that we did not focus on very heavily
initially due to not having a clear picture of memshonk's architecture from
the very start of development. In retrospect, we should have put more thought
into it early on, which would have made it easier to implement later. That
said, such work is very time intensive and we have spent a year working on
memshonk as it is. Weighing that against the diminishing returns of solving
hard security problems is not something we wish on anyone.

### Client authentication

memshonk creates a Unix socket in `~/.memshonk` that exposes its API and
debugging functionality to clients. Currently, the socket does not
authenticate clients. While adding authentication will make the user
experience around creating additional sessions more maclunky, it is
something we would like to implement.

One approach we considered was to use mutual TLS authentication, similar to
what Stephen did with [`raygun`](https://github.com/stephen-fox/raygun).
We would implement a command that creates a JSON authentication blob which
contains a private key and certificate (or maybe even just a private key)
that the user would pass to memshonk via stdin. TLS over Unix sockets is
definitely possible and, while it sounds like overkill, protects against
the server socket being replaced by a hostile program.

### Sandboxing (the lack thereof)

Another security property we considered is sandboxing. This is particularly
difficult because:

- A debugger is an inherently privileged program since it can read and
  write arbitrary memory in other processes (plus some operating systems
  require the debugger to run as the `root` user anyways)
- Process isolation mechanisms likely prohibit the use of memory-manipulation
  operating system APIs required by memshonk
- Sandboxing is antithetical to providing a shell (i.e., the purpose
  of a shell *is* to provide arbitrary code execution)

In theory, memshonk can be separated into different processes to work
around the first two limitations. In other words: we would likely need
to separate memshonk into the following processes:

- `debugger daemon`: A privileged, unsandboxed process doing the debugging
  work which enforces some kind of policy about which processes it attaches
  to (i.e., refuses to attach to itself and checks process names to ensure
  they match the program targeted by the project file - the latter of which
  has its own PITA implications)
- `session daemon`: A sandboxed helper process responsible for running
  plugins and the shell logic for user sessions
- `client`: The existing memshonk process that connects to the session
  daemon. This process would be easy to sandbox, since it is functionally
  a "TV set"... That is, assuming we would not just change how everything
  works

As for dealing with a shell... yeah, idk. FreeBSD jails and Linux containers
are well suited for solving that problem, but other operating systems such
as OpenBSD and Windows have no equivalent functionality (well, OpenBSD gets
kind of close with `unveil` - but it is currently super broken because it
does not persist from parent to child processes). In any case - these
facilities bring their own demons and increase the complexity on memshonk.

The goal of sandboxing memshonk would be to protect against hostile scripts,
plugins, and code copy-pasta'd from stackoverflow.

The ultimate, unsolvable problem with this goal is that we cannot prevent
hostile plugins or scripts from reading and writing memory of the process
being debugged. It is safe to assume that the debugged process is itself
not sandboxed, so it will always present an easy sandbox escape.

Now, if memshonk develops its own disassembler (i.e., is no longer a pure
debugger / dynamic analysis tool), this problem changes quite a bit since
the debugger code may not be needed if the user is doing only static
analysis work. While we have started that work, it is nowhere near being
usable... which brings us back to the time trade off of doing this work.

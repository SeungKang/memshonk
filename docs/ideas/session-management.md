# Session management

We have the foundation in place for session support. However, we are missing
the high-level functionality that allows users to view and manage sessions.
This document attempts to capture some initial thoughts on what session
management looks like and how it may be implemented. Each section in this
document represents a new feature we need to implement.

## Shell command

We need a command for managing sessions (e.g., `sessions VERB [ARGS...]`).
I am not attached to calling it `sessions`. Maybe we can call it `w` or `W`
after the [`w` program](https://man.openbsd.org/w) - though using lowercase
`w` would require removing the `write` command's `w` alias. Side note: I am
partial to using a capital letter if we use `W` because of the power that
comes with the command - but we can always change this detail later.

This new command will have the following subcommands:

- `id` - Lists information about the current session
- `ls [SESSION-ID...]` - Lists all sessions if SESSION-ID was not specified.
  Alternatively, lists one or more sessions with additional details for each
  specified SESSION-ID
- `rm SESSION-ID...` - Disconnects and removes one or more sessions for each
  SESSION-ID specified

To create this command, we can make a copy of `internal/commands/plugins.go`
(which also uses subcommands) and call the new file `sessions.go` or whatever
we end up calling the command. After renaming the various references from
`Plugins` in the file, we will need to add the command's `Schema` function
to the `BuiltinCommands` function in `common.go`.

The subcommands can use the following objects from the second argument of the
command's `Run` method (assuming `Run`'s second argument is named `s`):

- `s.Info()` (returns a recently-added `SessionInfo object`) - For `id`
- `s.SharedState().Sessions` (returns a recently-added `SessionManager`
  object) - For `ls` and `rm`

For the output of the `id` and `ls` commands, we should consider adding
a `String` method to the `apicompat.SessionInfo` type which returns a pretty,
human-readable string describing the session's info. Having that logic in
one place guarantees consistent formatting when we want to show session info
to users.

## UI changes

We should provide a UI element that tells the user their session ID. This can
be part of the shell prompt for now. Maybe in the future we will have something
similar to tmux's status line (the thing at the bottom of a tmux window).

I have no super-specific thoughts on how it would appear in the prompt,
maybe something like this:

```
# Not attached:
(SESSION-ID) $

# Attached:
(SESSION-ID) [PID] $
```

Perhaps the `default` session can just be a special character or we just
do not show it at all. The shell prompt update code currently resides in
`internal/grsh/grsh.go` in the `Shell.setPrompt` method. That method can
use the `Shell.session` field to get the session's ID.

## Session ID customization via command-line argument

We need a way to customize the session ID using a command-line argument.
This will allow users to add some context to their sessions. This new
functionality would only apply to the client mode, but due to the way
the code is structured in `main.go`, it makes sense to add it to the
`mainWithError` function.

We would start by adding a `flag.String` call after the `help` variable in
`mainWithError`. The argument letter can be `S` (for `-S`), but I am not
attached to that. We can pass the flag's value to the `doClient` function
which can then pass the session ID to the `sessiond.NewClient` using the
`sessiond.ClientConfig` struct. We will need to add a new field to that
struct named `OptSessionID`.

The `sessiond.NewClient` function will pass the previously-mentioned field's
value to the line that creates the `apiConn` variable. The second argument of
the `cm.DialContext` is passed through the ConnMux to the `Server` which uses
that string as the session ID if it is non-empty.

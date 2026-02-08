# Add string-based `*Lookup` methods to `progctl.Process`

In the current implementation of `progctl.Process`, its methods expect memory
addresses to be represented using a memory.Pointer type. While this approach
works, it presents some maclunkiness:

- All the various command implementations must convert the user's pointer
  string to a memory.Pointer before calling the desired Process method.
  This leads to a lot of boiler plate code being copy-pasted throughout
  the code base
- We limit the user's command input to only a Pointer string. This explicit
  limitation keeps us from supporting other potential lookup types (like
  symbol or flag lookups)
- We never considered the possibility of allowing plugins to specify
  a memory.Pointer string. As a result, plugins currently only expose
  a uint-based API to read and write memory via the wrapper code in
  `internal/plugins/pluginscompat/compat.go`. It would be really nice
  to just pass a string from the plugin to the Process API because
  doing so would allow plugins to automatically inherit any newly
  added lookup functionality (such as the examples described above)

We should consider adding new methods to the progctl.Process that accept
a `string` instead of a `memory.Pointer` and have the Process code decide
how to handle the string. That way, the string handling code is kept in
one place. We can leave the existing memory.Pointer-based methods for
code that needs it (like the previously-mentioned plugin code).

We will end up with two categories of methods that contain the following
words:

- `*Addr` - for memory.Pointer-based APIs
- `*Lookup` - for string-based APIs

## What needs to be done

1. Add the following methods to the progctl.Process interface that accept
   strings instead of memory.Pointer:
   - `ReadFromLookup` (equivalent to ReadFromAddr)
   - `WriteToLookup` (equivalent to WriteToAddr)
   - `WatchLookup` (equivalent to Watch)
2. Rename the existing `Watch` method to `WatchAddr` in both progctl.Process
   and in progctl.Ctl
3. In `progctl.Ctl`, move the code for the following methods (except for the
   mutex locking code) into new private methods that have the same name, just
   with a lower-cased first letter:
   - `ReadFromAddr`
   - `WriteToAddr`
   - `WatchAddr`
4. Implement the new progctl.Process methods in progctl.Ctl. These methods
   will resolve the pointers using the private `Ctl.resolvePointer` method
   (make sure this happens *after* the existing mutex locking code) and
   call the private versions of the `*Addr` methods
5. If we did everything correctly, then memshonk should compile+run
6. In `internal/commands/`, find all the usages of the memory.Pointer-based
   APIs and switch them to use the new `*Lookup` APIs
7. Repeat step 5 :)

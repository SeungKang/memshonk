# Plugins

memshonk supports plugins through shared libraries, including:

- Shared objects (`.so` files) on Unix-like operating systems
- Dynamic link libraries (`.dll` files) on Windows

Plugins can be loaded and unloaded at runtime using the `plugin` command,
though plugins must explicitly declare support for being unloaded. memshonk
expects plugins to implement a handful of functions.

## Functionality

memshonk exposes the following functionality to a plugin:

- Read memory from an address in the attached process
- Write memory to an address in the attached process
- Create a memshonk command
- Create a "parser" function which is essentially a command that parses
  an object at an optionally-supplied address and returns a blob of JSON
  representing it

## Plugin API

We strongly recommend writing your plugin in [Rust](https://rust-lang.org/)
which will allow you to use the [`mskit`](../../plugin-api/mskit) library
as a foundation for your plugin. The mskit library implements several
functions and date types required by memshonk.

If you would like to implement your own variant of mskit in Rust or another
language, we recommend using mskit as a design reference.

### Required plugin API functions

memshonk requires plugins to implement a handful of building blocks.
`mskit` implements these functions for you, but if you would like to
do it yourself, please use the following source files as a reference:

- `plugin-api/mskit/src/lib.rs` - Plugin-side implementation reference
- `internal/plugins/libplugin/libctl.go` - Refer to the constants at the
  top of the file for a list of library functions that memshonk looks for

### Unloading

To support unloading, a plugin must:

- Export a function named `unload` which takes no arguments and returns no
  values. This function *must* ensure any threads started by the plugin
  are stopped

### Versioning

Plugins can specify a version by:

- Exporting a function named `version` which takes no arguments and returns
  an unsigned 32-bit integer representing the version number where each
  8-bits represents a field in [semantic versioning](https://semver.org/).
  Each 8 bits convey the following information:
  - First 8-bits: Major version
  - Second 8-bits: Minor version
  - Third 8-bits: Patch version
  - Fourth 8-bits: Reserved (truthfully, idk)

### Commands

To implement a custom command, export a function whose name is suffixed
with `_mscmd`. memshonk will discover the command by searching through
the library's exported symbols.

The function's signature must be:

```rust
#[no_mangle]
extern "C" fn example_mscmd(ctx: *mut Ctx, args: *mut u8, output_ptr: *mut *mut u8) -> *mut u8
```

Arguments:

- `ctx` is a `mskit::Ctx` object pointer. Interaction with this object is not
  required. If you do not plan to use it, you can use `_: usize` in its place
- `args` is a pointer to a `mskit::SharedBuf` containing a list of
  null-terminated argument strings passed to the command by the user
- `output_ptr` is a double pointer that the function should update to point
  at a `mskit::SharedBuf` created by this function which contains the output
  data produced by it

Return value:

The return value is used to indicate success or failure. If the function
failed, it should be a pointer to a `mskit::SharedBuf` containing an error
message. If the function succeeded, then a null pointer should be returned.

### Parsers

To implement a parser command, export a function whose name is suffixed
with `_mspar`. memshonk will discover the parser by searching through
the library's exported symbols.

The function's signature must be:

```rust
#[no_mangle]
extern "C" example_mspar(ctx: *mut Ctx, addr: usize, str_ptr: *mut *mut u8) -> *mut u8
```

Arguments:

- `ctx` is a `mskit::Ctx` object pointer. Interaction with this object is not
  required. If you do not plan to use it, you can use `_: usize` in its place
- `addr` is an optional address to read from (a value of `0` indicates
  the user did not pass an address to the parser command)
- `str_ptr` is double pointer that the found should update to point at
  a `mskit::SharedBuf` containing a JSON blob representing the object
  it parsed

Return value:

The return value is used to indicate success or failure. If the function
failed, it should be a pointer to a `mskit::SharedBuf` containing an error
message. If the function succeeded, then a null pointer should be returned.

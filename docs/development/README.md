# Development

This document provides guidance about developing memshonk.

## Building from source

*Warning*: Due to memshonk using a client-daemon architecture, it is highly
recommended to avoid using `go run` to execute memshonk. While using `go run`
*should* work - it adds another layer of complexity to troubleshooting issues
when they occur.

The following subsections document how to build memshonk from source, each
of which will produce an executable named `memshonk` (or `memshonk.exe` on
Windows) in the top of the repo.

### Default build

```sh
# Note: Make sure to "cd" to the top of the repo first.
go build
```

### Build with plugin "exec on reload" enabled

```sh
go build -tags plugins_execonreload
```

### Build with plugins disabled

```sh
go build -tags plugins_disabled
```

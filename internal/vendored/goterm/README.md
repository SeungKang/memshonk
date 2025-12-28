# goterm

This library is a fork of [buger's goterm][upstream] which provides basic
building blocks for building advanced console UIs.

Initially created for [Gor][gor].

[upstream]: https://github.com/buger/goterm
[gor]: https://github.com/buger/gor

## Features

- Terminal manipulation (clearing the screen, virtual cursor, colors)
- Support for virtual terminals (terminals not backed by file descriptors)
- Various UI elements, including: boxes, line charts, and tables

## Basic usage

Full screen console app, printing current time:

```go
func mainWithError() error {
    tm, err := goterm.NewStdioScreen()
    if err != nil {
        return err
    }

    tm.Clear() // Clear current screen

    for {
        // By moving cursor to top-left position we ensure that console output
        // will be overwritten each time, instead of adding new.
        tm.MoveCursor(1, 1)

        tm.Println("Current Time:", time.Now().Format(time.RFC1123))

        tm.Flush() // Call it every time at the end of rendering

        time.Sleep(time.Second)
    }
}
```

## Examples

Please refer to the [`examples/` directory](examples/). Each example can
be run by executing:

```sh
go run examples/EXAMPLE-NAME/main.go
```

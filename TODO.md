# TODO

## next stopping point

~~- unix support~~
- command output support (access the result of previous commands)
- plugin command for me3
- don't make ctrl + c exit
- retain shell command history

## diff

- diff the output of 2 commands

## multi session support

- more than one grumble shell interacting with each other

## plugin

- support for custom plugin commands
  - ex. lineup - makes all enemies lineup in front of player
  - ex. coords - prints x,y,z coords of all enemies
- me3 plugin finds all enemy structs

## parser
- check if we are attached to a process before running a parser
- fix assumption of user supplying absolute address in parser

## vmmap

~~- `vmmap object_Name` shows object with that name and regions under it~~
~~- code needs to be cleaned up~~

## progctl

~~- when MappedObjects is called go and actually ask windows~~
- Support for exitMonitor on Unix-like systems
- Need to implement Suspend and Resume methods for WindowsProcess

## memory

~~- fix the MappedObjects to be a slice instead of map, handle duplicate dlls~~

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

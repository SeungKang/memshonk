package linuxmaps

import (
	"fmt"
	"strings"
	"testing"
)

var LinuxMap = `563736a76000-563736aa5000 r--p 00000000 fe:02 2360744                    /usr/bin/bash
563736aa5000-563736b66000 r-xp 0002f000 fe:02 2360744                    /usr/bin/bash
563736b66000-563736b9e000 r--p 000f0000 fe:02 2360744                    /usr/bin/bash
563736b9e000-563736ba2000 r--p 00128000 fe:02 2360744                    /usr/bin/bash
563736ba2000-563736bab000 rw-p 0012c000 fe:02 2360744                    /usr/bin/bash
563736bab000-563736bb6000 rw-p 00000000 00:00 0
56373c05a000-56373c206000 rw-p 00000000 00:00 0                          [heap]
7f837d600000-7f837d8e9000 r--p 00000000 fe:02 2360155                    /usr/lib/locale/locale-archive
7f837d983000-7f837d993000 r--p 00000000 fe:02 2359318                    /usr/lib/x86_64-linux-gnu/libm.so.6
7f837d993000-7f837da06000 r-xp 00010000 fe:02 2359318                    /usr/lib/x86_64-linux-gnu/libm.so.6
7f837da06000-7f837da60000 r--p 00083000 fe:02 2359318                    /usr/lib/x86_64-linux-gnu/libm.so.6
7f837da60000-7f837da61000 r--p 000dc000 fe:02 2359318                    /usr/lib/x86_64-linux-gnu/libm.so.6
7f837da61000-7f837da62000 rw-p 000dd000 fe:02 2359318                    /usr/lib/x86_64-linux-gnu/libm.so.6
7f837da62000-7f837da65000 r--p 00000000 fe:02 2359716                    /usr/lib/x86_64-linux-gnu/libcap.so.2.66
7f837da65000-7f837da6a000 r-xp 00003000 fe:02 2359716                    /usr/lib/x86_64-linux-gnu/libcap.so.2.66
7f837da6a000-7f837da6c000 r--p 00008000 fe:02 2359716                    /usr/lib/x86_64-linux-gnu/libcap.so.2.66
7f837da6c000-7f837da6d000 r--p 0000a000 fe:02 2359716                    /usr/lib/x86_64-linux-gnu/libcap.so.2.66
7f837da6d000-7f837da6e000 rw-p 0000b000 fe:02 2359716                    /usr/lib/x86_64-linux-gnu/libcap.so.2.66
7f837da6e000-7f837da75000 r--p 00000000 fe:02 2385283                    /usr/lib/x86_64-linux-gnu/libnss_systemd.so.2
7f837da75000-7f837daa9000 r-xp 00007000 fe:02 2385283                    /usr/lib/x86_64-linux-gnu/libnss_systemd.so.2
7f837daa9000-7f837dab9000 r--p 0003b000 fe:02 2385283                    /usr/lib/x86_64-linux-gnu/libnss_systemd.so.2
7f837dab9000-7f837dabd000 r--p 0004b000 fe:02 2385283                    /usr/lib/x86_64-linux-gnu/libnss_systemd.so.2
7f837dabd000-7f837dabe000 rw-p 0004f000 fe:02 2385283                    /usr/lib/x86_64-linux-gnu/libnss_systemd.so.2
7f837dac5000-7f837dac8000 rw-p 00000000 00:00 0
7f837dac8000-7f837daee000 r--p 00000000 fe:02 2359315                    /usr/lib/x86_64-linux-gnu/libc.so.6
7f837daee000-7f837dc43000 r-xp 00026000 fe:02 2359315                    /usr/lib/x86_64-linux-gnu/libc.so.6
7f837dc43000-7f837dc96000 r--p 0017b000 fe:02 2359315                    /usr/lib/x86_64-linux-gnu/libc.so.6
7f837dc96000-7f837dc9a000 r--p 001ce000 fe:02 2359315                    /usr/lib/x86_64-linux-gnu/libc.so.6
7f837dc9a000-7f837dc9c000 rw-p 001d2000 fe:02 2359315                    /usr/lib/x86_64-linux-gnu/libc.so.6
7f837dc9c000-7f837dca9000 rw-p 00000000 00:00 0
7f837dca9000-7f837dcb8000 r--p 00000000 fe:02 2362637                    /usr/lib/x86_64-linux-gnu/libtinfo.so.6.4
7f837dcb8000-7f837dcc9000 r-xp 0000f000 fe:02 2362637                    /usr/lib/x86_64-linux-gnu/libtinfo.so.6.4
7f837dcc9000-7f837dcd7000 r--p 00020000 fe:02 2362637                    /usr/lib/x86_64-linux-gnu/libtinfo.so.6.4
7f837dcd7000-7f837dcdb000 r--p 0002d000 fe:02 2362637                    /usr/lib/x86_64-linux-gnu/libtinfo.so.6.4
7f837dcdb000-7f837dcdc000 rw-p 00031000 fe:02 2362637                    /usr/lib/x86_64-linux-gnu/libtinfo.so.6.4
7f837dcdc000-7f837dce3000 r--s 00000000 fe:02 2362243                    /usr/lib/x86_64-linux-gnu/gconv/gconv-modules.cache
7f837dce3000-7f837dce5000 rw-p 00000000 00:00 0
7f837dce5000-7f837dce6000 r--p 00000000 fe:02 2359311                    /usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2
7f837dce6000-7f837dd0b000 r-xp 00001000 fe:02 2359311                    /usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2
7f837dd0b000-7f837dd15000 r--p 00026000 fe:02 2359311                    /usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2
7f837dd15000-7f837dd17000 r--p 00030000 fe:02 2359311                    /usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2
7f837dd17000-7f837dd19000 rw-p 00032000 fe:02 2359311                    /usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2
7ffd94dba000-7ffd94ddb000 rw-p 00000000 00:00 0                          [stack]
7ffd94de4000-7ffd94de8000 r--p 00000000 00:00 0                          [vvar]
7ffd94de8000-7ffd94dea000 r-xp 00000000 00:00 0                          [vdso]`

func TestVmmapParser(t *testing.T) {
	reader := strings.NewReader(LinuxMap)

	regions, err := vmmap(reader)
	if err != nil {
		t.Fatalf("failed vmmap - %v", err)
	}

	fmt.Printf("%+v\n", regions)

	if regions.Len() != 45 {
		t.Fatalf("expected 45 regions, got %d", regions.Len())
	}
}

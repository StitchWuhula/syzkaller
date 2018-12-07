// Copyright 2018 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package proggen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/syzkaller/prog"
	_ "github.com/google/syzkaller/sys"
	"github.com/google/syzkaller/tools/syz-trace2syz/parser"
)

func TestParse(t *testing.T) {
	type Test struct {
		input  string
		output string
	}
	tests := []Test{
		{`
open("file", 66) = 3
write(3, "somedata", 8) = 8
`, `
r0 = open(&(0x7f0000000000)='file\x00', 0x42, 0x0)
write(r0, &(0x7f0000000040)='somedata\x00', 0x9)
`,
		}, {`
pipe([5,6]) = 0
write(6, "\xff\xff\xfe\xff", 4) = 4
`, `
pipe(&(0x7f0000000000)={0xffffffffffffffff, <r0=>0xffffffffffffffff})
write(r0, &(0x7f0000000040)="fffffeff00", 0x5)
`,
		}, {`
pipe({0x0, 0x1}) = 0
shmget(0x0, 0x1, 0x2, 0x3) = 0
`, `
pipe(&(0x7f0000000000))
shmget(0x0, 0x1, 0x2, &(0x7f0000001000/0x1)=nil)
`,
		}, {`
socket(29, 3, 1) = 3
getsockopt(-1, 132, 119, 0x200005c0, [14]) = -1 EBADF (Bad file descriptor)
`, `
socket$can_raw(0x1d, 0x3, 0x1)
getsockopt$inet_sctp6_SCTP_RESET_STREAMS(0xffffffffffffffff, 0x84, 0x77, &(0x7f0000000000), &(0x7f0000000040)=0x8)
`,
		}, {`
inotify_init() = 2
open("tmp", 66) = 3
inotify_add_watch(3, "\x2e", 0xfff) = 3
write(3, "temp", 5) = 5
inotify_rm_watch(2, 3) = 0
`, `
r0 = inotify_init()
r1 = open(&(0x7f0000000000)='tmp\x00', 0x42, 0x0)
r2 = inotify_add_watch(r1, &(0x7f0000000040)='.\x00', 0xfff)
write(r1, &(0x7f0000000080)='temp\x00', 0x5)
inotify_rm_watch(r0, r2)
`,
		}, {`
socket(1, 1, 0) = 3
socket(1, 1 | 2048, 0) = 3
socket(1, 1 | 524288, 0) = 3
socket(1, 1 | 524288, 0) = 3
`, `
socket$unix(0x1, 0x1, 0x0)
socket$unix(0x1, 0x801, 0x0)
socket$unix(0x1, 0x80001, 0x0)
socket$unix(0x1, 0x80001, 0x0)
`,
		}, {`
open("temp", 1) = 3
connect(3, {sa_family=2, sin_port=37957, sin_addr=0x0}, 16) = -1
`, `
r0 = open(&(0x7f0000000000)='temp\x00', 0x1, 0x0)
connect(r0, &(0x7f0000000040)=@in={0x2, 0x9445}, 0x80)
`,
		}, {`
open("temp", 1) = 3
connect(3, {sa_family=1, sun_path="temp"}, 110) = -1
`, `
r0 = open(&(0x7f0000000000)='temp\x00', 0x1, 0x0)
connect(r0, &(0x7f0000000040)=@un=@file={0x1, 'temp\x00'}, 0x80)
`,
		}, {`
open("temp", 1) = 3
bind(5, {sa_family=16, nl_pid=0x2, nl_groups=00000003}, 12)  = -1
`, `
open(&(0x7f0000000000)='temp\x00', 0x1, 0x0)
bind(0x5, &(0x7f0000000040)=@nl=@proc={0x10, 0x2, 0x3}, 0x80)
`,
		}, {`
socket(17, 3, 768)  = 3
ioctl(3, 35111, {ifr_name="\x6c\x6f", ifr_hwaddr=00:00:00:00:00:00}) = 0
`, `
r0 = socket$packet(0x11, 0x3, 0x300)
ioctl$sock_ifreq(r0, 0x8927, &(0x7f0000000000)={'lo\x00'})
`,
		}, {`
socket(1, 1, 0) = 3
connect(3, {sa_family=1, sun_path="temp"}, 110) = -1 ENOENT (Bad file descriptor)
`, `
r0 = socket$unix(0x1, 0x1, 0x0)
connect$unix(r0, &(0x7f0000000000)=@file={0x1, 'temp\x00'}, 0x6e)
`,
		}, {`
socket(1, 1, 0) = 3
`, `
socket$unix(0x1, 0x1, 0x0)
`,
		}, {`
socket(2, 1, 0) = 5
ioctl(5, 21537, [1]) = 0
`, `
r0 = socket$inet_tcp(0x2, 0x1, 0x0)
ioctl$int_in(r0, 0x5421, &(0x7f0000000000)=0x1)
`,
		}, {`
socket(2, 1, 0) = 3
setsockopt(3, 1, 2, [1], 4) = 0
`, `
r0 = socket$inet_tcp(0x2, 0x1, 0x0)
setsockopt$sock_int(r0, 0x1, 0x2, &(0x7f0000000000)=0x1, 0x4)
`,
		}, {`
9795  socket(17, 3, 768)  = 3
9795  ioctl(3, 35123, {ifr_name="\x6c\x6f", }) = 0
`, `
r0 = socket$packet(0x11, 0x3, 0x300)
ioctl$ifreq_SIOCGIFINDEX_team(r0, 0x8933, &(0x7f0000000000)={'lo\x00'})
`,
		}, {`
open("temp", 1) = 3
connect(3, {sa_family=2, sin_port=17812, sin_addr=0x0}, 16) = -1
`, `
r0 = open(&(0x7f0000000000)='temp\x00', 0x1, 0x0)
connect(r0, &(0x7f0000000040)=@in={0x2, 0x4594}, 0x80)
`,
		}, {`
ioprio_get(1, 0) = 4
`, `
ioprio_get$pid(0x1, 0x0)
`,
		}, {`
socket(17, 2, 768) = 3
`, `
socket$packet(0x11, 0x2, 0x300)
`,
		}, {`
socket(2, 1, 0) = 3
connect(3, {sa_family=2, sin_port=17812, sin_addr=0x0}, 16) = 0
`, `
r0 = socket$inet_tcp(0x2, 0x1, 0x0)
connect$inet(r0, &(0x7f0000000000)={0x2, 0x4594}, 0x10)
`,
		}, {`
socket(2, 1, 0) = 3
connect(3, {sa_family=2, sin_port=17812, sin_addr=0x7f000001}, 16) = 0
`, `
r0 = socket$inet_tcp(0x2, 0x1, 0x0)
connect$inet(r0, &(0x7f0000000000)={0x2, 0x4594, @rand_addr=0x7f000001}, 0x10)
`,
		},
	}
	target, err := prog.GetTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	target.ConstMap = make(map[string]uint64)
	for _, c := range target.Consts {
		target.ConstMap[c.Name] = c.Value
	}
	for _, test := range tests {
		input := strings.TrimSpace(test.input)
		tree, err := parser.ParseData([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		p := genProg(tree.TraceMap[tree.RootPid], target)
		if p == nil {
			t.Fatalf("failed to parse trace")
		}
		got := string(bytes.TrimSpace(p.Serialize()))
		want := strings.TrimSpace(test.output)
		if want != got {
			t.Errorf("input:\n%v\n\nwant:\n%v\n\ngot:\n%v", input, want, got)
		}
	}
}

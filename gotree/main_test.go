package main

import (
	"bytes"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		opt  *option
		err  bool
	}{
		{name: "basic", opt: &option{root: "t"}},
		{name: "ignore", opt: &option{root: "t", ignore: "dir3"}},
		{name: "total", opt: &option{root: "t", total: true}},
		{name: "version", opt: &option{root: "t", version: true}},
		{name: "fullpath", opt: &option{root: "t", full: true}},
		{name: "dirs", opt: &option{root: "t", dirs: true}},
		{name: "verbose", opt: &option{root: "t", verbose: true}},

		{name: "invalid root", opt: &option{root: "t/invalid"}, err: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("opt: %#v", test.opt)
			buf := bytes.NewBuffer([]byte{})
			errbuf := bytes.NewBuffer([]byte{})
			exit := run(buf, errbuf, test.opt)
			if test.err && exit == 0 {
				t.Error("expected exit code != 0, but return 0")
			}
			t.Logf("stdout:%v", buf)
			t.Logf("stderr:%v", errbuf)
			t.Logf("exit code:%v", exit)
		})
	}
}

package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	const dir = "testdata"
	tests := []struct {
		name string
		opt  *option
		err  bool
	}{
		{name: "basic", opt: &option{root: dir}},
		{name: "ignore", opt: &option{root: dir, ignore: "dir3"}},
		{name: "total", opt: &option{root: dir, total: true}},
		{name: "version", opt: &option{root: dir, version: true}},
		{name: "fullpath", opt: &option{root: dir, full: true}},
		{name: "dirs", opt: &option{root: dir, dirs: true}},
		{name: "verbose", opt: &option{root: dir, verbose: true}},

		{name: "invalid root", opt: &option{root: filepath.Join(dir, "invalid")}, err: true},
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

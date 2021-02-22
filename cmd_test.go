// Copyright (c) 2019-2020 Siemens AG
// Copyright (c) 2019-2021 Jonas Plum
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
//
// Author(s): Jonas Plum

package fscmd

import (
	"bytes"
	"github.com/spf13/cobra"
	"io"
	"io/fs"
	"os"
	"reflect"
	"regexp"
	"testing"
	"testing/fstest"
)

func stdout(f func()) []byte {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	outC := make(chan []byte)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r) // nolint
		outC <- buf.Bytes()
	}()

	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real stdout
	return <-outC
}

var testFS = &fstest.MapFS{
	"foo":        &fstest.MapFile{Data: []byte("foo")},
	"folder/bar": &fstest.MapFile{Data: []byte("bar")},
}

func defaultParse(_ *cobra.Command, args []string) (fs.FS, []string, error) {
	return testFS, args, nil
}

func Test_cat(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name     string
		args     args
		wantData []byte
	}{
		{"cat", args{"foo"}, []byte("foo")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData := stdout(func() { CatCmd(defaultParse)(nil, []string{tt.args.url}) })

			re := regexp.MustCompile(`\r?\n`) // TODO: improve newline handling
			gotDataString := re.ReplaceAllString(string(gotData), "")
			wantData := re.ReplaceAllString(string(tt.wantData), "")

			if len(gotDataString) != len(wantData) {
				t.Errorf("cat() len = %d, want %d", len(gotData), len(tt.wantData))
			}

			if !reflect.DeepEqual(gotDataString, wantData) {
				t.Errorf("cat() = %s, want %s", gotData, tt.wantData)
				t.Errorf("cat() = %x, want %x", gotData, tt.wantData)
			}
		})
	}
}

func Test_ls(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name     string
		args     args
		wantData []byte
	}{
		{"ls", args{"."}, []byte("folder/\nfoo\n")},
		{"ls folder", args{"folder"}, []byte("bar\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData := stdout(func() { LsCmd(defaultParse)(nil, []string{tt.args.url}) })
			if !reflect.DeepEqual(string(gotData), string(tt.wantData)) {
				t.Errorf("ls() = %s, want %s", gotData, tt.wantData)
				t.Errorf("ls() = %x, want %x", gotData, tt.wantData)
			}
		})
	}
}

func Test_file(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name     string
		args     args
		wantData []byte
	}{
		{"file", args{"foo"}, []byte("foo: text/plain\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData := stdout(func() { FileCmd(defaultParse)(nil, []string{tt.args.url}) })
			if !reflect.DeepEqual(string(gotData), string(tt.wantData)) {
				t.Errorf("file() = %s, want %s", gotData, tt.wantData)
			}
		})
	}
}

func Test_hashsum(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name     string
		args     args
		wantData []byte
	}{
		{"hashsum", args{"foo"}, []byte("MD5: acbd18db4cc2f85cedef654fccc4a4d8\n" +
			"SHA1: 0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33\n" +
			"SHA256: 2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae\n" +
			"SHA512: f7fbba6e0636f890e56fbbf3283e524c6fa3204ae298382d624741d0dc6638326e282c41be5e4254d8820772c5518a2c5a8c0c7f7eda19594a7eb539453e1ed7\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData := stdout(func() { HashsumCmd(defaultParse)(nil, []string{tt.args.url}) })
			if !reflect.DeepEqual(string(gotData), string(tt.wantData)) {
				t.Errorf("hashsum() = %s, want %s", gotData, tt.wantData)
			}
		})
	}
}

func Test_stat(t *testing.T) {
	result := `Name: foo
Size: 3
IsDir: false
Mode: ----------
Modified: 0001-01-01 00:00:00 +0000 UTC
`
	type args struct {
		url string
	}
	tests := []struct {
		name     string
		args     args
		wantData []byte
	}{
		{"stat", args{"foo"}, []byte(result)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData := stdout(func() { StatCmd(defaultParse)(nil, []string{tt.args.url}) })
			if !reflect.DeepEqual(string(gotData), string(tt.wantData)) {
				t.Errorf("stat() = '%s', want '%s'", gotData, tt.wantData)
			}
		})
	}
}

func Test_tree(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name     string
		args     args
		wantData []byte
	}{
		{"tree", args{"."}, []byte(".\n├── folder\n│   └── bar\n└── foo\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData := stdout(func() { TreeCmd(defaultParse)(nil, []string{tt.args.url}) })
			if !reflect.DeepEqual(string(gotData), string(tt.wantData)) {
				t.Errorf("tree() = '%s', want '%s'", gotData, tt.wantData)
				t.Errorf("tree() = '%x', want '%x'", gotData, tt.wantData)
			}
		})
	}
}

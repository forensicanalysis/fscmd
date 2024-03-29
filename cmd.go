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

// Package fscmd implements the fs command line tool that has various subcommands
// which imitate unix commands but for file system structures.
//
//	cat      Print files
//	file     Determine files types
//	hashsum  Print hashsums
//	ls       List directory contents
//	stat     Display file status
//	strings  Find the printable strings in an object, or other binary, file
//	tree     List contents of directories in a tree-like format
package fscmd

import (
	"crypto/md5"  // #nosec
	"crypto/sha1" // #nosec
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"

	"github.com/forensicanalysis/filetype"
)

const (
	bashCompletionFunc = `__fs_ls_completion()
{
local fs_output out

if [[ -z $cur ]]; then
	return
fi

if fs_output=$(fs ls "$cur" 2>/dev/null); then
	if [[ $cur == */ ]]; then
		COMPREPLY=($( compgen -P "$cur" -W "${fs_output}" ))
		return
	fi
	COMPREPLY=( "$cur " ) # TODO add / for folder ...; add already in first completion...
	return
fi

parent=$(dirname $cur)
if [[ parent == "." ]]; then
	parent=""
fi
partialbase=$(basename $cur)
if fs_output=$(fs ls "$parent" 2>/dev/null); then
	COMPREPLY=($( compgen -P "$parent/" -W "${fs_output}" -- "$partialbase" ))
fi
}

__fs__resource()
{
__fs_ls_completion
if [[ $? -eq 0 ]]; then
	return 0
fi
}

__fs_custom_func() {
case ${last_command} in
	fs_cat | fs_ls)
		__fs__resource
		return
		;;
	*)
		;;
esac
}
`
)

func FSCommand(parseFunc func(_ *cobra.Command, args []string) (fs.FS, []string, error)) *cobra.Command {
	var debug bool
	rootCmd := &cobra.Command{Use: "fs", Short: "recursive file, filesystem and archive commands",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetFlags(log.LstdFlags | log.Llongfile)
			if !debug {
				log.SetOutput(ioutil.Discard)
			}
		},
		BashCompletionFunction: bashCompletionFunc,
	}

	cat := &cobra.Command{Use: "cat", Short: "print files", Run: CatCmd(parseFunc)}
	file := &cobra.Command{Use: "file", Short: "determine file type", Run: FileCmd(parseFunc)}
	hashsum := &cobra.Command{Use: "hashsum", Short: "print hashsums", Run: HashsumCmd(parseFunc)}
	ls := &cobra.Command{Use: "ls", Short: "list directory contents", Run: LsCmd(parseFunc)}
	stat := &cobra.Command{Use: "stat", Short: "display file status", Run: StatCmd(parseFunc)}
	tree := &cobra.Command{Use: "tree", Short: "list contents of directories in a tree-like format", Run: TreeCmd(parseFunc)}
	complete := &cobra.Command{Use: "complete", Hidden: true, Run: func(cmd *cobra.Command, args []string) {
		if err := rootCmd.GenBashCompletionFile(".bash_completion.sh"); err == nil {
			log.Println("--")
			if _, err := os.Stat("/usr/local/etc/bash_completion.d"); !os.IsNotExist(err) {
				err = os.Rename(".bash_completion.sh", "/usr/local/etc/bash_completion.d/fs")
				log.Println(err)
			} else if _, err := os.Stat("/etc/bash_completion.d"); !os.IsNotExist(err) {
				err = os.Rename(".bash_completion.sh", "/etc/bash_completion.d/fs")
				log.Println(err)
			}
		}
	}}
	/* go install . && fs complete && . /usr/local/etc/bash_completion && fs ls <tab> */

	rootCmd.AddCommand(cat, file, hashsum, ls, stat, tree, complete)
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug output")
	_ = rootCmd.PersistentFlags().MarkHidden("debug")
	return rootCmd
}

func exitOnError(err error) {
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		// os.Exit(1)
		log.Fatal(err)
	}
}

func CatCmd(parse func(*cobra.Command, []string) (fs.FS, []string, error)) func(_ *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		fsys, names, err := parse(cmd, args)
		exitOnError(err)
		for _, name := range names {
			func() {
				r, err := fsys.Open(name)
				exitOnError(err)
				defer r.Close()

				_, err = io.Copy(os.Stdout, r)
				exitOnError(err)
			}()
		}
	}
}

func FileCmd(parse func(*cobra.Command, []string) (fs.FS, []string, error)) func(_ *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		fsys, names, err := parse(cmd, args)
		exitOnError(err)
		b := make([]byte, 8192)
		for _, name := range names {
			func() {
				f, err := fsys.Open(name)
				exitOnError(err)
				defer f.Close()
				_, err = f.Read(b)
				exitOnError(err)
				fmt.Printf("%s: %s\n", name, filetype.Detect(b).Mimetype.Value)
			}()
		}
	}
}

func HashsumCmd(parse func(*cobra.Command, []string) (fs.FS, []string, error)) func(_ *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		fsys, names, err := parse(cmd, args)
		exitOnError(err)
		for _, name := range names {
			md5hash := md5.New()   // #nosec
			sha1hash := sha1.New() // #nosec
			sha256hash := sha256.New()
			sha512hash := sha512.New()
			hash := io.MultiWriter(md5hash, sha1hash, sha256hash, sha512hash)

			r, err := fsys.Open(name)
			exitOnError(err)
			_, err = io.Copy(hash, r)
			exitOnError(err)
			exitOnError(r.Close())
			fmt.Printf("MD5: %x\n", md5hash.Sum(nil))
			fmt.Printf("SHA1: %x\n", sha1hash.Sum(nil))
			fmt.Printf("SHA256: %x\n", sha256hash.Sum(nil))
			fmt.Printf("SHA512: %x\n", sha512hash.Sum(nil))
		}
	}
}

func LsCmd(parse func(*cobra.Command, []string) (fs.FS, []string, error)) func(_ *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		fsys, names, err := parse(cmd, args)
		exitOnError(err)

		if len(names) == 0 {
			names = []string{"."}
		}
		for _, name := range names {
			fi, err := fs.Stat(fsys, name)
			exitOnError(err)
			if fi.IsDir() {
				entries, err := fs.ReadDir(fsys, name)
				exitOnError(err)

				for _, entry := range entries {
					child, err := fs.Stat(fsys, path.Join(name, entry.Name()))
					if err != nil {
						fmt.Println(entry, err)
						continue
					}
					if child.IsDir() {
						fmt.Println(entry.Name() + "/")
					} else {
						fmt.Println(entry.Name())
					}
				}
			} else {
				fmt.Println(name)
			}
		}
	}
}

func StatCmd(parse func(*cobra.Command, []string) (fs.FS, []string, error)) func(_ *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		fsys, names, err := parse(cmd, args)
		exitOnError(err)
		for _, name := range names {
			fi, err := fs.Stat(fsys, name)
			exitOnError(err)
			fmt.Printf("Name: %v\n", fi.Name())
			fmt.Printf("Size: %v\n", fi.Size())
			fmt.Printf("IsDir: %v\n", fi.IsDir())
			fmt.Printf("Mode: %s\n", fi.Mode())
			fmt.Printf("Modified: %s\n", fi.ModTime())
		}
	}
}

func TreeCmd(parse func(*cobra.Command, []string) (fs.FS, []string, error)) func(_ *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		fsys, names, err := parse(cmd, args)
		exitOnError(err)
		if len(names) == 0 {
			names = []string{"."}
		}
		for _, name := range names {
			tree := treeprint.New()
			tree.SetValue(name)
			children(fsys, tree, name)
			fmt.Println(strings.TrimSpace(tree.String()))
		}
	}
}

func children(fsys fs.FS, tree treeprint.Tree, name string) {
	fi, err := fs.Stat(fsys, name)
	exitOnError(err)
	if fi.IsDir() {
		entries, _ := fs.ReadDir(fsys, name)
		exitOnError(err)

		for _, entry := range entries {
			child := tree.AddBranch(entry.Name())
			children(fsys, child, path.Join(name, entry.Name()))
		}
	}
}

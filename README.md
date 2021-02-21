<h1 align="center">fscmd</h1>

<p  align="center">
 <a href="https://godocs.io/github.com/forensicanalysis/fscmd"><img src="https://godocs.io/github.com/forensicanalysis/fscmd?status.svg" alt="doc" /></a>
</p>

Create command line tools with various subcommands
which imitate unix commands but for [io/fs.FS](https://golang.org/pkg/io/fs) file system structures.

Subcommands:
 - **cat**: Print file contents
 - **file**: Determine files types
 - **hashsum**: Print hashsums
 - **ls**: List directory contents
 - **stat**: Display file status
 - **tree**: List contents of directories in a tree-like format

# Usage

``` go
func main() {
	fsys := &fstest.MapFS{
		"foo":        &fstest.MapFile{Data: []byte("foo")},
		"folder/bar": &fstest.MapFile{Data: []byte("bar")},
	}
	fsCmd := fscmd.FSCommand(fsys, nil)
	fsCmd.Use = "fs"
	fsCmd.Short = "example for fscmd usage"
	err := fsCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
```

```
go build . && ./fs cat foo
```

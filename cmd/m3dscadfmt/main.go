// m3dscadfmt formats m3dscad source files, or standard input when no files are given.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/unixpickle/m3dscad/scad"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	flags := flag.NewFlagSet("m3dscadfmt", flag.ContinueOnError)
	flags.SetOutput(errOut)
	write := flags.Bool("w", false, "write formatted source back to each file")
	list := flags.Bool("l", false, "list files whose formatting differs")
	flags.Usage = func() {
		fmt.Fprintln(errOut, "Usage: m3dscadfmt [-w] [-l] [file.scad ...]\nWith no files, reads standard input and writes formatted source to standard output.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	paths := flags.Args()
	if len(paths) == 0 {
		if *write || *list {
			fmt.Fprintln(errOut, "-w and -l require file arguments")
			return 2
		}
		paths = []string{""}
	}
	status := 0
	for _, path := range paths {
		err := func() error {
			var src []byte
			var err error
			if path == "" {
				src, err = io.ReadAll(in)
			} else {
				src, err = os.ReadFile(path)
			}
			if err != nil {
				return err
			}
			formatted, err := scad.FormatFile(path, string(src))
			if err != nil {
				return err
			}
			if formatted != string(src) {
				if *list {
					if _, err := fmt.Fprintln(out, path); err != nil {
						return err
					}
				}
				if *write {
					return replaceFile(path, []byte(formatted))
				}
			}
			if !*list && !*write {
				_, err = io.WriteString(out, formatted)
			}
			return err
		}()
		if err != nil {
			fmt.Fprintln(errOut, err)
			status = 1
		}
	}
	return status
}

// Replace only after formatting and writing succeed. Resolve symlinks so -w
// updates the source file rather than replacing a link with a regular file.
func replaceFile(path string, contents []byte) error {
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".m3dscadfmt-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := file.Chmod(info.Mode()); err != nil {
		return err
	}
	if _, err := file.Write(contents); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

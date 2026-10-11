package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStdin(t *testing.T) {
	var out, errors bytes.Buffer
	if status := run(nil, strings.NewReader("cube ( 1 );"), &out, &errors); status != 0 || out.String() != "cube(1);\n" {
		t.Fatalf("status=%d output=%q errors=%s", status, out.String(), &errors)
	}
	out.Reset()
	if status := run(nil, strings.NewReader("cube("), &out, &errors); status != 1 || out.Len() != 0 {
		t.Fatalf("invalid input: status=%d output=%q", status, out.String())
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	good, bad, link := filepath.Join(dir, "good.scad"), filepath.Join(dir, "bad.scad"), filepath.Join(dir, "link.scad")
	if err := os.WriteFile(good, []byte("cube (1);"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("cube("), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(good, link); err != nil {
		t.Fatal(err)
	}
	var out, errors bytes.Buffer
	if status := run([]string{"-w", "-l", bad, link}, nil, &out, &errors); status != 1 {
		t.Fatalf("status=%d, errors=%s", status, &errors)
	}
	if out.String() != link+"\n" {
		t.Fatalf("listed %q", out.String())
	}
	for path, want := range map[string]string{good: "cube(1);\n", bad: "cube("} {
		contents, err := os.ReadFile(path)
		if err != nil || string(contents) != want {
			t.Fatalf("%s: contents=%q error=%v", path, contents, err)
		}
	}
	if info, err := os.Stat(good); err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("file permissions: %v, %v", info, err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink: %v, %v", info, err)
	}
	out.Reset()
	if status := run([]string{"-l", good}, nil, &out, &errors); status != 0 || out.Len() != 0 {
		t.Fatal("formatted file listed")
	}
}

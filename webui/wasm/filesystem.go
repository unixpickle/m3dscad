//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"syscall/js"
	"time"
)

// jsFileSystem reads files on demand through the worker's JavaScript bridge.
// The worker forwards each request to the main thread, which reads the latest
// value directly from IndexedDB.
type jsFileSystem struct{}

func (j jsFileSystem) Open(name string) (fs.File, error) {
	data, err := j.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return &jsFile{
		Reader: bytes.NewReader(data),
		info: jsFileInfo{
			name: name,
			size: int64(len(data)),
		},
	}, nil
}

func (j jsFileSystem) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	data, err := readFileFromJS(name)
	if err != nil {
		return nil, &fs.PathError{Op: "read", Path: name, Err: err}
	}
	return data, nil
}

func readFileFromJS(name string) ([]byte, error) {
	requestFn := js.Global().Get("m3dscadReadFile")
	if requestFn.Type() != js.TypeFunction {
		return nil, fmt.Errorf("filesystem bridge is unavailable")
	}
	promise := requestFn.Invoke(name)
	type promiseResult struct {
		value js.Value
		err   error
	}
	resultChan := make(chan promiseResult, 1)
	var thenFn js.Func
	var catchFn js.Func
	thenFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
		value := js.Null()
		if len(args) > 0 {
			value = args[0]
		}
		resultChan <- promiseResult{value: value}
		return nil
	})
	catchFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
		errMsg := "file read failed"
		if len(args) > 0 {
			errMsg = jsErrorString(args[0], errMsg)
		}
		resultChan <- promiseResult{err: fmt.Errorf("%s", errMsg)}
		return nil
	})
	defer thenFn.Release()
	defer catchFn.Release()
	promise.Call("then", thenFn)
	promise.Call("catch", catchFn)
	result := <-resultChan
	if result.err != nil {
		return nil, result.err
	}
	return bytesFromJS(result.value)
}

func bytesFromJS(value js.Value) ([]byte, error) {
	uint8Array := js.Global().Get("Uint8Array")
	if value.Type() != js.TypeObject || !value.InstanceOf(uint8Array) {
		return nil, fmt.Errorf("file contents must be a Uint8Array")
	}
	data := make([]byte, value.Get("byteLength").Int())
	if copied := js.CopyBytesToGo(data, value); copied != len(data) {
		return nil, fmt.Errorf("copied %d of %d bytes", copied, len(data))
	}
	return data, nil
}

type jsFile struct {
	*bytes.Reader
	info jsFileInfo
}

func (j *jsFile) Close() error               { return nil }
func (j *jsFile) Stat() (fs.FileInfo, error) { return j.info, nil }

type jsFileInfo struct {
	name string
	size int64
}

func (j jsFileInfo) Name() string       { return j.name }
func (j jsFileInfo) Size() int64        { return j.size }
func (j jsFileInfo) Mode() fs.FileMode  { return 0o444 }
func (j jsFileInfo) ModTime() time.Time { return time.Time{} }
func (j jsFileInfo) IsDir() bool        { return false }
func (j jsFileInfo) Sys() any           { return nil }

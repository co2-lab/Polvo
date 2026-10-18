// Command libpolvo is the bridge as a static C library, for the window to link into its executable.
// Each function is the bridge's function of the same name; how they cross into C is the window's
// spec of the library (desktop/src/remote.spec.md), whose tests use this library.
package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"github.com/co2-lab/polvo/internal/bridge"
)

// fail hands err to C through out, as text to free with polvo_free.
func fail(out **C.char, err error) {
	if out != nil {
		*out = C.CString(err.Error())
	}
}

//export polvo_open
func polvo_open(dir, listen, shell, relay *C.char, port C.int, err **C.char) C.int64_t {
	h, e := bridge.Open(C.GoString(dir), C.GoString(listen), C.GoString(shell), C.GoString(relay), int(port))
	if e != nil {
		fail(err, e)
		return 0
	}
	return C.int64_t(h)
}

//export polvo_close
func polvo_close(h C.int64_t) { bridge.Close(bridge.Handle(h)) }

//export polvo_call
func polvo_call(h C.int64_t, req *C.char, err **C.char) *C.char {
	b, e := bridge.Call(bridge.Handle(h), []byte(C.GoString(req)))
	if e != nil {
		fail(err, e)
		return nil
	}
	return C.CString(string(b))
}

//export polvo_attach
func polvo_attach(h C.int64_t, machine, id *C.char, err **C.char) C.int64_t {
	t, e := bridge.Attach(bridge.Handle(h), C.GoString(machine), C.GoString(id))
	if e != nil {
		fail(err, e)
		return 0
	}
	return C.int64_t(t)
}

//export polvo_read
func polvo_read(t C.int64_t, buf *C.uchar, n C.int, err **C.char) C.int {
	p := unsafe.Slice((*byte)(unsafe.Pointer(buf)), int(n))
	k, e := bridge.Read(bridge.Handle(t), p)
	if e != nil {
		fail(err, e)
		return -1
	}
	return C.int(k)
}

//export polvo_write
func polvo_write(t C.int64_t, buf *C.uchar, n C.int, err **C.char) C.int {
	p := C.GoBytes(unsafe.Pointer(buf), n)
	k, e := bridge.Write(bridge.Handle(t), p)
	if e != nil {
		fail(err, e)
		return -1
	}
	return C.int(k)
}

//export polvo_resize
func polvo_resize(t C.int64_t, rows, cols C.uint16_t, err **C.char) C.int {
	if e := bridge.Resize(bridge.Handle(t), uint16(rows), uint16(cols)); e != nil {
		fail(err, e)
		return -1
	}
	return 0
}

//export polvo_detach
func polvo_detach(t C.int64_t) { bridge.Detach(bridge.Handle(t)) }

//export polvo_free
func polvo_free(p *C.char) { C.free(unsafe.Pointer(p)) }

func main() {}

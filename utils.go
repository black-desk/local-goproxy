package main

/*
#cgo LDFLAGS: -ldpkg
#define LIBDPKG_VOLATILE_API
#include <dpkg/dpkg.h>
#include <dpkg/fsys.h>
#include <dpkg/db-fsys.h>
#include <dpkg/pkg-list.h>
#include <fnmatch.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"unsafe"
)

func init() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	C.dpkg_program_init(C.CString("test"))
	C.dpkg_db_set_dir(C.CString("/var/lib/dpkg"))
	C.modstatdb_open(C.msdbrw_readonly)
	C.ensure_allinstfiles_available_quiet()
	C.ensure_diversions()
	C.fsys_hash_init()
	go func() {
		<-c
		C.fsys_hash_reset()
		C.dpkg_program_done()
	}()
}

var mux sync.Mutex

func GetVersion(absPath string) (version string, err error) {
	mux.Lock()
	defer mux.Unlock()
	cpath := C.CString(absPath)
	defer C.free(unsafe.Pointer(cpath))

	node := C.fsys_hash_find_node(cpath, 0)
	if node.packages == nil {
		err = fmt.Errorf("cannot find file %s belongs to which package in dpkg database", absPath)
		return
	}

	version = "v" + C.GoString(node.packages.pkg.installed.version.version)
	return
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

//go:build linux

package main

/*
#cgo CFLAGS: -I${SRCDIR}
#cgo LDFLAGS: -L${SRCDIR}/enet/lib -l:libenet_linux.a
#include "enet/enet.h"
#include <stdlib.h>
*/
import "C"

func enetInit() int {
	return int(C.enet_initialize())
}

func enetDeinit() {
	C.enet_deinitialize()
}

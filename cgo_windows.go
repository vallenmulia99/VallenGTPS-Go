//go:build windows

package main

/*
#cgo CFLAGS: -I${SRCDIR}
#cgo LDFLAGS: -L${SRCDIR}/enet/lib -lenet -lws2_32 -lwinmm
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

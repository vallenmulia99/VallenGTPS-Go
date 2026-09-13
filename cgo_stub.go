//go:build !windows && !linux

package main

// Stub untuk IDE / gopls agar tidak error saat analisis.
// File ini TIDAK akan ikut compile saat build Windows maupun Linux.

func enetInit() int  { return 0 }
func enetDeinit()    {}

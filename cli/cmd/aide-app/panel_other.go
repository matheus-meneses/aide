//go:build !darwin && cgo

package main

import "unsafe"

func promotePanel(unsafe.Pointer) {}

func showPanelNative(unsafe.Pointer) {}

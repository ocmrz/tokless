//go:build windows

package util

func suspendTTY(restore func()) (func(), bool) {
	return restore, false
}

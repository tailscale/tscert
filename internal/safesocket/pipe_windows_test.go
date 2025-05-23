// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package safesocket

import (
	"golang.org/x/sys/windows"
)

func init() {
	// downgradeSDDL is a test helper that downgrades the windowsSDDL variable if
	// the currently running user does not have sufficient priviliges to set the
	// SDDL.
	downgradeSDDL = func() (cleanup func()) {
		// The current default descriptor can not be set by mere mortal users,
		// so we need to undo that for executing tests as a regular user.
		if !isCurrentProcessElevated() {
			var orig string
			orig, windowsSDDL = windowsSDDL, ""
			return func() { windowsSDDL = orig }
		}
		return func() {}
	}
}

func isCurrentProcessElevated() bool {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return false
	}
	defer token.Close()

	return token.IsElevated()
}

//go:build windows

package engine

import (
	"errors"
	"syscall"
)

const (
	// PROCESS_QUERY_LIMITED_INFORMATION. Enough to read a process's exit
	// code, and grantable for processes this one does not own, which
	// PROCESS_QUERY_INFORMATION often is not.
	processQueryLimitedInformation = 0x1000

	// STILL_ACTIVE, the value GetExitCodeProcess reports for a process that
	// has not exited yet.
	stillActive = 259
)

// processAlive reports whether a process with the given PID exists.
//
// Opening a handle is not on its own a liveness check. Windows keeps a dead
// process's record until every handle to it is closed, and OpenProcess — what
// os.FindProcess does — keeps succeeding for the whole of that window. A
// parent that has killed a child but not yet reaped it still holds such a
// handle, so os.FindProcess answers "alive" for a process that is plainly
// dead.
//
// That is exactly the state a crash leaves behind, and exactly the state this
// check exists for: it made the engine refuse to reopen its own data
// directory after a crash, reporting the lock as held by a running process.
// Reading the exit code tells the two cases apart.
func processAlive(pid int) bool {
	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		// Access denied means the process exists but belongs to another
		// user. Refusing to steal its lock is the correct behaviour, and it
		// matches what the Unix build does with EPERM.
		return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
	}
	defer syscall.CloseHandle(handle)

	var code uint32
	if err := syscall.GetExitCodeProcess(handle, &code); err != nil {
		// The handle opened but the exit code is unreadable. Treating that
		// as alive keeps the lock rather than risking two writers.
		return true
	}
	return code == stillActive
}

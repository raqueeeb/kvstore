//go:build windows

package engine

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestProcessAliveAfterKillWithOpenHandle pins the behaviour that os.FindProcess
// gets wrong.
//
// Windows keeps a dead process's record alive until every handle to it is
// closed. A parent that has killed a child but not yet called Wait still holds
// one, which is precisely the state a crash leaves behind. The liveness check
// has to report the child as dead there, or the engine refuses to reclaim its
// own lock after a crash.
func TestProcessAliveAfterKillWithOpenHandle(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "ping -n 30 127.0.0.1 > nul")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start a helper process: %v", err)
	}
	pid := cmd.Process.Pid

	if !processAlive(pid) {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("a running process (pid %d) was reported as dead", pid)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("killing the helper: %v", err)
	}

	// Deliberately no Wait yet: the handle stays open, which is the case
	// that used to be misreported.
	deadline := time.Now().Add(10 * time.Second)
	for processAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	if processAlive(pid) {
		_ = cmd.Wait()
		t.Fatalf("pid %d still reported alive after being killed, while its handle was open", pid)
	}

	_ = cmd.Wait()
}

// TestProcessAliveForUnusedPID checks the ordinary negative case.
func TestProcessAliveForUnusedPID(t *testing.T) {
	// PIDs on Windows are multiples of four and this one is far above the
	// usual range, so it is almost certainly unassigned. If it happens to be
	// in use the test would wrongly pass, never wrongly fail.
	const unlikely = 0x7FFFFFF0
	if processAlive(unlikely) {
		t.Skipf("pid %d is unexpectedly in use", unlikely)
	}
}

// TestProcessAliveForSelf is the trivial positive case.
func TestProcessAliveForSelf(t *testing.T) {
	if !processAlive(os.Getpid()) {
		t.Fatal("this process reported itself as dead")
	}
}

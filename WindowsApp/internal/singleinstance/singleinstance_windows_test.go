//go:build windows

package singleinstance

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestAcquireAllowsOnlyOnePrimaryInstance(t *testing.T) {
	name := fmt.Sprintf(`Local\ZhengshiWMS.WindowsApp.Test.%d`, time.Now().UnixNano())
	first, primary, err := Acquire(name)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if !primary {
		t.Fatal("first acquisition was not primary")
	}
	pid, err := readPID(first.pidPath)
	if err != nil {
		t.Fatal(err)
	}
	if pid != uint32(os.Getpid()) {
		t.Fatalf("primary pid = %d", pid)
	}
	second, primary, err := Acquire(name)
	if err != nil {
		t.Fatal(err)
	}
	if second == nil {
		t.Fatal("second acquisition did not return a guard")
	}
	if primary {
		second.Close()
		t.Fatal("second acquisition unexpectedly became primary")
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReadPIDRejectsCorruptState(t *testing.T) {
	name := t.TempDir() + `\instance.pid`
	if err := os.WriteFile(name, []byte("not-a-process"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPID(name); err == nil {
		t.Fatal("expected corrupt pid error")
	}
}

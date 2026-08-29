package noderun

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDenyCommandHighRisk(t *testing.T) {
	if runtime.GOOS == "windows" {
		if !denyCommand("Stop-Computer") {
			t.Fatal("Stop-Computer")
		}
		if !denyCommand("Restart-Computer -Force") {
			t.Fatal("Restart-Computer")
		}
		if denyCommand("Write-Output hello") {
			t.Fatal("benign powershell denied")
		}
		return
	}
	if !denyCommand("sudo rm -rf /tmp/x") {
		t.Fatal("sudo")
	}
	if !denyCommand("rm -rf /") {
		t.Fatal("rm -rf /")
	}
	if denyCommand("echo hello") {
		t.Fatal("benign denied")
	}
}

func TestDenySensitiveSSHDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	p := filepath.Join(home, ".ssh", "id_rsa")
	if !denySensitivePath(p) {
		t.Fatalf("expected deny %s", p)
	}
	if denySensitivePath(t.TempDir()) {
		t.Fatal("temp dir is not sensitive")
	}
}

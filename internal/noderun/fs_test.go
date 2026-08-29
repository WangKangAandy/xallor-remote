package noderun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInWorkspace(t *testing.T) {
	ws := t.TempDir()
	inside, err := resolveInWorkspace(ws, "a/b.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(ws, "a", "b.txt")
	if inside != want {
		t.Fatalf("got %s want %s", inside, want)
	}
	if _, err := resolveInWorkspace(ws, filepath.Join(ws, "..", "outside.txt")); err == nil {
		t.Fatal("expected deny for parent traversal")
	}
	absOut := filepath.Join(os.TempDir(), "xallor-outside.txt")
	if _, err := resolveInWorkspace(ws, absOut); err == nil {
		t.Fatal("expected deny for abs outside")
	}
}

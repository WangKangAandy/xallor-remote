package noderun

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveInWorkspace(workspace, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("empty path")
	}
	ws, err := filepath.Abs(filepath.Clean(workspace))
	if err != nil {
		return "", err
	}
	target := p
	if !filepath.IsAbs(target) {
		target = filepath.Join(ws, p)
	}
	target, err = filepath.Abs(filepath.Clean(target))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(ws, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("outside workspace")
	}
	if denySensitivePath(target) {
		return "", fmt.Errorf("sensitive path")
	}
	if real, err := filepath.EvalSymlinks(target); err == nil {
		rel2, err := filepath.Rel(ws, real)
		if err != nil || rel2 == ".." || strings.HasPrefix(rel2, ".."+string(os.PathSeparator)) {
			return "", fmt.Errorf("symlink escapes workspace")
		}
		if denySensitivePath(real) {
			return "", fmt.Errorf("sensitive path")
		}
		target = real
	}
	return target, nil
}

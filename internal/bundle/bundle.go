package bundle

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed agents tools
var files embed.FS

type Result struct {
	Written []string
	Kept    []string
}

func Write(projectDir string) (Result, error) {
	var res Result

	err := fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		dest := filepath.Join(projectDir, ".opencode", path)
		if _, err := os.Stat(dest); err == nil {
			res.Kept = append(res.Kept, path)
			return nil
		}

		data, err := files.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return err
		}
		res.Written = append(res.Written, path)
		return nil
	})
	if err != nil {
		return res, fmt.Errorf("bundle: write to %s: %w", projectDir, err)
	}
	return res, nil
}

func Agents() ([]string, error) {
	entries, err := files.ReadDir("agents")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

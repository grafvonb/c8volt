// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package embedded

import (
	"embed"
	"io/fs"
	"path/filepath"
)

//go:embed **/*.bpmn
var FS embed.FS

func List() ([]string, error) {
	var out []string
	err := fs.WalkDir(FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(p) == ".bpmn" {
			out = append(out, p)
		}
		return nil
	})
	return out, err
}

/*
   Copyright (c) 2026 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package rpm

import (
	"fmt"
	"os"
	"path/filepath"
)

var rpmdbFileCandidates = []string{"rpmdb.sqlite", "Packages.db", "Packages"}
var rpmdbDirCandidates = []string{
	filepath.FromSlash("var/lib/rpm"),
	filepath.FromSlash("usr/lib/sysimage/rpm"),
}

type RPMDBLocation struct {
	Root     string
	Dir      string
	RelDir   string
	FileName string
	FilePath string
}

func FindRPMDB(root string) (RPMDBLocation, error) {
	if root == "" {
		return RPMDBLocation{}, fmt.Errorf("rpmdb root is empty")
	}

	tryDirs := []string{root}
	for _, d := range rpmdbDirCandidates {
		tryDirs = append(tryDirs, filepath.Join(root, d))
	}

	for _, dbDir := range tryDirs {
		for _, f := range rpmdbFileCandidates {
			fp := filepath.Join(dbDir, f)
			if _, err := os.Stat(fp); err == nil {
				relDir, err := filepath.Rel(root, dbDir)
				if err != nil {
					return RPMDBLocation{}, fmt.Errorf("failed to compute rpmdb relative dir: %w", err)
				}
				return RPMDBLocation{
					Root:     root,
					Dir:      dbDir,
					RelDir:   relDir,
					FileName: f,
					FilePath: fp,
				}, nil
			}
		}
	}

	return RPMDBLocation{}, fmt.Errorf("rpmdb not found under %s", root)
}


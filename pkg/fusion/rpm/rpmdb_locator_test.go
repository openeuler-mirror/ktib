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
	"os"
	"path/filepath"
	"testing"
)

func TestFindRPMDB_VarLibRPM(t *testing.T) {
	root := t.TempDir()
	dbDir := filepath.Join(root, filepath.FromSlash("var/lib/rpm"))
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbFile := filepath.Join(dbDir, "rpmdb.sqlite")
	if err := os.WriteFile(dbFile, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	loc, err := FindRPMDB(root)
	if err != nil {
		t.Fatalf("expected to find rpmdb, got err=%v", err)
	}
	if loc.FileName != "rpmdb.sqlite" {
		t.Fatalf("unexpected FileName=%s", loc.FileName)
	}
	if loc.RelDir != filepath.FromSlash("var/lib/rpm") {
		t.Fatalf("unexpected RelDir=%s", loc.RelDir)
	}
}

func TestFindRPMDB_Sysimage(t *testing.T) {
	root := t.TempDir()
	dbDir := filepath.Join(root, filepath.FromSlash("usr/lib/sysimage/rpm"))
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbFile := filepath.Join(dbDir, "Packages.db")
	if err := os.WriteFile(dbFile, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	loc, err := FindRPMDB(root)
	if err != nil {
		t.Fatalf("expected to find rpmdb, got err=%v", err)
	}
	if loc.FileName != "Packages.db" {
		t.Fatalf("unexpected FileName=%s", loc.FileName)
	}
	if loc.RelDir != filepath.FromSlash("usr/lib/sysimage/rpm") {
		t.Fatalf("unexpected RelDir=%s", loc.RelDir)
	}
}


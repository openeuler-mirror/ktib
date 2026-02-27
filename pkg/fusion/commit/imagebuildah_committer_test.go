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

package commit

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestTarDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "os-release"), []byte("ID=kylin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/bin/bash", filepath.Join(root, "bin", "sh")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	tarPath := filepath.Join(t.TempDir(), "rootfs.tar")
	if err := tarDirectory(root, tarPath); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tr := tar.NewReader(f)
	seen := map[string]tar.Header{}
	for {
		h, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		seen[h.Name] = *h
	}

	if _, ok := seen["etc/"]; !ok {
		t.Fatalf("expected etc/ dir in tar, got keys=%v", keys(seen))
	}
	if _, ok := seen["etc/os-release"]; !ok {
		t.Fatalf("expected etc/os-release in tar, got keys=%v", keys(seen))
	}
	if h, ok := seen["bin/sh"]; !ok {
		t.Fatalf("expected bin/sh in tar, got keys=%v", keys(seen))
	} else if h.Typeflag != tar.TypeSymlink {
		t.Fatalf("expected symlink type for bin/sh, got %v", h.Typeflag)
	}
}

func keys(m map[string]tar.Header) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

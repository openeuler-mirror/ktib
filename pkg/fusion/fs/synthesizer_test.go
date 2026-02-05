/*
   Copyright (c) 2023 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package fs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	csarchive "github.com/containers/storage/pkg/archive"
)

func TestApplyTarWithFilter_GzipTarRequiresDecompression(t *testing.T) {
	t.Parallel()

	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	err := tw.WriteHeader(&tar.Header{
		Name: "var/lib/rpm/Packages",
		Mode: 0o644,
		Size: int64(len("dummy")),
	})
	if err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := tw.Write([]byte("dummy")); err != nil {
		t.Fatalf("write body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}

	var gzBuf bytes.Buffer
	zw := gzip.NewWriter(&gzBuf)
	if _, err := io.Copy(zw, &tarBuf); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	dest := t.TempDir()
	filter := func(p string) bool { return p == "/var/lib/rpm/Packages" }

	if err := applyTarWithFilter(bytes.NewReader(gzBuf.Bytes()), dest, filter); err == nil {
		t.Fatalf("expected invalid tar header error when applying gzipped tar without decompression")
	}

	decompressed, err := csarchive.DecompressStream(io.NopCloser(bytes.NewReader(gzBuf.Bytes())))
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	defer decompressed.Close()

	if err := applyTarWithFilter(decompressed, dest, filter); err != nil {
		t.Fatalf("apply: %v", err)
	}

	outPath := filepath.Join(dest, "var/lib/rpm/Packages")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != "dummy" {
		t.Fatalf("unexpected extracted content: %q", string(data))
	}
}

func TestDecompressStream_InvalidInput(t *testing.T) {
	t.Parallel()

	decompressed, err := csarchive.DecompressStream(io.NopCloser(bytes.NewReader([]byte{0x1f, 0x8b, 0x08})))
	if err != nil {
		return
	}
	defer decompressed.Close()

	if _, err := io.ReadAll(decompressed); err == nil {
		t.Fatalf("expected decompression read error")
	}
}

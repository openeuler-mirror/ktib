//go:build !linux
package fs

import (
	"archive/tar"
)

func setFileMetadata(path string, hdr *tar.Header) error {
	// No-op for non-Linux or when syscalls are not available
	return nil
}

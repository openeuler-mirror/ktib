//go:build linux

package fs

import (
	"archive/tar"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func setFileMetadata(path string, hdr *tar.Header) error {
	// 1. Ownership (Lchown to handle symlinks correctly if possible, but os.Lchown is good)
	if err := os.Lchown(path, hdr.Uid, hdr.Gid); err != nil {
		// Log but continue? strict mode might fail.
		// For now, return error to be safe, or maybe ignore if running as non-root?
		// ktib usually runs as root (in container).
		return err
	}

	// 2. Xattrs
	for k, v := range hdr.Xattrs {
		// syscall.Lsetxattr is needed for symlinks, Setxattr follows links.
		// But Go syscall package on Linux has Lsetxattr.
		if err := unix.Lsetxattr(path, k, []byte(v), 0); err != nil {
			// Some filesystems don't support xattr, or permission denied.
			// Warning is better than failure here usually.
			// But for strict synthesis, maybe error?
			// Let's return error for now.
			return err
		}
	}

	// 3. Mode (Permissions)
	// Symlinks don't have permissions on Linux (they are 777 always).
	// So only apply for non-symlinks.
	if hdr.Typeflag != tar.TypeSymlink {
		// os.Chmod follows symlinks. We want to change the file itself.
		// If it's a regular file, Chmod is fine.
		// If it's a hard link, Chmod affects all links (which is correct).
		if err := os.Chmod(path, os.FileMode(hdr.Mode)); err != nil {
			return err
		}
	}

	// 4. Times (Access and Mod time)
	// os.Chtimes follows symlinks.
	// To set times on symlink itself, we need syscall.UtimesNanoAt or similar.
	// Go doesn't expose Lutimes easily in os package.
	// For now, we skip time setting for symlinks or just accept we set target's time.
	// Setting target's time is default behavior of tar usually unless --atime-preserve.
	if hdr.Typeflag != tar.TypeSymlink {
		if err := os.Chtimes(path, hdr.AccessTime, hdr.ModTime); err != nil {
			return err
		}
	} else {
		// Try to use Lutimes if available via syscall?
		// syscall.Lutimes exists on Linux.
		ts := []unix.Timeval{
			timeToTimeval(hdr.AccessTime),
			timeToTimeval(hdr.ModTime),
		}
		// ignore error for symlinks if it fails
		_ = unix.Lutimes(path, ts)
	}

	return nil
}

func timeToTimeval(t time.Time) unix.Timeval {
	return unix.Timeval{
		Sec:  t.Unix(),
		Usec: int64(t.Nanosecond() / 1000),
	}
}

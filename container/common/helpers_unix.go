// +build linux

package common

import (
	"os"
	"path"
	"runtime"
	"syscall"
	"unsafe"
)

func listDirectories(dirpath string, parent string, recursive bool, output map[string]struct{}) error {
	f, err := os.Open(dirpath)
	if err != nil {
		// Ignore if this hierarchy does not exist.
		if os.IsNotExist(err) {
			err = nil
		}
		return err
	}

	fd := int(f.Fd())
	buf := make([]byte, 8192)
	var nbuf, bufp int

	dirs := make([]string, 0, 128)
	for {
		if bufp >= nbuf {
			bufp = 0
			var errno error
			nbuf, errno = syscall.ReadDirent(fd, buf)
			runtime.KeepAlive(f)
			if errno != nil {
				return os.NewSyscallError("readdirs", errno)
			}
			if nbuf <= 0 {
				break
			}
		}
		var nb int
		nb, dirs = parseDirent(buf[bufp:nbuf], dirs)
		bufp += nb
	}
	f.Close()

	for _, d := range dirs {
		name := path.Join(parent, d)
		output[name] = struct{}{}

		// List subcontainers if asked to.
		if recursive {
			err := listDirectories(path.Join(dirpath, d), name, true, output)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func parseDirent(buf []byte, dirs []string) (consumed int, newdirs []string) {
	origlen := len(buf)
	for len(buf) > 0 {
		reclen, ok := direntReclen(buf)
		if !ok || reclen > uint64(len(buf)) {
			return origlen, dirs
		}
		rec := buf[:reclen]
		buf = buf[reclen:]
		ino, ok := direntIno(rec)
		if !ok {
			break
		}
		if ino == 0 { // File absent in directory.
			continue
		}
		dt, ok := direntType(rec)
		if !ok {
			break
		}
		// We only grab directories.
		if dt != syscall.DT_DIR {
			continue
		}

		const namoff = uint64(unsafe.Offsetof(syscall.Dirent{}.Name))
		namlen, ok := direntNamlen(rec)
		if !ok || namoff+namlen > uint64(len(rec)) {
			break
		}
		name := rec[namoff : namoff+namlen]
		for i, c := range name {
			if c == 0 {
				name = name[:i]
				break
			}
		}
		// Check for useless names before allocating a string.
		if string(name) == "." || string(name) == ".." {
			continue
		}

		dirs = append(dirs, string(name))
	}
	return origlen - len(buf), dirs
}

// readInt returns the size-bytes unsigned integer in native byte order at offset off.
func readInt(b []byte, off, size uintptr) (u uint64, ok bool) {
	if len(b) < int(off+size) {
		return 0, false
	}
	if isBigEndian {
		return readIntBE(b[off:], size), true
	}
	return readIntLE(b[off:], size), true
}

func readIntBE(b []byte, size uintptr) uint64 {
	switch size {
	case 1:
		return uint64(b[0])
	case 2:
		_ = b[1] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[1]) | uint64(b[0])<<8
	case 4:
		_ = b[3] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[3]) | uint64(b[2])<<8 | uint64(b[1])<<16 | uint64(b[0])<<24
	case 8:
		_ = b[7] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
			uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
	default:
		panic("common: readInt with unsupported size")
	}
}

func readIntLE(b []byte, size uintptr) uint64 {
	switch size {
	case 1:
		return uint64(b[0])
	case 2:
		_ = b[1] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[0]) | uint64(b[1])<<8
	case 4:
		_ = b[3] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24
	case 8:
		_ = b[7] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
	default:
		panic("common: readInt with unsupported size")
	}
}

func direntIno(buf []byte) (uint64, bool) {
	return readInt(buf, unsafe.Offsetof(syscall.Dirent{}.Ino), unsafe.Sizeof(syscall.Dirent{}.Ino))
}

func direntReclen(buf []byte) (uint64, bool) {
	return readInt(buf, unsafe.Offsetof(syscall.Dirent{}.Reclen), unsafe.Sizeof(syscall.Dirent{}.Reclen))
}

func direntNamlen(buf []byte) (uint64, bool) {
	reclen, ok := direntReclen(buf)
	if !ok {
		return 0, false
	}
	return reclen - uint64(unsafe.Offsetof(syscall.Dirent{}.Name)), true
}

func direntType(buf []byte) (uint64, bool) {
	return readInt(buf, unsafe.Offsetof(syscall.Dirent{}.Type), unsafe.Sizeof(syscall.Dirent{}.Type))
}
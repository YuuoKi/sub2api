//go:build linux

package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type linuxVideoAssetFilesystem struct {
	root             string
	afterRootBoundFn func()
}

func newPlatformVideoAssetFilesystem(root string) videoAssetFilesystem {
	return newLinuxVideoAssetFilesystem(root, nil)
}

func newLinuxVideoAssetFilesystem(root string, afterRootBound func()) videoAssetFilesystem {
	return &linuxVideoAssetFilesystem{root: strings.TrimSpace(root), afterRootBoundFn: afterRootBound}
}

func (f *linuxVideoAssetFilesystem) Supported() bool {
	return f != nil && f.root != ""
}

func (f *linuxVideoAssetFilesystem) Create(taskID int64) (*videoAssetTempFile, error) {
	if !f.Supported() || taskID <= 0 {
		return nil, ErrVideoAssetDownload
	}
	taskFD, err := f.openTaskDirectory(taskID, true)
	if err != nil {
		return nil, err
	}
	temporaryName, temporaryFD, err := createLinuxAssetTemp(taskFD)
	if err != nil {
		_ = unix.Close(taskFD)
		return nil, err
	}
	file := os.NewFile(uintptr(temporaryFD), "video-asset-temporary")
	committed := false
	directoryClosed := false
	closeDirectory := func() {
		if !directoryClosed {
			_ = unix.Close(taskFD)
			directoryClosed = true
		}
	}
	return &videoAssetTempFile{
		File: file,
		commitFn: func() error {
			if err := unix.Renameat2(taskFD, temporaryName, taskFD, "result.mp4", unix.RENAME_NOREPLACE); err != nil {
				return err
			}
			committed = true
			err := unix.Fsync(taskFD)
			closeDirectory()
			return err
		},
		abortFn: func() {
			_ = file.Close()
			if !committed {
				_ = unix.Unlinkat(taskFD, temporaryName, 0)
			}
			closeDirectory()
		},
	}, nil
}

func (f *linuxVideoAssetFilesystem) Open(taskID int64) (*os.File, error) {
	if !f.Supported() || taskID <= 0 {
		return nil, ErrVideoLocalAssetNotFound
	}
	taskFD, err := f.openTaskDirectory(taskID, false)
	if err != nil {
		return nil, err
	}
	assetFD, err := unix.Openat(taskFD, "result.mp4", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	_ = unix.Close(taskFD)
	if err != nil {
		return nil, err
	}
	var stat unix.Stat_t
	if err = unix.Fstat(assetFD, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG {
		_ = unix.Close(assetFD)
		if err != nil {
			return nil, err
		}
		return nil, ErrVideoLocalAssetNotFound
	}
	return os.NewFile(uintptr(assetFD), fmt.Sprintf("video-task-%d.mp4", taskID)), nil
}

func (f *linuxVideoAssetFilesystem) openTaskDirectory(taskID int64, create bool) (int, error) {
	rootFD, err := f.openRoot(create)
	if err != nil {
		return -1, err
	}
	if f.afterRootBoundFn != nil {
		f.afterRootBoundFn()
	}
	currentFD := rootFD
	for _, component := range []string{"assets", "video", strconv.FormatInt(taskID, 10)} {
		nextFD, openErr := openLinuxAssetDirectory(currentFD, component, create)
		_ = unix.Close(currentFD)
		if openErr != nil {
			return -1, openErr
		}
		currentFD = nextFD
	}
	return currentFD, nil
}

func (f *linuxVideoAssetFilesystem) openRoot(create bool) (int, error) {
	absolute, err := filepath.Abs(f.root)
	if err != nil || absolute == string(filepath.Separator) {
		return -1, ErrVideoAssetDownload
	}
	components := strings.Split(strings.TrimPrefix(filepath.Clean(absolute), string(filepath.Separator)), string(filepath.Separator))
	currentFD, err := unix.Open(string(filepath.Separator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	for index, component := range components {
		allowCreate := create && index == len(components)-1
		nextFD, openErr := openLinuxAssetDirectory(currentFD, component, allowCreate)
		_ = unix.Close(currentFD)
		if openErr != nil {
			return -1, openErr
		}
		currentFD = nextFD
	}
	return currentFD, nil
}

func openLinuxAssetDirectory(parentFD int, name string, create bool) (int, error) {
	flags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW
	fd, err := unix.Openat(parentFD, name, flags, 0)
	if errors.Is(err, unix.ENOENT) && create {
		if mkdirErr := unix.Mkdirat(parentFD, name, 0o750); mkdirErr != nil && !errors.Is(mkdirErr, unix.EEXIST) {
			return -1, mkdirErr
		}
		fd, err = unix.Openat(parentFD, name, flags, 0)
	}
	return fd, err
}

func createLinuxAssetTemp(taskFD int) (string, int, error) {
	for attempt := 0; attempt < 16; attempt++ {
		random := make([]byte, 12)
		if _, err := rand.Read(random); err != nil {
			return "", -1, err
		}
		name := ".result-" + hex.EncodeToString(random) + ".tmp"
		fd, err := unix.Openat(taskFD, name, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o640)
		if err == nil {
			return name, fd, nil
		}
		if !errors.Is(err, unix.EEXIST) {
			return "", -1, err
		}
	}
	return "", -1, errors.New("cannot allocate unique video asset temporary file")
}

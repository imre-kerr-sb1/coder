package provisionersdk

import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/xerrors"
)

const (
	// TemplateArchiveLimit represents the maximum size of a template in bytes.
	TemplateArchiveLimit = 1 << 20
)

func dirHasExt(dir string, ext string) (bool, error) {
	dirEnts, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, fi := range dirEnts {
		if strings.HasSuffix(fi.Name(), ext) {
			return true, nil
		}
	}

	return false, nil
}

// Tar archives a Terraform directory.
func Tar(directory string, limit int64) ([]byte, error) {
	var buffer bytes.Buffer
	tarWriter := tar.NewWriter(&buffer)
	totalSize := int64(0)

	const tfExt = ".tf"
	hasTf, err := dirHasExt(directory, tfExt)
	if err != nil {
		return nil, err
	}
	if !hasTf {
		absPath, err := filepath.Abs(directory)
		if err != nil {
			return nil, err
		}

		// Show absolute path to aid in debugging. E.g. showing "." is
		// useless.
		return nil, xerrors.Errorf(
			"%s is not a valid template since it has no %s files",
			absPath, tfExt,
		)
	}

	var walkFn filepath.WalkFunc
	walkFn = func(file string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
			// Per https://github.com/coder/coder/issues/5677, we want to
			// resolve symlinks.
			var linkDest string
			// Resolve symlinks.
			linkDest, err = os.Readlink(file)
			if err != nil {
				return err
			}

			destInfo, err := os.Stat(linkDest)
			if err != nil {
				return err
			}
			if destInfo.IsDir() {
				return filepath.Walk(linkDest, func(path string, info fs.FileInfo, err error) error {
					walkFn(path, info, err)
				})
			}
			return nil
		}

		header, err := tar.FileInfoHeader(fileInfo, "")
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(directory, file)
		if err != nil {
			return err
		}
		if strings.HasPrefix(rel, ".") || strings.HasPrefix(filepath.Base(rel), ".") {
			// Don't archive hidden files!
			if fileInfo.IsDir() && rel != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(rel, ".tfstate") {
			// Don't store tfstate!
			return nil
		}
		// Use unix paths in the tar archive.
		header.Name = filepath.ToSlash(rel)
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if !fileInfo.Mode().IsRegular() {
			return nil
		}
		data, err := os.Open(file)
		if err != nil {
			return err
		}
		defer data.Close()
		wrote, err := io.Copy(tarWriter, data)
		if err != nil {
			return err
		}
		totalSize += wrote
		if limit != 0 && totalSize >= limit {
			return xerrors.Errorf("Archive too big. Must be <= %d bytes", limit)
		}
		return data.Close()
	}

	err = filepath.Walk(directory, walkFn)
	if err != nil {
		return nil, err
	}
	err = tarWriter.Flush()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// Untar extracts the archive to a provided directory.
func Untar(directory string, archive []byte) error {
	reader := tar.NewReader(bytes.NewReader(archive))
	for {
		header, err := reader.Next()
		if xerrors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if header.Name == "." || strings.Contains(header.Name, "..") {
			continue
		}
		// #nosec
		target := filepath.Join(directory, filepath.FromSlash(header.Name))
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// Max file size of 10MB.
			_, err = io.CopyN(file, reader, (1<<20)*10)
			if xerrors.Is(err, io.EOF) {
				err = nil
			}
			if err != nil {
				return err
			}
			_ = file.Close()
		}
	}
}

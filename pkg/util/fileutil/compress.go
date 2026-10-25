package fileutil

import (
	"archive/zip"
	"fmt"
	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/errutil"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ZipToWriter(src string, dst io.Writer) error {
	zipWriter := zip.NewWriter(dst)
	defer log.Close(zipWriter)

	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		if d.IsDir() {
			_, err = zipWriter.Create(relPath + "/")
			return err
		}

		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer log.Close(file)

		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func Unzip(src, dst string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer log.Close(reader)

	err = os.MkdirAll(dst, 0755)
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		dstPath := filepath.Join(dst, file.Name)
		if !strings.HasPrefix(dstPath, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			err = os.MkdirAll(dstPath, file.Mode())
			if err != nil {
				return errutil.Wrap(err)
			}
			continue
		}

		err = os.MkdirAll(filepath.Dir(dstPath), 0755)
		if err != nil {
			return err
		}

		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		srcFile, err := file.Open()
		if err != nil {
			log.Close(dstFile)
			return err
		}

		_, err = io.Copy(dstFile, srcFile)

		log.Close(srcFile)
		log.Close(dstFile)

		if err != nil {
			return err
		}
	}

	return nil
}

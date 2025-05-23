package osx

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/sync/errgroup"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/ext/iox"
)

func WriteFileFromReaderAsync(ctx context.Context, path string, reader io.Reader, perm os.FileMode, wg *errgroup.Group) error {

	wg.Go(func() error {
		_, err := WriteFileFromReader(ctx, path, reader, perm)
		return err
	})

	return nil
}

func WriteFileFromReader(ctx context.Context, path string, reader io.Reader, perm os.FileMode) (int64, error) {

	// Don't close the reader - let the caller handle that
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return 0, errors.Errorf("creating file %s: %w", path, err)
	}
	defer file.Close()

	written, err := io.Copy(file, reader)
	if err != nil {
		return written, errors.Errorf("copying to file %s: %w", path, err)
	}

	return written, nil
}

func WriteFromReaderToFilePaths(ctx context.Context, reader io.Reader, paths ...string) error {

	if len(paths) == 0 {
		return nil
	}

	if len(paths) == 1 {
		_, err := WriteFileFromReader(ctx, paths[0], reader, 0644)
		return err
	}

	// Create all files upfront to fail early if any can't be created
	files := make([]*os.File, len(paths))
	for i, path := range paths {
		file, err := os.Create(path)
		if err != nil {
			// Close any files we managed to open
			for j := 0; j < i; j++ {
				files[j].Close()
			}
			return errors.Errorf("creating file %s: %w", path, err)
		}
		files[i] = file
	}

	// Ensure all files are closed when we're done
	defer func() {
		for _, file := range files {
			file.Close()
		}
	}()

	// Create a MultiWriter from all files
	writers := make([]io.Writer, len(files))
	for i, file := range files {
		writers[i] = file
	}
	mw := io.MultiWriter(writers...)

	// Handle context cancellation using a wrapper
	readWrapper := iox.NewContextReader(ctx, reader)

	_, err := io.Copy(mw, readWrapper)
	if err != nil {
		if ctx.Err() != nil {
			return errors.Errorf("operation cancelled: %w", ctx.Err())
		}
		return errors.Errorf("copying to multiple files: %w", err)
	}

	return nil
}

func CopyFile(ctx context.Context, src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return errors.Errorf("statting source file %s: %w", src, err)
	}

	srcfd, err := os.Open(src)
	if err != nil {
		return errors.Errorf("opening source file %s: %w", src, err)
	}
	defer srcfd.Close()

	dstfd, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return errors.Errorf("creating destination file %s: %w", dst, err)
	}
	defer dstfd.Close()

	readWrapper := iox.NewContextReader(ctx, srcfd)

	if _, err = io.Copy(dstfd, readWrapper); err != nil {
		if ctx.Err() != nil {
			slog.WarnContext(ctx, "copy file operation cancelled, the destination file will be deleted", "destination", dst)
			go os.Remove(dst)
			return errors.Errorf("operation cancelled: %w", ctx.Err())
		}
		return errors.Errorf("copying from %s to %s: %w", src, dst, err)
	}

	// Set the same modification time
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return errors.Errorf("setting file times for %s: %w", dst, err)
	}

	return nil
}

func Copy(ctx context.Context, src string, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return errors.Errorf("statting source %s: %w", src, err)
	}

	// Handle symlinks specially
	if (srcInfo.Mode() & os.ModeSymlink) != 0 {
		linkTarget, err := os.Readlink(src)
		if err != nil {
			return errors.Errorf("reading symlink %s: %w", src, err)
		}
		return os.Symlink(linkTarget, dst)
	}

	if !srcInfo.IsDir() {
		return CopyFile(ctx, src, dst)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return errors.Errorf("creating destination directory %s: %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return errors.Errorf("reading source directory %s: %w", src, err)
	}

	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return errors.Errorf("operation cancelled: %w", err)
	}

	for _, entry := range entries {
		srcPath := path.Join(src, entry.Name())
		dstPath := path.Join(dst, entry.Name())

		// Check for context cancellation periodically
		if err := ctx.Err(); err != nil {
			return errors.Errorf("operation cancelled: %w", err)
		}

		if entry.IsDir() {
			if err = Copy(ctx, srcPath, dstPath); err != nil {
				return errors.Errorf("copying directory %s: %w", srcPath, err)
			}
		} else {
			if err = CopyFile(ctx, srcPath, dstPath); err != nil {
				return errors.Errorf("copying file %s: %w", srcPath, err)
			}
		}
	}

	// Preserve directory modification time
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return errors.Errorf("setting directory times for %s: %w", dst, err)
	}

	return nil
}

// CopyFile efficiently copies a single file with zero‑copy, buffered fallback,
// context cancellation cleanup, and metadata preservation.

func RenameDir(ctx context.Context, src, dst string) error {
	// walk the directory and rename each file
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := RenameDir(ctx, srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := os.Rename(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func RenameDirFast(ctx context.Context, src string, dest string) error {

	// if dest exists, move it to a temporary location
	if _, err := os.Stat(dest); err == nil {
		// move the directory to a temporary location
		tempDir, err := os.MkdirTemp("", "ec1-tmp-trash-")
		if err != nil {
			return err
		}
		defer func() {
			go os.RemoveAll(tempDir)
		}()

		// move the directory to the temporary location
		if err := os.Rename(dest, filepath.Join(tempDir, filepath.Base(dest))); err != nil {
			return err
		}
	}

	// rename the directory
	if err := os.Rename(src, dest); err != nil {
		return err
	}

	return nil
}

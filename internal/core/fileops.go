package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
)

const (
	// maxRetries is the number of attempts for locked file operations.
	maxRetries = 3

	// baseBackoff is the initial retry delay.
	baseBackoff = 500 * time.Millisecond
)

// isRetryableError returns true if the error is a transient Windows file
// locking error that may succeed on retry.
func isRetryableError(err error) bool {
	var errno windows.Errno
	if errors.As(err, &errno) {
		switch errno {
		case windows.ERROR_SHARING_VIOLATION, // 32
			windows.ERROR_LOCK_VIOLATION: // 33
			return true
		}
	}
	return false
}

// isAccessDenied returns true if the error is an access-denied error.
func isAccessDenied(err error) bool {
	var errno windows.Errno
	if errors.As(err, &errno) {
		return errno == windows.ERROR_ACCESS_DENIED // 5
	}
	return os.IsPermission(err)
}

// SafeDelete removes a file or directory after safety validation.
// In dryRun mode, it calculates and returns the size without deleting.
// It retries up to 3 times with exponential backoff for locked files.
// Returns the number of bytes freed (or that would be freed).
func SafeDelete(path string, dryRun bool) (int64, error) {
	// Validate path through safety checks.
	if err := ValidatePath(path); err != nil {
		return 0, fmt.Errorf("safety check failed for %s: %w", path, err)
	}

	// Check if path exists.
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // Nothing to delete.
		}
		return 0, fmt.Errorf("cannot stat %s: %w", path, err)
	}

	// Calculate size.
	var size int64
	if info.IsDir() {
		size, err = GetDirSize(path)
		if err != nil {
			// Non-fatal: we can still attempt deletion.
			size = 0
		}
	} else {
		size = info.Size()
	}

	if dryRun {
		return size, nil
	}

	// Attempt deletion with retry.
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := baseBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		if info.IsDir() {
			lastErr = os.RemoveAll(path)
		} else {
			lastErr = os.Remove(path)
		}

		if lastErr == nil {
			return size, nil
		}

		// If it's a retryable error, try again.
		if isRetryableError(lastErr) {
			continue
		}

		// For access denied, try removing read-only attribute and retry.
		if isAccessDenied(lastErr) && !info.IsDir() {
			_ = os.Chmod(path, 0o666)
			continue
		}

		// Non-retryable error: bail out.
		break
	}

	return 0, fmt.Errorf("failed to delete %s after %d attempts: %w", path, maxRetries, lastErr)
}

// SafeCleanDir removes files matching a glob pattern within a directory.
// Returns total bytes freed and number of files deleted.
func SafeCleanDir(dir string, pattern string, dryRun bool) (int64, int, error) {
	if err := ValidatePath(dir); err != nil {
		return 0, 0, fmt.Errorf("safety check failed for %s: %w", dir, err)
	}

	// Verify directory exists.
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("cannot stat directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return 0, 0, fmt.Errorf("not a directory: %s", dir)
	}

	// Find matching files.
	globPattern := filepath.Join(dir, pattern)
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid glob pattern %s: %w", globPattern, err)
	}

	var totalBytes int64
	var totalFiles int

	for _, match := range matches {
		freed, delErr := SafeDelete(match, dryRun)
		if delErr != nil {
			// Log but continue — don't let one failure stop the whole batch.
			continue
		}
		totalBytes += freed
		totalFiles++
	}

	return totalBytes, totalFiles, nil
}

// GetDirSize calculates the total size of all files in a directory tree.
func GetDirSize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip files we can't access rather than aborting.
			return nil
		}
		if !d.IsDir() {
			info, infoErr := d.Info()
			if infoErr != nil {
				return nil
			}
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error walking directory %s: %w", path, err)
	}
	return total, nil
}

// GetFileSize returns the size of a single file.
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("cannot stat file %s: %w", path, err)
	}
	if info.IsDir() {
		return 0, fmt.Errorf("path is a directory, use GetDirSize: %s", path)
	}
	return info.Size(), nil
}

// FormatSize returns a human-readable representation of a byte count.
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

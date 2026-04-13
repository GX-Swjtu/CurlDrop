package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanOldFiles(t *testing.T) {
	dir := t.TempDir()

	// Create an old file (10 days ago)
	oldFile := filepath.Join(dir, "old.txt")
	os.WriteFile(oldFile, []byte("old"), 0644)
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	os.Chtimes(oldFile, oldTime, oldTime)

	// Create a recent file (now)
	recentFile := filepath.Join(dir, "recent.txt")
	os.WriteFile(recentFile, []byte("recent"), 0644)

	// Create a boundary file (just past the cutoff)
	boundaryFile := filepath.Join(dir, "boundary.txt")
	os.WriteFile(boundaryFile, []byte("boundary"), 0644)
	boundaryTime := time.Now().Add(-7*24*time.Hour - time.Minute)
	os.Chtimes(boundaryFile, boundaryTime, boundaryTime)

	cleanOldFiles(dir, 7)

	// old.txt should be deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old.txt should have been deleted")
	}

	// recent.txt should still exist
	if _, err := os.Stat(recentFile); err != nil {
		t.Error("recent.txt should still exist")
	}

	// boundary.txt should be deleted (it's older than 7 days)
	if _, err := os.Stat(boundaryFile); !os.IsNotExist(err) {
		t.Error("boundary.txt should have been deleted")
	}
}

func TestCleanOldFiles_SkipsDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create a subdirectory with old modification time
	subDir := filepath.Join(dir, "olddir")
	os.Mkdir(subDir, 0755)
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	os.Chtimes(subDir, oldTime, oldTime)

	cleanOldFiles(dir, 7)

	// Subdirectory should still exist
	if _, err := os.Stat(subDir); err != nil {
		t.Error("subdirectory should not be deleted by cleanOldFiles")
	}
}

func TestCleanOldFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	// Should not panic on empty directory
	cleanOldFiles(dir, 7)
}

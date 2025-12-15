package fsutil

// Note: All fsutil module tests have been reorganized into separate files:
//
// - atomic_test.go: Atomic file operation tests (AtomicWriteFile, SafeRename, syncDir)
// - safe_test.go: Safe operations tests (EnsureDir, FileExists, FileSize, CleanPath, AbsPath, RemoveAll, ListFiles, ListDirs)
// - walk_test.go: Directory walking tests (Walk, WalkFilesWithExt, FindFiles, DirSize, CountFiles, IsHidden, FilterHidden)
//
// This improves test organization and maintainability by following the single-responsibility principle.

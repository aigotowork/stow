package index

// Note: All index module tests have been reorganized into separate files:
//
// - cache_test.go: Cache functionality tests (Exists, Count, Keys, CleanupExpired, Stats, etc.)
// - mapper_test.go: KeyMapper tests (FindExact, RemoveByFileName, GetConflicts, Stats, String, etc.)
// - scanner_test.go: Scanner tests (ScanNamespace, ScanAndValidate, CountFiles, ListKeys, etc.)
// - sanitize_test.go: Key sanitization and utilities (SanitizeKey, GenerateFileName, ExtractKeyFromFileName, etc.)
//
// This improves test organization and maintainability by following the single-responsibility principle.

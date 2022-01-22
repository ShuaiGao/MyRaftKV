package datadir

import "path/filepath"

const (
	memberDirSegment   = "member"
	snapDirSegment     = "snap"
	walDirSegment      = "wal"
	backendFileSegment = "db"
)

func ToBackendFileName(dataDir string) string {
	return filepath.Join(ToSnapDir(dataDir), backendFileSegment)
}
func ToWalDir(dataDir string) string {
	return filepath.Join(ToMemberDir(dataDir), walDirSegment)
}

func ToSnapDir(dataDir string) string {
	return filepath.Join(ToMemberDir(dataDir), snapDirSegment)
}

func ToMemberDir(dataDir string) string {
	return filepath.Join(dataDir, memberDirSegment)
}

package files

// Package files will own file metadata, upload authorization, signed URLs,
// and object-storage integration.
//
// PostgreSQL stores metadata.
// Object storage stores file bytes.
//
// Modules attach files through this platform package instead of storing
// files independently.

package fs

import "fmt"

// KeyGen generates Redis key names for a given volume.
type KeyGen struct {
	Volume string
}

// NewKeyGen creates a KeyGen for the given volume.
func NewKeyGen(volume string) *KeyGen {
	return &KeyGen{Volume: volume}
}

// Meta returns the metadata key for a path.
// e.g., fs:main:meta:/configs/prod
func (k *KeyGen) Meta(path string) string {
	return fmt.Sprintf("fs:%s:meta:%s", k.Volume, path)
}

// Data returns the data key for a path.
// e.g., fs:main:data:/configs/prod/app.conf
func (k *KeyGen) Data(path string) string {
	return fmt.Sprintf("fs:%s:data:%s", k.Volume, path)
}

// Dir returns the directory set key for a path.
// e.g., fs:main:dir:/configs/prod
func (k *KeyGen) Dir(path string) string {
	return fmt.Sprintf("fs:%s:dir:%s", k.Volume, path)
}

// Xattr returns the extended attributes key for a path.
// e.g., fs:main:xattr:/configs/prod/app.conf
func (k *KeyGen) Xattr(path string) string {
	return fmt.Sprintf("fs:%s:xattr:%s", k.Volume, path)
}

// Idx returns the index key for a path.
// e.g., fs:main:idx:/configs/prod/app.conf
func (k *KeyGen) Idx(path string) string {
	return fmt.Sprintf("fs:%s:idx:%s", k.Volume, path)
}

// IdxPrefix returns the prefix for all index keys in this volume.
// e.g., fs:main:idx:
func (k *KeyGen) IdxPrefix() string {
	return fmt.Sprintf("fs:%s:idx:", k.Volume)
}

// IdxSchemaVersion returns the key storing the index schema version.
func (k *KeyGen) IdxSchemaVersion() string {
	return fmt.Sprintf("fs:%s:idx:__schema_ver__", k.Volume)
}

// VolumeRootPattern returns a SCAN pattern to discover all volumes.
// Matches fs:*:meta:/ to find volume root metadata keys.
func VolumeRootPattern() string {
	return "fs:*:meta:/"
}

package main

import (
	"os"
	"path/filepath"
)

// catude was called lazyrig before its first release. The first run under the new name moves the old
// folders (preferences, profiles, cache, backups, trash) to the new ones, once, when the new ones do not
// exist yet. Nothing else is touched.
func migrateLegacy() {
	if os.Getenv("CATUDE_HOME") != "" {
		return // tests and portable setups choose their own folder
	}
	var bases []string
	if d, err := os.UserConfigDir(); err == nil {
		bases = append(bases, d)
	}
	if d, err := os.UserCacheDir(); err == nil {
		bases = append(bases, d)
	}
	for _, b := range bases {
		old, now := filepath.Join(b, "lazyrig"), filepath.Join(b, "catude")
		if fi, err := os.Stat(old); err != nil || !fi.IsDir() {
			continue
		}
		if _, err := os.Stat(now); err == nil {
			continue
		}
		os.Rename(old, now)
	}
}

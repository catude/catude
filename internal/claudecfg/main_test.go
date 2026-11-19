package claudecfg

import (
	"os"
	"testing"
)

// Tests write backups (Backup puts them in the user cache folder): keep them out of the real one.
func TestMain(m *testing.M) {
	d, err := os.MkdirTemp("", "catude-claudecfg-test")
	if err != nil {
		panic(err)
	}
	for _, k := range []string{"LOCALAPPDATA", "XDG_CACHE_HOME", "HOME", "USERPROFILE", "APPDATA", "XDG_CONFIG_HOME"} {
		os.Setenv(k, d)
	}
	code := m.Run()
	os.RemoveAll(d)
	os.Exit(code)
}

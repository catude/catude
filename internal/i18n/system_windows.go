//go:build windows

package i18n

import "golang.org/x/sys/windows"

// systemLanguages are the user's preferred Windows display languages (ja-JP, zh-CN…).
func systemLanguages() []string {
	ls, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err != nil {
		return nil
	}
	return ls
}

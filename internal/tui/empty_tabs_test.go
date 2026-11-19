package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/i18n"
	"github.com/catude/catude/internal/paths"
)

// Every tab must draw before anything has loaded, in every language and at several sizes: the first
// frame after opening a tab comes before its data (a panic here kills the whole program).
func TestEveryTabDrawsWithoutData(t *testing.T) {
	defer i18n.Set("en")
	for _, lang := range []string{"en", "ja", "es"} {
		i18n.Set(lang)
		for _, size := range [][2]int{{140, 40}, {80, 24}, {40, 12}} {
			for i := range tabNames {
				name := fmt.Sprintf("%s/%dx%d/%s", lang, size[0], size[1], tabNames[i])
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("%s: panic: %v", name, r)
						}
					}()
					m := newModel(paths.Env{ClaudeDir: t.TempDir()}, "test")
					m.w, m.h = size[0], size[1]
					m.tab = tab(i)
					_ = m.View()
					m.loaded = true
					_ = m.View()
					// the usual keys on an empty tab
					for _, k := range []tea.KeyMsg{{Type: tea.KeyDown}, {Type: tea.KeyUp}, {Type: tea.KeyEnter}, {Type: tea.KeyPgDown}} {
						nm, _ := m.Update(k)
						m = nm.(model)
						_ = m.View()
					}
				}()
			}
		}
	}
}

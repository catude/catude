package tui

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// desktopNotify shows a system notification without waiting for it. The text goes through environment
// variables, never into the command line or a script, so a session name cannot inject anything.
var desktopNotify = func(title, body string) tea.Cmd {
	return func() tea.Msg {
		title, body = oneLine(title), oneLine(body)
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			ps := `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; ` +
				`$t = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02); ` +
				`$x = $t.GetElementsByTagName('text'); $x.Item(0).AppendChild($t.CreateTextNode($env:CATUDE_N_TITLE)) > $null; ` +
				`$x.Item(1).AppendChild($t.CreateTextNode($env:CATUDE_N_BODY)) > $null; ` +
				`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe').Show([Windows.UI.Notifications.ToastNotification]::new($t))`
			cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
		case "darwin":
			cmd = exec.Command("osascript", "-e", `display notification (system attribute "CATUDE_N_BODY") with title (system attribute "CATUDE_N_TITLE")`)
		default:
			if _, err := exec.LookPath("notify-send"); err != nil {
				return nil
			}
			cmd = exec.Command("notify-send", "--app-name=catude", "--", title, body)
		}
		cmd.Env = append(os.Environ(), "CATUDE_N_TITLE="+title, "CATUDE_N_BODY="+body)
		_ = cmd.Start()
		go cmd.Wait()
		return nil
	}
}

func oneLine(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
	if r := []rune(s); len(r) > 200 {
		s = string(r[:199]) + "…"
	}
	return s
}

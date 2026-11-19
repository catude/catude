package tui

import "regexp"

// ansiRe matches SGR colour sequences so a selected row can be re-rendered without nested styles.
var ansiRe = regexp.MustCompile("\x1b" + `\[[0-9;]*m`)

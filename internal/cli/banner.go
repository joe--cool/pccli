package cli

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

//go:embed banner.ansi
var bannerANSI string

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func printBanner(out io.Writer) {
	colorEnabled := bannerColorEnabled(out)
	banner := strings.Trim(bannerANSI, "\n")
	if !colorEnabled {
		banner = ansiPattern.ReplaceAllString(banner, "")
	}
	banner = strings.Trim(banner, "\n")
	if colorEnabled {
		banner += "\x1b[0m"
	}
	banner += "\n\n"
	_, _ = fmt.Fprint(out, banner)
}

func bannerColorEnabled(out io.Writer) bool {
	switch colorMode() {
	case "always":
		return true
	case "never":
		return false
	default:
		if os.Getenv("NO_COLOR") != "" {
			return false
		}
		file, ok := out.(*os.File)
		return ok && term.IsTerminal(int(file.Fd()))
	}
}

func colorMode() string {
	if value := strings.ToLower(os.Getenv("PCCLI_COLOR")); value != "" {
		return value
	}
	if value := strings.ToLower(os.Getenv("PCO_COLOR")); value != "" {
		return value
	}
	return "auto"
}

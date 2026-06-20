package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

type Renderer struct {
	Out   io.Writer
	JSON  bool
	color bool
}

func NewRenderer(out io.Writer, jsonOutput bool, colorMode string) Renderer {
	color := colorEnabled(out, colorMode)
	if color {
		lipgloss.SetColorProfile(termenv.TrueColor)
	}
	return Renderer{
		Out:   out,
		JSON:  jsonOutput,
		color: color,
	}
}

func (r Renderer) WriteJSON(value any) error {
	encoder := json.NewEncoder(r.Out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func (r Renderer) Table(headers []string, rows [][]string) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(r.Out, "No results.")
		return err
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		Headers(headers...).
		Rows(rows...)

	if r.color {
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2465F5"))
		t.StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle.PaddingRight(2)
			}
			return lipgloss.NewStyle().PaddingRight(2)
		})
	} else {
		t.StyleFunc(func(row, col int) lipgloss.Style {
			return lipgloss.NewStyle().PaddingRight(2)
		})
	}

	_, err := fmt.Fprintln(r.Out, t.String())
	return err
}

func (r Renderer) KeyValues(rows [][2]string) error {
	if r.JSON {
		values := map[string]string{}
		for _, row := range rows {
			values[row[0]] = row[1]
		}
		return r.WriteJSON(values)
	}

	width := 0
	for _, row := range rows {
		if len(row[0]) > width {
			width = len(row[0])
		}
	}
	for _, row := range rows {
		label := row[0]
		if r.color {
			label = lipgloss.NewStyle().Bold(true).Render(label)
		}
		if _, err := fmt.Fprintf(r.Out, "%-*s  %s\n", width, label, row[1]); err != nil {
			return err
		}
	}
	return nil
}

func Int(value int) string {
	if value == 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func Float(value float64) string {
	if value == 0 {
		return ""
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

func DurationSeconds(seconds int) string {
	if seconds == 0 {
		return ""
	}
	minutes := seconds / 60
	remainder := seconds % 60
	return fmt.Sprintf("%d:%02d", minutes, remainder)
}

func colorEnabled(out io.Writer, mode string) bool {
	switch mode {
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

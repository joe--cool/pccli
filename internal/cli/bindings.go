package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Bindings preserve flag presence instead of confusing defaults with updates.
type bindings struct{ steps []func(*cobra.Command) error }

func newBindings() *bindings { return &bindings{} }
func (b *bindings) apply(c *cobra.Command) error {
	for _, f := range b.steps {
		if e := f(c); e != nil {
			return usageError{e}
		}
	}
	return nil
}
func (b *bindings) str(c *cobra.Command, name, help string, dst **string) {
	v := c.Flags().String(name, "", help)
	b.steps = append(b.steps, func(c *cobra.Command) error {
		if c.Flags().Changed(name) {
			*dst = v
		}
		return nil
	})
}
func (b *bindings) integer(c *cobra.Command, name, help string, dst **int) {
	v := c.Flags().Int(name, 0, help)
	b.steps = append(b.steps, func(c *cobra.Command) error {
		if c.Flags().Changed(name) {
			*dst = v
		}
		return nil
	})
}
func (b *bindings) float(c *cobra.Command, name, help string, dst **float64) {
	v := c.Flags().Float64(name, 0, help)
	b.steps = append(b.steps, func(c *cobra.Command) error {
		if c.Flags().Changed(name) {
			*dst = v
		}
		return nil
	})
}
func (b *bindings) boolean(c *cobra.Command, name, help string, dst **bool) {
	v := c.Flags().Bool(name, false, help)
	b.steps = append(b.steps, func(c *cobra.Command) error {
		if c.Flags().Changed(name) {
			*dst = v
		}
		return nil
	})
}
func (b *bindings) sequence(c *cobra.Command, dst **[]string) {
	v := c.Flags().StringSlice("sequence", nil, "full section labels, e.g. 'Verse 1,Chorus 1'; empty clears")
	b.steps = append(b.steps, func(c *cobra.Command) error {
		if c.Flags().Changed("sequence") {
			x := *v
			if x == nil {
				x = []string{}
			}
			*dst = &x
		}
		return nil
	})
}
func (b *bindings) length(c *cobra.Command, dst **int) {
	v := c.Flags().String("length", "", "length as seconds or m:ss; 0 clears")
	b.steps = append(b.steps, func(c *cobra.Command) error {
		if c.Flags().Changed("length") {
			n, e := parseLengthSeconds(*v)
			if e != nil {
				return e
			}
			*dst = &n
		}
		return nil
	})
}
func parseLengthSeconds(v string) (int, error) {
	parts := strings.Split(v, ":")
	if len(parts) > 2 || v == "" {
		return 0, fmt.Errorf("invalid --length: use seconds or m:ss")
	}
	n, e := strconv.ParseInt(parts[0], 10, 32)
	if e != nil || n < 0 {
		return 0, fmt.Errorf("invalid --length: use nonnegative seconds or m:ss")
	}
	if len(parts) == 2 {
		sec, e := strconv.ParseInt(parts[1], 10, 32)
		if e != nil || sec < 0 || sec > 59 || n > (2147483647-sec)/60 {
			return 0, fmt.Errorf("invalid or overflowing --length")
		}
		n = n*60 + sec
	}
	return int(n), nil
}
func bindSong(c *cobra.Command, b *bindings, admin, author, copyright, themes **string, ccli **int, hidden **bool) {
	b.str(c, "admin", "song administrator", admin)
	b.str(c, "author", "author; empty clears", author)
	b.str(c, "copyright", "copyright text; empty clears", copyright)
	b.str(c, "themes", "themes text; use tags commands for relational tags", themes)
	b.integer(c, "ccli", "positive CCLI number", ccli)
	b.boolean(c, "hidden", "set hidden state (selection uses --visibility)", hidden)
}
func bindArrangement(c *cobra.Command, b *bindings, bpm **float64, meter, key, notes **string, length **int, lyrics **bool, sequence **[]string) {
	b.float(c, "bpm", "positive beats per minute", bpm)
	b.str(c, "meter", "meter, e.g. 4/4", meter)
	b.str(c, "chart-key", "chart pitch, e.g. G or Cm", key)
	b.str(c, "notes", "arrangement notes; empty clears", notes)
	b.length(c, length)
	b.boolean(c, "lyrics-enabled", "set lyrics enabled state", lyrics)
	b.sequence(c, sequence)
}

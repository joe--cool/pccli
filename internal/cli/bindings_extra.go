package cli

import (
	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

// Additional documented metadata stays explicit and typed; no raw attribute passthrough.
func bindExtraFields(c *cobra.Command, b *bindings, input any) {
	switch m := input.(type) {
	case *services.SongCreate:
		b.str(c, "import-ccli-chord-chart", "import ccli chord chart (see field reference)", &m.ImportCCLIChordChart)
		b.str(c, "import-ccli-chord-pro", "import ccli chord pro (see field reference)", &m.ImportCCLIChordPro)
		b.str(c, "import-ccli-lyrics", "import ccli lyrics (see field reference)", &m.ImportCCLILyrics)
	case *services.ArrangementCreate:
		b.str(c, "chord-chart", "chord chart (see field reference)", &m.ChordChart)
		b.integer(c, "chord-chart-chord-color", "chord chart chord color (see field reference)", &m.ChordChartChordColor)
		b.integer(c, "chord-chart-columns", "chord chart columns (see field reference)", &m.ChordChartColumns)
		b.str(c, "chord-chart-font", "chord chart font (see field reference)", &m.ChordChartFont)
		b.integer(c, "chord-chart-font-size", "chord chart font size (see field reference)", &m.ChordChartFontSize)
		b.str(c, "import-chart-pro", "import chart pro (see field reference)", &m.ImportChartPro)
		b.str(c, "import-chart-pro-options", "import chart pro options (see field reference)", &m.ImportChartProOptions)
		b.str(c, "import-rehearsal-mix", "import rehearsal mix (see field reference)", &m.ImportRehearsalMix)
		b.str(c, "import-rehearsal-mix-options", "import rehearsal mix options (see field reference)", &m.ImportRehearsalMixOptions)
		b.str(c, "import-rehearsal-pack", "import rehearsal pack (see field reference)", &m.ImportRehearsalPack)
		b.str(c, "import-rehearsal-pack-options", "import rehearsal pack options (see field reference)", &m.ImportRehearsalPackOptions)
		b.integer(c, "isrc", "isrc (see field reference)", &m.ISRC)
		b.str(c, "mtid", "mtid (see field reference)", &m.MTID)
		b.boolean(c, "number-chart-enabled", "number chart enabled (see field reference)", &m.NumberChartEnabled)
		b.boolean(c, "numeral-chart-enabled", "numeral chart enabled (see field reference)", &m.NumeralChartEnabled)
		b.str(c, "print-margin", "print margin (see field reference)", &m.PrintMargin)
		b.str(c, "print-orientation", "print orientation (see field reference)", &m.PrintOrientation)
		b.str(c, "print-page-size", "print page size (see field reference)", &m.PrintPageSize)
		b.str(c, "rehearsal-mix-id", "rehearsal mix id (see field reference)", &m.RehearsalMixID)
		b.str(c, "rehearsal-mix-thumbnail", "rehearsal mix thumbnail (see field reference)", &m.RehearsalMixThumbnail)
		b.str(c, "import-ccli-chord-chart", "import ccli chord chart (see field reference)", &m.ImportCCLIChordChart)
		b.str(c, "import-ccli-chord-pro", "import ccli chord pro (see field reference)", &m.ImportCCLIChordPro)
		b.str(c, "import-ccli-lyrics", "import ccli lyrics (see field reference)", &m.ImportCCLILyrics)
	case *services.ArrangementPatch:
		b.str(c, "chord-chart", "chord chart (see field reference)", &m.ChordChart)
		b.integer(c, "chord-chart-chord-color", "chord chart chord color (see field reference)", &m.ChordChartChordColor)
		b.integer(c, "chord-chart-columns", "chord chart columns (see field reference)", &m.ChordChartColumns)
		b.str(c, "chord-chart-font", "chord chart font (see field reference)", &m.ChordChartFont)
		b.integer(c, "chord-chart-font-size", "chord chart font size (see field reference)", &m.ChordChartFontSize)
		b.str(c, "import-chart-pro", "import chart pro (see field reference)", &m.ImportChartPro)
		b.str(c, "import-chart-pro-options", "import chart pro options (see field reference)", &m.ImportChartProOptions)
		b.str(c, "import-rehearsal-mix", "import rehearsal mix (see field reference)", &m.ImportRehearsalMix)
		b.str(c, "import-rehearsal-mix-options", "import rehearsal mix options (see field reference)", &m.ImportRehearsalMixOptions)
		b.str(c, "import-rehearsal-pack", "import rehearsal pack (see field reference)", &m.ImportRehearsalPack)
		b.str(c, "import-rehearsal-pack-options", "import rehearsal pack options (see field reference)", &m.ImportRehearsalPackOptions)
		b.integer(c, "isrc", "isrc (see field reference)", &m.ISRC)
		b.str(c, "mtid", "mtid (see field reference)", &m.MTID)
		b.boolean(c, "number-chart-enabled", "number chart enabled (see field reference)", &m.NumberChartEnabled)
		b.boolean(c, "numeral-chart-enabled", "numeral chart enabled (see field reference)", &m.NumeralChartEnabled)
		b.str(c, "print-margin", "print margin (see field reference)", &m.PrintMargin)
		b.str(c, "print-orientation", "print orientation (see field reference)", &m.PrintOrientation)
		b.str(c, "print-page-size", "print page size (see field reference)", &m.PrintPageSize)
		b.str(c, "rehearsal-mix-id", "rehearsal mix id (see field reference)", &m.RehearsalMixID)
		b.str(c, "rehearsal-mix-thumbnail", "rehearsal mix thumbnail (see field reference)", &m.RehearsalMixThumbnail)
	case *services.KeyCreate:
		b.str(c, "alternate-keys", "alternate keys (see field reference)", &m.AlternateKeys)
		b.str(c, "import-ccli-chord-chart", "import ccli chord chart (see field reference)", &m.ImportCCLIChordChart)
		b.str(c, "import-ccli-chord-chart-options", "import ccli chord chart options (see field reference)", &m.ImportCCLIChordChartOptions)
		b.str(c, "import-ccli-lyrics", "import ccli lyrics (see field reference)", &m.ImportCCLILyrics)
		b.str(c, "import-chart-pro", "import chart pro (see field reference)", &m.ImportChartPro)
		b.str(c, "import-rehearsal-mix", "import rehearsal mix (see field reference)", &m.ImportRehearsalMix)
		b.str(c, "import-rehearsal-pack", "import rehearsal pack (see field reference)", &m.ImportRehearsalPack)
	case *services.KeyPatch:
		b.str(c, "alternate-keys", "alternate keys (see field reference)", &m.AlternateKeys)
		b.str(c, "import-ccli-chord-chart", "import ccli chord chart (see field reference)", &m.ImportCCLIChordChart)
		b.str(c, "import-ccli-chord-chart-options", "import ccli chord chart options (see field reference)", &m.ImportCCLIChordChartOptions)
		b.str(c, "import-ccli-lyrics", "import ccli lyrics (see field reference)", &m.ImportCCLILyrics)
		b.str(c, "import-chart-pro", "import chart pro (see field reference)", &m.ImportChartPro)
		b.str(c, "import-rehearsal-mix", "import rehearsal mix (see field reference)", &m.ImportRehearsalMix)
		b.str(c, "import-rehearsal-pack", "import rehearsal pack (see field reference)", &m.ImportRehearsalPack)
	}
}

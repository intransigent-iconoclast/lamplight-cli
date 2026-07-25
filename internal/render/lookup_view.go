package render

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/service"
	utils "github.com/intransigent-iconoclast/lamplight-cli/pkg/util"
)

const (
	glyphOwned   = "✓"
	glyphMissing = "○"
)

type Options struct {
	Plain bool
	Width int
}

// Color auto-disables when w isn't a color-capable terminal, so the same path
// is safe for pipes, files and tests.
func Lookup(w io.Writer, res *service.LookupResult, opts Options) {
	width := opts.Width
	if width <= 0 {
		width = 100
	}

	titleW := width - 22
	if titleW < 20 {
		titleW = 20
	}
	if titleW > 70 {
		titleW = 70
	}

	if opts.Plain {
		renderPlain(w, res, titleW)
		return
	}
	renderRich(w, res, titleW)
}

func renderRich(w io.Writer, res *service.LookupResult, titleW int) {
	r := lipgloss.NewRenderer(w)

	nameStyle := r.NewStyle().Bold(true)
	bioStyle := r.NewStyle().Faint(true)
	card := r.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1)

	header := nameStyle.Render(res.Author.Name)
	if bio := strings.TrimSpace(res.Author.Bio); bio != "" {
		header += "\n" + bioStyle.Render(utils.SmartTruncate(bio, titleW+10))
	}
	fmt.Fprintln(w, card.Render(header))
	fmt.Fprintln(w)

	seriesStyle := r.NewStyle().Bold(true).Foreground(lipgloss.Color("75"))
	idxStyle := r.NewStyle().Faint(true)
	yearStyle := r.NewStyle().Faint(true)
	ratingStyle := r.NewStyle().Foreground(lipgloss.Color("220"))
	ownedGlyph := r.NewStyle().Foreground(lipgloss.Color("42")).Render(glyphOwned)
	missingRow := r.NewStyle().Faint(true)

	idx := 0
	row := func(b dao.Book) {
		idx++
		idxCol := padLeft(strconv.Itoa(idx), 3)
		titleCol := padRight(utils.SmartTruncate(utils.CleanString(b.Title), titleW), titleW)
		yearCol := yearStr(b.Year)
		rateCol := ratingStr(b.Rating)

		if b.Owned {
			fmt.Fprintf(w, "  %s %s  %s  %s  %s\n",
				ownedGlyph, idxStyle.Render(idxCol), titleCol,
				yearStyle.Render(yearCol), ratingStyle.Render(rateCol))
			return
		}
		// dim the whole missing row in one pass to avoid nested-ANSI breakage
		plain := fmt.Sprintf("%s %s  %s  %s  %s", glyphMissing, idxCol, titleCol, yearCol, rateCol)
		fmt.Fprintf(w, "  %s\n", missingRow.Render(plain))
	}

	for _, s := range res.Series {
		fmt.Fprintln(w, " "+seriesStyle.Render(s.Name))
		for _, b := range s.Books {
			row(b)
		}
		fmt.Fprintln(w)
	}
	if len(res.Standalone) > 0 {
		if len(res.Series) > 0 {
			fmt.Fprintln(w, " "+seriesStyle.Render("Other books"))
		}
		for _, b := range res.Standalone {
			row(b)
		}
		fmt.Fprintln(w)
	}

	legend := fmt.Sprintf(" %s owned   %s missing", ownedGlyph, glyphMissing)
	hint := r.NewStyle().Faint(true).Render("lamplight lookup --get <n>  →  search & download")
	fmt.Fprintf(w, "%s    %s\n", legend, hint)
}

func renderPlain(w io.Writer, res *service.LookupResult, titleW int) {
	fmt.Fprintln(w, res.Author.Name)
	if bio := strings.TrimSpace(res.Author.Bio); bio != "" {
		fmt.Fprintln(w, utils.SmartTruncate(bio, titleW+10))
	}
	fmt.Fprintln(w)

	idx := 0
	row := func(b dao.Book) {
		idx++
		glyph := glyphMissing
		if b.Owned {
			glyph = glyphOwned
		}
		title := padRight(utils.SmartTruncate(utils.CleanString(b.Title), titleW), titleW)
		fmt.Fprintf(w, "  %s %s  %s  %s  %s\n",
			glyph, padLeft(strconv.Itoa(idx), 3), title, yearStr(b.Year), ratingStr(b.Rating))
	}

	for _, s := range res.Series {
		fmt.Fprintln(w, s.Name)
		for _, b := range s.Books {
			row(b)
		}
		fmt.Fprintln(w)
	}
	if len(res.Standalone) > 0 {
		if len(res.Series) > 0 {
			fmt.Fprintln(w, "Other books")
		}
		for _, b := range res.Standalone {
			row(b)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "%s owned  %s missing\n", glyphOwned, glyphMissing)
}

// Books renders a flat, numbered book-search result list. Unlike Lookup (which
// is one author's catalog), these span many authors, so the author name is
// shown per row — a common-word query blends title- and author-matches, and the
// author column is what makes that legible.
func Books(w io.Writer, books []dao.Book, opts Options) {
	width := opts.Width
	if width <= 0 {
		width = 100
	}
	titleW := (width - 24) / 2
	if titleW < 18 {
		titleW = 18
	}
	if titleW > 50 {
		titleW = 50
	}
	authorW := titleW
	if opts.Plain {
		renderBooksPlain(w, books, titleW, authorW)
		return
	}
	renderBooksRich(w, books, titleW, authorW)
}

func firstAuthor(b dao.Book) string {
	if len(b.Authors) == 0 {
		return ""
	}
	return b.Authors[0]
}

func renderBooksRich(w io.Writer, books []dao.Book, titleW, authorW int) {
	r := lipgloss.NewRenderer(w)
	idxStyle := r.NewStyle().Faint(true)
	authorStyle := r.NewStyle().Foreground(lipgloss.Color("75"))
	yearStyle := r.NewStyle().Faint(true)
	ratingStyle := r.NewStyle().Foreground(lipgloss.Color("220"))
	ownedGlyph := r.NewStyle().Foreground(lipgloss.Color("42")).Render(glyphOwned)
	missingRow := r.NewStyle().Faint(true)

	for i, b := range books {
		idxCol := padLeft(strconv.Itoa(i+1), 3)
		titleCol := padRight(utils.SmartTruncate(utils.CleanString(b.Title), titleW), titleW)
		authorCol := padRight(utils.SmartTruncate(firstAuthor(b), authorW), authorW)
		yearCol := yearStr(b.Year)
		rateCol := ratingStr(b.Rating)

		if b.Owned {
			fmt.Fprintf(w, "  %s %s  %s  %s  %s  %s\n",
				ownedGlyph, idxStyle.Render(idxCol), titleCol,
				authorStyle.Render(authorCol), yearStyle.Render(yearCol), ratingStyle.Render(rateCol))
			continue
		}
		plain := fmt.Sprintf("%s %s  %s  %s  %s  %s", glyphMissing, idxCol, titleCol, authorCol, yearCol, rateCol)
		fmt.Fprintf(w, "  %s\n", missingRow.Render(plain))
	}
	fmt.Fprintln(w)

	legend := fmt.Sprintf(" %s owned   %s missing", ownedGlyph, glyphMissing)
	hint := r.NewStyle().Faint(true).Render("lamplight lookup --get <n>  →  search & download")
	fmt.Fprintf(w, "%s    %s\n", legend, hint)
}

func renderBooksPlain(w io.Writer, books []dao.Book, titleW, authorW int) {
	for i, b := range books {
		glyph := glyphMissing
		if b.Owned {
			glyph = glyphOwned
		}
		titleCol := padRight(utils.SmartTruncate(utils.CleanString(b.Title), titleW), titleW)
		authorCol := padRight(utils.SmartTruncate(firstAuthor(b), authorW), authorW)
		fmt.Fprintf(w, "  %s %s  %s  %s  %s  %s\n",
			glyph, padLeft(strconv.Itoa(i+1), 3), titleCol, authorCol, yearStr(b.Year), ratingStr(b.Rating))
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s owned  %s missing\n", glyphOwned, glyphMissing)
}

func yearStr(year int) string {
	if year <= 0 {
		return "    "
	}
	return strconv.Itoa(year)
}

func ratingStr(rating float64) string {
	if rating <= 0 {
		return ""
	}
	return fmt.Sprintf("★%.1f", rating)
}

func padRight(s string, w int) string {
	if gap := w - lipgloss.Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

func padLeft(s string, w int) string {
	if gap := w - lipgloss.Width(s); gap > 0 {
		return strings.Repeat(" ", gap) + s
	}
	return s
}

package utils

import (
	"fmt"
	"html"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"golang.org/x/term"
)

type HeaderRow string

const (
	DOWNLOADER_SAFE   HeaderRow = "INDEX\tNAME\tCLIENT-TYPE\tSCHEME\tHOST\tPORT\tBASE-URL\tUSERNAME\tLABEL\tPRIORITY"
	DOWNLOADER_UNSAFE HeaderRow = "INDEX\tNAME\tCLIENT-TYPE\tSCHEME\tHOST\tPORT\tBASE-URL\tUSERNAME\tPASSWORD\tLABEL\tPRIORITY"
	INDEXER_SAFE      HeaderRow = "INDEX\tNAME\tBASE_URL\tTYPE\tENABLED\tPRIORITY"
	INDEXER_UNSAFE    HeaderRow = "INDEX\tNAME\tBASE_URL\tTYPE\tENABLED\tAPI_KEY\tPRIORITY"
	SEARCH_RESULTS    HeaderRow = "INDEX\tTITLE\tFORMAT\tINDEXER\tSIZE_MB\tSEEDERS\tLEECHERS"
	PROVIDER_SAFE     HeaderRow = "INDEX\tNAME\tTYPE\tHOST\tPORT\tSCHEME\tENABLED"
	PROVIDER_UNSAFE   HeaderRow = "INDEX\tNAME\tTYPE\tHOST\tPORT\tSCHEME\tAPI_KEY\tENABLED"
	HISTORY           HeaderRow = "INDEX\tTITLE\tINDEXER\tCLIENT\tSIZE_MB\tDOWNLOADED_AT"
)

// https://stackoverflow.com/questions/71628061/difference-between-any-interface-as-constraint-vs-type-of-argument
// This function works by taking a lambda that converts the type to a list of strings
func PrintOutput[T any](out io.Writer, headers string, data []T, row func(T) []string) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	// print out the headers
	fmt.Fprintln(w, headers)

	for i, it := range data {
		// print out the row utilizing the "row" function to convert the type to a []string
		fmt.Fprintln(w, strconv.Itoa(i+1)+"\t"+strings.Join(row(it), "\t"))
	}

	return w.Flush()
}

// falls back to 120 if we can't figure it out
func TerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 120
	}
	return w
}

// cuts at a word boundary so it doesn't look janky
func SmartTruncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	if i := strings.LastIndex(cut, " "); i > max/2 {
		cut = cut[:i]
	}
	return cut + "..."
}

// strips _prowlarr / _jackett suffix — it's implied, no need to repeat it every row
func CleanIndexerName(name string) string {
	for _, suffix := range []string{"_prowlarr", "_jackett"} {
		name = strings.TrimSuffix(name, suffix)
	}
	return name
}

// Converts the file size from indexers (shown as number of bytes) to mb.
func BytesToMb(i int) string {
	f := math.Round((float64(i)/1_000_000.0)*100) / 100
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// This function takes a string and cleans up the text. Seems to be url encoded for some reason and mostly decoded.
// This is not super efficient but :shrug:
func CleanString(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.ReplaceAll(s, " 039 ", "'")
	s = strings.ReplaceAll(s, " amp ", "&")
	s = strings.ReplaceAll(s, " '", "'")
	s = strings.ReplaceAll(s, "' ", "'")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

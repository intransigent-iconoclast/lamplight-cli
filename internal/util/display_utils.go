package utils

import (
	"fmt"
	"html"
	"io"
	"math"
	"strconv"
	"strings"
	"text/tabwriter"
)

type HeaderRow string

const (
	DOWNLOADER_SAFE   HeaderRow = "INDEX\tNAME\tCLIENT-TYPE\tSCHEME\tHOST\tPORT\tBASE-URL\tUSERNAME\tLABEL\tPRIORITY"
	DOWNLOADER_UNSAFE HeaderRow = "INDEX\tNAME\tCLIENT-TYPE\tSCHEME\tHOST\tPORT\tBASE-URL\tUSERNAME\tPASSWORD\tLABEL\tPRIORITY"
	INDEXER_SAFE      HeaderRow = "INDEX\tNAME\tBASE_URL\tTYPE\tENABLED\tPRIORITY"
	INDEXER_UNSAFE    HeaderRow = "INDEX\tNAME\tBASE_URL\tTYPE\tENABLED\tAPI_KEY\tPRIORITY"
	SEARCH_RESULTS    HeaderRow = "INDEX\tTITLE\tINDEXER\tSIZE_MB\tSEEDERS\tLEECHERS"
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

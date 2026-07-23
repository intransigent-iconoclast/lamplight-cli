package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/service"
	"github.com/stretchr/testify/assert"
)

func sampleResult() *service.LookupResult {
	return &service.LookupResult{
		Author: dao.Author{Name: "Becky Chambers", Bio: "American SF author."},
		Series: []service.Series{{
			Name: "Wayfarers",
			Books: []dao.Book{
				{Title: "The Long Way to a Small, Angry Planet", Year: 2014, Rating: 4.3, Owned: true},
				{Title: "Record of a Spaceborn Few", Year: 2018, Rating: 4.0},
			},
		}},
		Standalone: []dao.Book{
			{Title: "To Be Taught, If Fortunate", Year: 2019, Owned: true},
		},
	}
}

func TestRenderPlain(t *testing.T) {
	var buf bytes.Buffer
	Lookup(&buf, sampleResult(), Options{Plain: true, Width: 100})
	out := buf.String()

	// no ANSI escapes in plain mode
	assert.NotContains(t, out, "\x1b[")

	assert.Contains(t, out, "Becky Chambers")
	assert.Contains(t, out, "Wayfarers")
	assert.Contains(t, out, "Other books")
	// owned/missing glyphs and global index numbering
	assert.Contains(t, out, glyphOwned+"   1  The Long Way")
	assert.Contains(t, out, glyphMissing+"   2  Record of a Spaceborn Few")
	assert.Contains(t, out, glyphOwned+"   3  To Be Taught")
	assert.Contains(t, out, "★4.3")
}

func TestRenderRichDisablesColorForNonTTY(t *testing.T) {
	// a bytes.Buffer isn't a terminal, so lipgloss must not emit color codes.
	var buf bytes.Buffer
	Lookup(&buf, sampleResult(), Options{Width: 100})
	out := buf.String()

	assert.NotContains(t, out, "\x1b[", "color must be off when writing to a non-terminal")
	assert.True(t, strings.Contains(out, "Becky Chambers"))
	assert.Contains(t, out, "owned")
	assert.Contains(t, out, "missing")
}

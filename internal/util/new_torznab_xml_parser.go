package utils

import (
	"encoding/xml"
	"strconv"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
)

type TorznabRSS struct {
	XMLName xml.Name       `xml:"rss"`
	Channel TorznabChannel `xml:"channel"`
}

type TorznabChannel struct {
	IndexerTitle string        `xml:"title"`
	Items        []TorznabItem `xml:"item"`
}

type TorznabAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type TorznabItem struct {
	Title      string        `xml:"title"`
	Size       int           `xml:"size"`
	Files      int           `xml:"files"`
	Link       string        `xml:"link"`
	Categories []int         `xml:"category"`
	Attrs      []TorznabAttr `xml:"attr"`
}

func ParseTorznabXml(xmlStr string) ([]dao.SearchResult, error) {
	xmlStr = strings.TrimSpace(xmlStr)
	if xmlStr == "" {
		return []dao.SearchResult{}, nil
	}

	var doc TorznabRSS
	err := xml.Unmarshal([]byte(xmlStr), &doc)
	if err != nil {
		return nil, err
	}

	results := make([]dao.SearchResult, 0, len(doc.Channel.Items))

	for _, it := range doc.Channel.Items {
		var (
			sizeBytes  *int64
			seeders    *int
			leechers   *int
			formatAttr string
		)

		if it.Size > 0 {
			sz := int64(it.Size)
			sizeBytes = &sz
		}

		for _, a := range it.Attrs {
			switch a.Name {
			case "category":
				if v, err := strconv.Atoi(a.Value); err == nil {
					it.Categories = append(it.Categories, v)
				}
			case "format":
				// some book indexers (e.g. Libgen) explicitly include the file format
				formatAttr = strings.ToLower(strings.TrimSpace(a.Value))
			case "seeders":
				if v, err := strconv.Atoi(a.Value); err == nil {
					x := v
					seeders = &x
				}
			case "leechers":
				if v, err := strconv.Atoi(a.Value); err == nil {
					x := v
					leechers = &x
				}
			case "peers":
				// some indexers send "peers" instead of leechers;
				// only use it if we don't already have leechers
				if leechers == nil {
					if v, err := strconv.Atoi(a.Value); err == nil {
						x := v
						leechers = &x
					}
				}
			}
		}

		results = append(results, dao.SearchResult{
			Title:      it.Title,
			Link:       it.Link,
			IndexerName: doc.Channel.IndexerTitle,
			SizeBytes:  sizeBytes,
			Seeders:    seeders,
			Leechers:   leechers,
			Categories: it.Categories,
			FormatAttr: formatAttr,
		})
	}

	return results, nil
}

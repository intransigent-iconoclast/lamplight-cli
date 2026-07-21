package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ResolvedTorrent struct {
	Magnet    string
	FileBytes []byte
}

// note to self: this is what is called a sentinal error
// its prefereable in THIS case and is to break the flow after client.CheckRedirect is called
// golang http client by default follows all redirects ... we don't want that in this case since redirect
// is actually a magnet link normally ... weird i know but :shrug:
var errMagnetRedirect = errors.New("magnet redirect")

func Resolve(ctx context.Context, baseClient *http.Client, link string) (*ResolvedTorrent, error) {
	link = strings.TrimSpace(link)

	// if link is a magnet
	if strings.HasPrefix(strings.ToLower(link), "magnet:") {
		return &ResolvedTorrent{
			Magnet: link,
		}, nil
	}

	// copy client so you don't mutate the base client unintended
	client := *baseClient
	// for saving the magnet that is actually the redirect link
	var redirectedMagnet string
	// this is a special method for checking if a redirect is occuring ... normal behavior with
	// go client is to follow redirects but with this we can interrupt that flow and perfomr some action
	// in thise case we want to not follow the redirect (since location is a magnet) and just end there.
	// NOTE: this sets it IN ADVANCE before the call is made and before the request is even built
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if strings.HasPrefix(strings.ToLower(req.URL.String()), "magnet:") {
			redirectedMagnet = req.URL.String()
			return errMagnetRedirect
		}
		return nil
	}

	// once magnet link is retrieved we're good to fetch
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// make the request to get the magnet or the torrent file
	resp, err := client.Do(req)
	if err != nil {
		// THIS is where the checkRedirect would return an error WITH the magnet link
		if errors.Is(err, errMagnetRedirect) {
			return &ResolvedTorrent{
				Magnet: redirectedMagnet,
			}, nil
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// haven't verrified below this reallhy ...
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	// Reject obvious HTML
	lower := bytes.ToLower(bytes.TrimSpace(body))
	if bytes.HasPrefix(lower, []byte("<!doctype html")) ||
		bytes.HasPrefix(lower, []byte("<html")) {
		return nil, fmt.Errorf("received html instead of torrent file")
	}

	// Torrent files are bencoded dictionaries and thus must start with 'd'
	if body[0] != 'd' {
		return nil, fmt.Errorf("response does not look like a torrent file")
	}

	return &ResolvedTorrent{
		FileBytes: body,
	}, nil
}

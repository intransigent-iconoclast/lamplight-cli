package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
)

type DelugeClient struct {
	client        *http.Client
	clientDetails *entity.Downloader
}

func NewDelugeClient(
	client *http.Client,
	clientDetails *entity.Downloader,
) *DelugeClient {

	if client == nil {
		jar, _ := cookiejar.New(nil)
		client = &http.Client{
			Jar:     jar,
			Timeout: 10 * time.Second,
		}
	} else if client.Jar == nil {
		jar, _ := cookiejar.New(nil)
		client.Jar = jar
	}

	return &DelugeClient{
		client:        client,
		clientDetails: clientDetails,
	}
}

func (c *DelugeClient) Supports(kind entity.DownloaderType) bool {
	return kind == entity.Deluge
}

func (c *DelugeClient) Add(ctx context.Context, resolved *ResolvedTorrent) error {
	if err := c.Authenticate(ctx); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	if resolved.Magnet != "" {
		return c.addMagnet(ctx, resolved.Magnet)
	}

	if len(resolved.FileBytes) > 0 {
		return c.addFile(ctx, resolved.FileBytes)
	}

	return fmt.Errorf("invalid resolved torrent")
}

func (c *DelugeClient) addMagnet(ctx context.Context, magnet string) error {
	params := []any{magnet, map[string]any{}}

	body := dao.DelugeAddMagnetBody{
		Method: "core.add_torrent_magnet",
		Params: params,
		ID:     2,
	}

	return c.sendRPC(ctx, body)
}

func (c *DelugeClient) addFile(ctx context.Context, torrentBytes []byte) error {
	encoded := base64.StdEncoding.EncodeToString(torrentBytes)

	params := []any{
		"lamplight.torrent",
		encoded,
		map[string]any{},
	}

	body := dao.DelugeAddFileBody{
		Method: "core.add_torrent_file",
		Params: params,
		ID:     2,
	}

	return c.sendRPC(ctx, body)
}

func (c *DelugeClient) sendRPC(ctx context.Context, payload any) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal rpc body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		buildDelugeUrl(*c.clientDetails),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return fmt.Errorf("create rpc request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("rpc request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deluge returned status %d", resp.StatusCode)
	}

	return nil
}

// this feels very verbose
func (c *DelugeClient) Authenticate(ctx context.Context) error {
	reqBody := dao.DelugeAuthBody{
		Method: "auth.login",
		Params: []string{c.clientDetails.Password},
		ID:     1,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal auth body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		buildDelugeUrl(*c.clientDetails),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return fmt.Errorf("create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth returned status %d", resp.StatusCode)
	}

	var respBody dao.DelugeAuthResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return fmt.Errorf("decode auth response: %w", err)
	}

	// If Deluge returned an error object (not null)
	if respBody.Error != nil && string(respBody.Error) != "null" {
		return fmt.Errorf("deluge auth error: %s", string(respBody.Error))
	}

	// If result is false
	if !respBody.Result {
		return fmt.Errorf("deluge authentication failed (invalid password?)")
	}

	return nil
}

// Builds the delgue Url. In theory it could be possible that the url this constructs is invalid.
// but that doesn't matter because any function invoking this should handle error and display the url. This
// would always be caused by user error (incorectly constructed at client creation time).
// TODO: Add error for this type of issue so user knows what they did wrong.
func buildDelugeUrl(details entity.Downloader) string {
	return fmt.Sprintf("%s://%s:%d/json", details.Scheme, details.Host, details.Port)
}

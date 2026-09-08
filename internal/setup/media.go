package setup

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// maxMediaBytes caps the size of uploaded emoji/sticker files so a misbehaving
// URL can't balloon memory beyond Discord's own limits (256KB emoji).
const maxMediaBytes = 6 << 20 // 6 MiB

var mediaClient = &http.Client{Timeout: 30 * time.Second}

// fetchAsset downloads bytes from a public URL, refusing anything larger than
// maxMediaBytes.
func fetchAsset(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "MinjiBot-server-setup")
	resp, err := mediaClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download %s returned %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMediaBytes))
	if err != nil {
		return nil, "", err
	}
	if len(body) >= maxMediaBytes {
		return nil, "", fmt.Errorf("download %s exceeds the %d byte limit", url, maxMediaBytes)
	}
	return body, http.DetectContentType(body), nil
}

// dataURI wraps bytes as the data:image URI the emoji endpoint requires.
func dataURI(data []byte, contentType string) string {
	ft := "png"
	if i := strings.Index(contentType, "/"); contentType != "" && i >= 0 {
		ft = contentType[i+1:]
	}
	return "data:image/" + ft + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func (r *Runner) runEmojis(ctx context.Context, guildID string, cfg *Config, steps *[]Step) {
	for _, ec := range cfg.Emojis {
		data, ct, err := fetchAsset(ctx, ec.URL)
		if err != nil {
			*steps = append(*steps, Step{Section: "emojis", Label: ec.Name, Status: "failed", Detail: err.Error()})
			continue
		}
		emoji, err := r.s.GuildEmojiCreate(guildID, &discordgo.EmojiParams{
			Name:  ec.Name,
			Image: dataURI(data, ct),
		})
		if err != nil {
			*steps = append(*steps, Step{Section: "emojis", Label: ec.Name, Status: "failed", Detail: err.Error()})
			continue
		}
		*steps = append(*steps, Step{Section: "emojis", Label: ec.Name, Status: "ok", Detail: emoji.ID})
	}
}

func (r *Runner) runStickers(ctx context.Context, guildID string, cfg *Config, steps *[]Step) {
	for _, sc := range cfg.Stickers {
		data, _, err := fetchAsset(ctx, sc.URL)
		if err != nil {
			*steps = append(*steps, Step{Section: "stickers", Label: sc.Name, Status: "failed", Detail: err.Error()})
			continue
		}
		// Discord has no typed sticker-create helper in v0.29; the endpoint
		// takes multipart/form-data with name, description, tags, and a file.
		body, contentType, err := stickerMultipart(sc, data)
		if err != nil {
			*steps = append(*steps, Step{Section: "stickers", Label: sc.Name, Status: "failed", Detail: err.Error()})
			continue
		}
		endpoint := discordgo.EndpointGuildStickers(guildID)
		resp, err := r.s.RequestRaw(http.MethodPost, endpoint, contentType, body, endpoint, 0)
		if err != nil {
			*steps = append(*steps, Step{Section: "stickers", Label: sc.Name, Status: "failed", Detail: err.Error()})
			continue
		}
		var st discordgo.Sticker
		if err := json.Unmarshal(resp, &st); err != nil {
			*steps = append(*steps, Step{Section: "stickers", Label: sc.Name, Status: "failed", Detail: "unexpected response"})
			continue
		}
		*steps = append(*steps, Step{Section: "stickers", Label: sc.Name, Status: "ok", Detail: st.ID})
	}
}

// stickerMultipart builds multipart/form-data body for sticker upload.
func stickerMultipart(sc StickerConfig, data []byte) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("name", sanitizeName(sc.Name)); err != nil {
		return nil, "", err
	}
	tags := sc.Tags
	if tags == "" {
		tags = "minji"
	}
	if err := w.WriteField("tags", tags); err != nil {
		return nil, "", err
	}
	if sc.Description != "" {
		_ = w.WriteField("description", sc.Description)
	}
	fw, err := w.CreateFormFile("file", safeFilename(sc.Name))
	if err != nil {
		return nil, "", err
	}
	if _, err := fw.Write(data); err != nil {
		return nil, "", err
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.FormDataContentType(), nil
}

func safeFilename(name string) string {
	name = sanitizeName(name)
	if name == "" {
		name = "sticker"
	}
	return name + ".png"
}

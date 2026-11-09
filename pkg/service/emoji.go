package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ServiceIcons maps canonical provider hosts to Telegram custom emoji IDs.
type ServiceIcons map[string]string

var emojiSetName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

// LoadServiceIcons resolves logos by the fallback emoji assigned in @Stickers.
// An empty set name disables logos without making a network request.
func LoadServiceIcons(ctx context.Context, token, setName string) (ServiceIcons, error) {
	return loadServiceIcons(ctx, &http.Client{Timeout: 10 * time.Second}, "https://api.telegram.org", token, setName)
}

func loadServiceIcons(ctx context.Context, client *http.Client, endpoint, token, setName string) (ServiceIcons, error) {
	if setName == "" {
		return nil, nil
	}
	if !emojiSetName.MatchString(setName) {
		return nil, errors.New("invalid custom emoji set name")
	}
	form := url.Values{"name": {setName}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/bot"+token+"/getStickerSet", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.New("create custom emoji request: invalid Telegram API URL")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		// net/http includes the bot token in url.Error.URL. Keep only its cause.
		var requestError *url.Error
		if errors.As(err, &requestError) {
			err = requestError.Err
		}
		return nil, fmt.Errorf("request custom emoji set: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request custom emoji set: HTTP %d", resp.StatusCode)
	}
	const maxResponseSize = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, errors.New("read custom emoji response failed")
	}
	if len(body) > maxResponseSize {
		return nil, errors.New("custom emoji response is too large")
	}
	var response struct {
		OK        bool `json:"ok"`
		ErrorCode int  `json:"error_code"`
		Result    struct {
			StickerType string `json:"sticker_type"`
			Stickers    []struct {
				Emoji         string `json:"emoji"`
				CustomEmojiID string `json:"custom_emoji_id"`
			} `json:"stickers"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, errors.New("invalid custom emoji response JSON")
	}
	if !response.OK {
		return nil, fmt.Errorf("custom emoji request rejected by Telegram: code %d", response.ErrorCode)
	}
	if response.Result.StickerType != "custom_emoji" {
		return nil, errors.New("configured sticker set is not a custom emoji set")
	}
	icons := make(ServiceIcons)
	for _, sticker := range response.Result.Stickers {
		if !validEmojiID(sticker.CustomEmojiID) {
			continue
		}
		for _, host := range []string{"open.spotify.com", "music.yandex.com", "music.apple.com", "music.youtube.com"} {
			if strings.TrimSuffix(sticker.Emoji, "\ufe0f") != strings.TrimSuffix(serviceEmoji(host), "\ufe0f") {
				continue
			}
			if _, exists := icons[host]; exists {
				return nil, fmt.Errorf("ambiguous custom emoji for provider %s", host)
			}
			icons[host] = sticker.CustomEmojiID
		}
	}
	if len(icons) == 0 {
		return nil, errors.New("custom emoji set contains no matching service logos")
	}
	return icons, nil
}

func serviceEmoji(host string) string {
	switch host {
	case "open.spotify.com":
		return "🎧"
	case "music.yandex.com":
		return "🎵"
	case "music.apple.com":
		return "🎶"
	case "music.youtube.com":
		return "▶️"
	default:
		return ""
	}
}

func validEmojiID(id string) bool {
	number, err := strconv.ParseUint(id, 10, 64)
	return err == nil && number != 0 && strconv.FormatUint(number, 10) == id
}

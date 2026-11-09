package service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/tennuem/tbot/pkg/provider"
	"github.com/tennuem/telegram"
)

var commandHelp = "/help — команды\n/ping — проверка доступности\n/list — список треков"

func NewTelegramHandler(svc Service, icons ServiceIcons) telegram.Handler {
	mux := telegram.NewServeMux()
	mux.HandleFunc("/help", func(w *telegram.ResponseWriter, r *telegram.Request) { w.Text = commandHelp })
	mux.HandleFunc("/ping", func(w *telegram.ResponseWriter, r *telegram.Request) { w.Text = "pong" })
	mux.HandleFunc("*", func(w *telegram.ResponseWriter, r *telegram.Request) {
		if r.Message == nil || r.Message.From == nil {
			return
		}
		user := r.Message.From
		name := strings.TrimSpace(user.FirstName + " " + user.LastName)
		if user.UserName != "" {
			name = "@" + user.UserName
		}
		resp, err := svc.FindLinks(context.Background(), &Message{
			URL: r.Message.Text, UserID: user.ID, UserName: name,
			ChatID: r.Message.Chat.ID, FirstSeenAt: time.Unix(int64(r.Message.Date), 0).UTC(),
		})
		if errors.Is(err, ErrLinkNotFound) {
			return
		}
		w.ReplyToMessageID = r.Message.MessageID
		if err != nil {
			w.Text = "Не удалось найти трек. Попробуйте ещё раз позже."
			if errors.Is(err, ErrProviderNotFound) {
				w.Text = "Этот музыкальный сервис пока не поддерживается."
			}
			return
		}
		w.ParseMode = "HTML"
		w.DisableWebPagePreview = true
		w.Text = formatTrack(resp, icons)
	})
	mux.HandleFunc("/list", func(w *telegram.ResponseWriter, r *telegram.Request) {
		if r.Message == nil || r.Message.From == nil {
			return
		}
		resp, err := svc.GetList(context.Background(), r.Message.From.ID)
		if errors.Is(err, ErrLinksNotFound) {
			w.Text = "Вы ещё не присылали треки."
			return
		}
		if err != nil {
			w.Text = "Не удалось загрузить список треков."
			return
		}
		w.DisableWebPagePreview = true
		w.Text = resp
	})
	return mux
}

func formatTrack(m *Message, icons ServiceIcons) string {
	var text strings.Builder
	fmt.Fprintf(&text, "<b>%s</b>", html.EscapeString(m.Title))
	links := append([]Link(nil), m.Links...)
	sort.SliceStable(links, func(i, j int) bool { return links[i].Provider < links[j].Provider })
	var rendered []string
	for _, link := range links {
		if link.Provider == m.Provider {
			continue
		}
		u, err := provider.NormalizeURL(link.URL)
		if err != nil || u.Host != link.Provider {
			continue
		}
		name := link.Name
		switch link.Provider {
		case "music.yandex.com":
			name = "Яндекс Музыка"
		case "open.spotify.com":
			name = "Spotify"
		case "music.youtube.com":
			name = "YouTube Music"
		case "music.apple.com":
			name = "Apple Music"
		}
		label := fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(u.String()), html.EscapeString(name))
		if id := icons[link.Provider]; validEmojiID(id) {
			if emoji := serviceEmoji(link.Provider); emoji != "" {
				label = fmt.Sprintf(`<tg-emoji emoji-id="%s">%s</tg-emoji> %s`, id, emoji, label)
			}
		}
		rendered = append(rendered, label)
	}
	if len(rendered) > 0 {
		text.WriteString("\n\n" + strings.Join(rendered, " · "))
	} else {
		text.WriteString("\n\nСсылки на другие сервисы пока не найдены.")
	}
	if m.CacheStatus == "hit" {
		text.WriteString("\n\n<i>Из кеша.")
		if m.Refreshing {
			text.WriteString(" Ссылки перепроверяются в фоне.")
		}
		text.WriteString("</i>")
		if m.UserID != 0 {
			name := strings.Join(strings.Fields(m.UserName), " ")
			if name == "" {
				name = fmt.Sprintf("пользователь %d", m.UserID)
			}
			fmt.Fprintf(&text, "\nВпервые прислал: %s", html.EscapeString(name))
			if !m.FirstSeenAt.IsZero() {
				text.WriteString(" · " + m.FirstSeenAt.In(time.FixedZone("MSK", 3*60*60)).Format("02.01.2006 15:04") + " МСК")
			}
		}
	}
	return text.String()
}

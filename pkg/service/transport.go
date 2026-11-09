package service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tennuem/tbot/pkg/provider"
	"github.com/tennuem/telegram"
)

var commandHelp = "Пришлите ссылку на трек — найду его в других музыкальных сервисах.\n\n/help — команды\n/ping — проверка доступности\n/list — ваши треки\n/chatid — ID текущего чата"

func NewTelegramHandler(svc Service, icons ServiceIcons, botUsername string) telegram.Handler {
	startedAt := time.Now()
	return telegram.HandlerFunc(func(w *telegram.ResponseWriter, r *telegram.Request) {
		if r == nil || r.Message == nil {
			return
		}
		if !r.Message.IsCommand() {
			handleTrack(svc, icons, w, r)
			return
		}
		command, recipient, addressed := strings.Cut(r.Message.CommandWithAt(), "@")
		if addressed && !strings.EqualFold(recipient, botUsername) {
			return
		}
		w.ReplyToMessageID = r.Message.MessageID
		switch command {
		case "start", "help":
			w.Text = commandHelp
		case "ping":
			w.Text = formatPing(time.Now(), startedAt, r.Message.Date)
		case "list":
			handleList(svc, w, r)
		case "chatid":
			if r.Message.Chat != nil {
				w.Text = fmt.Sprintf("ID чата: %d", r.Message.Chat.ID)
			}
		default:
			w.Text = "Неизвестная команда. Используйте /help."
		}
	})
}

func formatPing(now, startedAt time.Time, sentAt int) string {
	text := "pong 🏓"
	if sentAt > 0 {
		delay := max(now.Sub(time.Unix(int64(sentAt), 0)), 0).Round(time.Second)
		text += fmt.Sprintf("\nВремя с отправки: ≈%.0f с (точность 1 с)", delay.Seconds())
	}
	uptime := max(now.Sub(startedAt), 0).Truncate(time.Second)
	return fmt.Sprintf("%s\nБот работает: %d ч %d мин %d с", text,
		int64(uptime/time.Hour), int64(uptime/time.Minute)%60, int64(uptime/time.Second)%60)
}

func handleTrack(svc Service, icons ServiceIcons, w *telegram.ResponseWriter, r *telegram.Request) {
	if r.Message.From == nil || r.Message.Chat == nil {
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
		MessageID: r.Message.MessageID,
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
}

func handleList(svc Service, w *telegram.ResponseWriter, r *telegram.Request) {
	if r.Message.From == nil {
		return
	}
	page := 1
	if arg := strings.TrimSpace(r.Message.CommandArguments()); arg != "" {
		var err error
		page, err = strconv.Atoi(arg)
		if err != nil || page < 1 {
			w.Text = "Используйте /list или /list 2 для выбора страницы."
			return
		}
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
	w.Text = formatListPage(resp, page)
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
	if (m.CacheStatus == "hit" || m.SeenBefore) && (m.UserID != 0 || m.UserName != "") {
		name := strings.Join(strings.Fields(m.UserName), " ")
		if name == "" {
			name = fmt.Sprintf("пользователь %d", m.UserID)
		}
		fmt.Fprintf(&text, "\n\nВпервые прислал: %s", html.EscapeString(name))
		if !m.FirstSeenAt.IsZero() {
			text.WriteString(" · " + m.FirstSeenAt.In(time.FixedZone("MSK", 3*60*60)).Format("02.01.2006 15:04") + " МСК")
		}
	}
	return text.String()
}

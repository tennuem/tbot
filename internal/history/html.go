package history

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tennuem/tbot/pkg/provider"
	"github.com/tennuem/tbot/pkg/service"
)

type ExtractOptions struct {
	ChatID       int64
	Timezone     string
	ExcludeUsers map[int]bool
}

type ExtractReport struct {
	Messages         int `json:"messages"`
	ExcludedMessages int `json:"excluded_messages"`
	UnknownAuthors   int `json:"unknown_authors"`
	MissingAuthors   int `json:"missing_authors"`
	InvalidDates     int `json:"invalid_dates"`
	InvalidLinks     int `json:"invalid_links"`
}

var authorID = regexp.MustCompile(`(?:^|/)author_(\d+)\.`)
var htmlURL = regexp.MustCompile(`https?://[^\s<>"']+`)

func ExtractHTML(input string, options ExtractOptions) (*Archive, ExtractReport, error) {
	var report ExtractReport
	zone, err := time.LoadLocation(options.Timezone)
	if err != nil {
		return nil, report, fmt.Errorf("load export timezone: %w", err)
	}
	info, err := os.Stat(input)
	if err != nil {
		return nil, report, err
	}
	paths := []string{input}
	if info.IsDir() {
		paths, err = filepath.Glob(filepath.Join(input, "messages*.html"))
		if err != nil {
			return nil, report, err
		}
		sort.Slice(paths, func(i, j int) bool { return pageNumber(paths[i]) < pageNumber(paths[j]) })
	}
	if len(paths) == 0 {
		return nil, report, errors.New("no messages*.html files found")
	}
	archive := &Archive{Version: Version, CreatedAt: time.Now().UTC(), Timezone: options.Timezone, Shares: []service.Share{}}
	var currentID int
	var currentName string
	seen := make(map[string]bool)
	for _, path := range paths {
		doc, err := readHTML(path)
		if err != nil {
			return nil, report, err
		}
		if archive.ChatName == "" {
			archive.ChatName = strings.TrimSpace(doc.Find("title").First().Text())
		}
		doc.Find(".history > .message.default").Each(func(_ int, message *goquery.Selection) {
			report.Messages++
			body := message.ChildrenFiltered(".body").First()
			if !message.HasClass("joined") {
				currentID, currentName = 0, ""
			}
			if src, exists := message.ChildrenFiltered(".userpic_wrap").Find("img").Attr("src"); exists {
				if match := authorID.FindStringSubmatch(src); len(match) == 2 {
					currentID, _ = strconv.Atoi(match[1])
				}
			}
			if name := body.ChildrenFiltered(".from_name").First(); name.Length() != 0 {
				currentName = strings.Join(strings.Fields(name.Text()), " ")
			}
			if options.ExcludeUsers[currentID] {
				report.ExcludedMessages++
				return
			}
			links := collectLinks(body)
			if len(links) == 0 {
				return
			}
			if currentName == "" {
				report.MissingAuthors++
				return
			}
			if currentID <= 0 {
				report.UnknownAuthors++
			}
			date, _ := body.ChildrenFiltered(".date").First().Attr("title")
			sentAt, err := time.ParseInLocation("2 January 2006, 15:04:05", date, zone)
			if err != nil {
				report.InvalidDates++
				return
			}
			id, _ := message.Attr("id")
			messageID, err := strconv.Atoi(strings.TrimPrefix(id, "message"))
			if err != nil || messageID <= 0 {
				report.InvalidDates++
				return
			}
			for _, link := range links {
				u, err := provider.NormalizeURL(link.url)
				if err != nil {
					report.InvalidLinks++
					continue
				}
				share := service.Share{MessageID: messageID, ChatID: options.ChatID, UserID: currentID,
					UserName: currentName, SentAt: sentAt.UTC(), URL: u.String(), Provider: u.Host, Title: link.title}
				key := ShareKey(share)
				if !seen[key] {
					seen[key] = true
					archive.Shares = append(archive.Shares, share)
				}
			}
		})
	}
	if report.Messages == 0 {
		return nil, report, errors.New("no Telegram messages found in HTML export")
	}
	sort.SliceStable(archive.Shares, func(i, j int) bool { return archive.Shares[i].SentAt.Before(archive.Shares[j].SentAt) })
	return archive, report, archive.Validate(false)
}

func readHTML(path string) (*goquery.Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() > 100<<20 {
		return nil, errors.New("cannot read HTML export or page exceeds 100 MiB")
	}
	return goquery.NewDocumentFromReader(io.LimitReader(file, (100<<20)+1))
}

type extractedLink struct{ url, title string }

func collectLinks(body *goquery.Selection) []extractedLink {
	var links []extractedLink
	text := body.ChildrenFiltered(".text")
	text.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
			links = append(links, extractedLink{url: href})
		}
	})
	// Do not extract links from preview descriptions, replies, or forwarded authors.
	body.ChildrenFiltered(".media_wrap").Find("a.webpage_preview[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
			links = append(links, extractedLink{url: href, title: strings.Join(strings.Fields(a.Find(".webpage_title").Text()), " ")})
		}
	})
	plain := text.Clone()
	plain.Find("a, script, style").Remove()
	plain.Find("br").ReplaceWithHtml(" ")
	for _, raw := range htmlURL.FindAllString(plain.Text(), -1) {
		raw = strings.TrimRight(raw, ".,;!")
		for strings.HasSuffix(raw, ")") && strings.Count(raw, ")") > strings.Count(raw, "(") {
			raw = strings.TrimSuffix(raw, ")")
		}
		links = append(links, extractedLink{url: raw})
	}
	// Prefer preview titles when the same URL appears both in text and preview.
	sort.SliceStable(links, func(i, j int) bool { return links[i].title != "" && links[j].title == "" })
	return links
}

func pageNumber(path string) int {
	n, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "messages"), ".html"))
	return n
}

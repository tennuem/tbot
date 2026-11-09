package service

import (
	"fmt"
	"strings"
)

func formatListPage(text string, page int) string {
	const budget = 3500 // Leave room for a footer within Telegram's 4096-character limit.
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	var pages []string
	var current strings.Builder
	units := 0
	for i := 0; i < len(lines); i += 2 {
		title, link := lines[i], ""
		if i+1 < len(lines) {
			link = lines[i+1]
		}
		if textUnits(link) > budget-2 {
			link = "Ссылка слишком длинная для сообщения; она сохранена в экспорте истории."
		}
		title = clipText(title, budget-textUnits(link)-2)
		entry := title + "\n" + link + "\n"
		if units+textUnits(entry) > budget && units > 0 {
			pages = append(pages, current.String())
			current.Reset()
			units = 0
		}
		current.WriteString(entry)
		units += textUnits(entry)
	}
	if current.Len() > 0 {
		pages = append(pages, current.String())
	}
	if page < 1 || page > len(pages) {
		return fmt.Sprintf("Такой страницы нет. Доступно страниц: %d.", len(pages))
	}
	result := pages[page-1]
	if len(pages) > 1 {
		result += fmt.Sprintf("\nСтраница %d из %d.", page, len(pages))
		if page < len(pages) {
			result += fmt.Sprintf(" Далее: /list %d", page+1)
		}
	}
	return result
}

func textUnits(text string) int {
	units := 0
	for _, r := range text {
		units++
		if r > 0xffff {
			units++
		}
	}
	return units
}

func clipText(text string, budget int) string {
	if textUnits(text) <= budget {
		return text
	}
	if budget <= 0 {
		return ""
	}
	units := 0
	for i, r := range text {
		size := 1
		if r > 0xffff {
			size = 2
		}
		if units+size > budget-1 {
			return text[:i] + "…"
		}
		units += size
	}
	return text
}

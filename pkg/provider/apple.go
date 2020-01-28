package provider

import (
	"fmt"
	"net/http"
	"regexp"

	"golang.org/x/net/html"
)

func NewAppleProvider() Provider {
	return &appleProvider{}
}

type appleProvider struct{}

func (p *appleProvider) GetTitle(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	n, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	// ‎Песня «DLBM» (Miyagi &amp; Эндшпиль &amp; N.E.R.A.K.) в Apple Music
	// Песня «Babushka Boi» (A$AP Rocky) в Apple Music
	r := regexp.MustCompile(`^.+«(.+)» \((.+)\)`)

	title := pageTitle(n)
	// fmt.Printf("find: %#v\n", r.FindStringSubmatch(title))
	ss := r.FindStringSubmatch(title)

	return fmt.Sprintf("%s - %s", ss[1], ss[2]), nil
}

func (p *appleProvider) GetURL(title string) (string, error) {
	return "", ErrURLNotFound
}

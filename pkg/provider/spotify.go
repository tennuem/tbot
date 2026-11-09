package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	spotifyAPI "github.com/zmb3/spotify"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/spotify"
)

func NewSpotifyProvider(ctx context.Context, cid, csecret string) Provider {
	cfg := clientcredentials.Config{
		ClientID:     cid,
		ClientSecret: csecret,
		TokenURL:     spotify.Endpoint.TokenURL,
	}
	c := spotifyAPI.NewClient(newSpotifyHTTPClient(ctx, cfg))
	return &spotifyProvider{client: &c}
}

func newSpotifyHTTPClient(ctx context.Context, cfg clientcredentials.Config) *http.Client {
	// oauth2's token source refreshes lazily and synchronizes concurrent callers.
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: 10 * time.Second})
	httpClient := cfg.Client(ctx)
	httpClient.Timeout = 10 * time.Second
	return httpClient
}

type spotifyClient interface {
	GetTrack(id spotifyAPI.ID) (*spotifyAPI.FullTrack, error)
	Search(query string, t spotifyAPI.SearchType) (*spotifyAPI.SearchResult, error)
}

type spotifyProvider struct {
	client spotifyClient
}

func (p *spotifyProvider) Name() string {
	return "Spotify"
}

func (p *spotifyProvider) Host() string {
	return "open.spotify.com"
}

func (p *spotifyProvider) GetTitle(url string) (string, error) {
	u, err := NormalizeURL(url)
	if err != nil || u.Host != p.Host() {
		return "", ErrTitleNotFound
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "track" || !spotifyTrackID.MatchString(parts[1]) {
		return "", ErrTitleNotFound
	}
	id := parts[1]
	track, err := p.client.GetTrack(spotifyAPI.ID(id))
	if err != nil {
		return "", errors.Wrap(err, "get track")
	}
	var artists []string
	for _, a := range track.Artists {
		artists = append(artists, a.Name)
	}
	title := fmt.Sprintf("%s — %s", track.Name, strings.Join(artists, ", "))
	return title, nil
}

var spotifyTrackID = regexp.MustCompile(`^[A-Za-z0-9]{22}$`)

func (p *spotifyProvider) GetURL(title string) (string, error) {
	results, err := p.client.Search(title, spotifyAPI.SearchTypeTrack)
	if err != nil {
		return "", errors.Wrap(err, "search track")
	}
	if results.Tracks == nil || len(results.Tracks.Tracks) == 0 {
		return "", ErrURLNotFound
	}
	v, ok := results.Tracks.Tracks[0].ExternalURLs["spotify"]
	if !ok {
		return "", ErrURLNotFound
	}
	return v, nil
}

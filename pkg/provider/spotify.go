package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	spotifyAPI "github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/spotify"
)

func NewSpotifyProvider(logger log.Logger, cid, csecret string) Provider {
	cfg := clientcredentials.Config{
		ClientID:     cid,
		ClientSecret: csecret,
		TokenURL:     spotify.Endpoint.TokenURL,
	}
	return &spotifyProvider{cfg, logger}
}

type spotifyProvider struct {
	cfg    clientcredentials.Config
	logger log.Logger
}

func (p *spotifyProvider) GetTitle(url string) (string, error) {
	substr := "track/"
	id := url[strings.Index(url, substr)+len(substr):]
	token, err := p.cfg.Token(context.Background())
	api := spotifyAPI.Authenticator{}.NewClient(token)
	track, err := api.GetTrack(spotifyAPI.ID(id))
	if err != nil {
		return "", errors.Wrap(err, "get track")
	}
	var artists []string
	for _, a := range track.Artists {
		artists = append(artists, a.Name)
	}
	title := fmt.Sprintf("%s — %s", track.Name, strings.Join(artists, ", "))
	level.Info(p.logger).Log("method", "GetTitle", "msg", title)
	return title, nil
}

func (p *spotifyProvider) GetURL(title string) (string, error) {
	token, err := p.cfg.Token(context.Background())
	api := spotifyAPI.Authenticator{}.NewClient(token)
	results, err := api.Search(title, spotifyAPI.SearchTypeTrack)
	if err != nil {
		return "", errors.Wrap(err, "search track")
	}
	if results.Tracks == nil && len(results.Tracks.Tracks) == 0 {
		return "", ErrURLNotFound
	}
	v, ok := results.Tracks.Tracks[0].ExternalURLs["spotify"]
	if !ok {
		return "", ErrURLNotFound
	}
	return v, nil
}

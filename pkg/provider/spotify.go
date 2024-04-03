package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/tennuem/tbot/tools/logging"
	spotifyAPI "github.com/zmb3/spotify"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/spotify"
)

func NewSpotifyProvider(ctx context.Context, cid, csecret string) Provider {
	logger := logging.FromContext(ctx)
	logger = log.With(logger, "component", "spotify")
	cfg := clientcredentials.Config{
		ClientID:     cid,
		ClientSecret: csecret,
		TokenURL:     spotify.Endpoint.TokenURL,
	}
	p := &spotifyProvider{logger: logger}
	token, err := cfg.Token(context.Background())
	if err != nil {
		level.Error(logger).Log("get token", err)
	}
	c := spotifyAPI.Authenticator{}.NewClient(token)
	p.client = &c

	go func() {
		for range time.NewTicker(time.Second * 1).C {
			select {
			case <-ctx.Done():
				level.Error(logger).Log("err", ctx.Err())
				return
			default:
				if p.token.Valid() {
					continue
				}
				token, err := cfg.Token(context.Background())
				if err != nil {
					level.Error(logger).Log("get token", err)
					continue
				}
				p.token = token
				c := spotifyAPI.Authenticator{}.NewClient(token)
				p.client = &c
			}
		}
	}()
	return p
}

type spotifyClient interface {
	GetTrack(id spotifyAPI.ID) (*spotifyAPI.FullTrack, error)
	Search(query string, t spotifyAPI.SearchType) (*spotifyAPI.SearchResult, error)
}

type spotifyProvider struct {
	token  *oauth2.Token
	client spotifyClient
	logger log.Logger
}

func (p *spotifyProvider) Name() string {
	return "spotify"
}

func (p *spotifyProvider) Host() string {
	return "open.spotify.com"
}

func (p *spotifyProvider) GetTitle(url string) (string, error) {
	substr := "track/"
	id := url[strings.Index(url, substr)+len(substr):]
	track, err := p.client.GetTrack(spotifyAPI.ID(id))
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

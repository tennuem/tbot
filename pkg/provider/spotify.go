package provider

import (
	"context"
	"fmt"
	"log"
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
	p := &spotifyProvider{}
	token, err := cfg.Token(context.Background())
	if err != nil {
		log.Fatalln("get token", err)
	}
	c := spotifyAPI.Authenticator{}.NewClient(token)
	p.client = &c

	go func() {
		for range time.NewTicker(time.Second * 1).C {
			select {
			case <-ctx.Done():
				log.Println("err", ctx.Err())
				return
			default:
				if p.token.Valid() {
					continue
				}
				token, err := cfg.Token(context.Background())
				if err != nil {
					log.Println("get token", err)
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
}

func (p *spotifyProvider) Name() string {
	return "🎵 spotify"
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

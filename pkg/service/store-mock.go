package service

import "context"

func NewStoreMock() Store {
	return &storeMock{}
}

type storeMock struct{}

func (s *storeMock) Save(ctx context.Context, m *Message) error {
	return nil
}

func (s *storeMock) FindByURL(ctx context.Context, url string) (*Message, error) {
	return &Message{
		URL:   "https://music.yandex.com/album/8508157/track/57016085",
		Title: "Babushka Boi — A$AP Rocky",
		Links: []string{
			"https://music.youtube.com/watch?v=KViOTZ62zBg",
			"https://music.apple.com/us/album/babushka-boi-single/1477644647",
		},
	}, nil
}

func (s *storeMock) FindByUsername(ctx context.Context, username string) ([]Message, error) {
	return []Message{Message{
		URL:   "https://music.yandex.com/album/8508157/track/57016085",
		Title: "Babushka Boi — A$AP Rocky",
		Links: []string{
			"https://music.youtube.com/watch?v=KViOTZ62zBg",
			"https://music.apple.com/us/album/babushka-boi-single/1477644647",
		},
	}}, nil
}

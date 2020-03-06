package service

import "context"

func NewStoreMock() Store {
	return &storeMock{}
}

type storeMock struct{}

func (s *storeMock) Save(ctx context.Context, m *Model) error {
	return nil
}

func (s *storeMock) FindByMsg(ctx context.Context, msg string) *Model {
	return &Model{
		Msg:   "https://music.yandex.com/album/8508157/track/57016085",
		Title: "Babushka Boi — A$AP Rocky",
		URL: []string{
			"https://music.youtube.com/watch?v=KViOTZ62zBg",
			"https://music.apple.com/us/album/babushka-boi-single/1477644647",
		},
	}
}

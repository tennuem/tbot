package provider

func NewMockProvider() Provider {
	return &mockProvider{}
}

type mockProvider struct{}

func (p *mockProvider) Host() string {
	return "example.com"
}

func (p *mockProvider) GetTitle(url string) (string, error) {
	return "test_title", nil
}

func (p *mockProvider) GetURL(title string) (string, error) {
	return "test_url", nil
}

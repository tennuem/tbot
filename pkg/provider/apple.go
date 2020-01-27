package provider

func NewAppleProvider() Provider {
	return &appleProvider{}
}

type appleProvider struct{}

func (p *appleProvider) GetTitle(url string) (string, error) {
	return "", ErrTitleNotFound
}

func (p *appleProvider) GetURL(title string) (string, error) {
	return "", ErrURLNotFound
}

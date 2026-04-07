package llm

type OpenRouterProvider struct {
	*OpenAIProvider
}

func NewOpenRouterProvider(cfg Config) (*OpenRouterProvider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL(ProviderOpenRouter)
	}

	provider, err := NewOpenAIProvider(cfg)
	if err != nil {
		return nil, err
	}

	return &OpenRouterProvider{OpenAIProvider: provider}, nil
}

func (p *OpenRouterProvider) Name() string {
	return "OpenRouter"
}

func (p *OpenRouterProvider) Type() ProviderType {
	return ProviderOpenRouter
}

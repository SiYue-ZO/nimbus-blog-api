package webapi

import (
	"context"
	"fmt"

	translator "github.com/Conight/go-googletrans"
	"github.com/scc749/nimbus-blog-api/internal/repo"
)

type translationWebAPI struct {
	conf translator.Config
}

func NewTranslationWebAPI() repo.TranslationWebAPI {
	conf := translator.Config{
		UserAgent:   []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"},
		ServiceUrls: []string{"translate.google.com"},
		Proxy:       "http://127.0.0.1:7897",
	}

	return &translationWebAPI{
		conf: conf,
	}
}

func (t *translationWebAPI) Translate(ctx context.Context, text, source, destination string) (string, error) {
	trans := translator.New(t.conf)

	// "auto", "en"
	result, err := trans.Translate(text, source, destination)
	if err != nil {
		return "", fmt.Errorf("TranslationWebAPI - Translate - trans.Translate: %w", err)
	}

	return result.Text, nil
}

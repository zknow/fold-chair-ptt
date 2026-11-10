package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"fold-chair-ptt/internal/ptt"
	"fold-chair-ptt/internal/store"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	client   *ptt.Client
	settings store.Settings

	mu      sync.Mutex
	cancel  context.CancelFunc
	article *ptt.Article
}

func NewApp() *App {
	return &App{
		client:   ptt.NewClient(),
		settings: store.Load(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	runtime.WindowSetAlwaysOnTop(ctx, a.settings.AlwaysOnTop)
}

func (a *App) shutdown(ctx context.Context) {
	a.StopMonitor()
}

func (a *App) GetSettings() store.Settings {
	return a.settings
}

func (a *App) SaveSettings(s store.Settings) error {
	s = store.Normalize(s)
	a.mu.Lock()
	a.settings = s
	a.mu.Unlock()
	return store.Save(s)
}

func (a *App) SetAlwaysOnTop(v bool) error {
	a.mu.Lock()
	a.settings.AlwaysOnTop = v
	s := a.settings
	a.mu.Unlock()
	runtime.WindowSetAlwaysOnTop(a.ctx, v)
	return store.Save(s)
}

func (a *App) LoadArticle(rawURL string) (*ptt.Article, error) {
	normalized, err := ptt.NormalizeURL(rawURL)
	if err != nil {
		return nil, err
	}

	a.StopMonitor()

	html, err := a.client.Get(normalized)
	if err != nil {
		return nil, err
	}
	art, err := ptt.ParseArticle(normalized, html)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.article = art
	a.mu.Unlock()

	return art, nil
}

func (a *App) StartMonitor() error {
	a.mu.Lock()
	art := a.article
	interval := time.Duration(a.settings.PollIntervalMs) * time.Millisecond
	a.mu.Unlock()

	if art == nil {
		return errors.New("尚未載入文章")
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	a.cancel = cancel
	a.mu.Unlock()

	monitor := ptt.NewMonitor(a.client, art, interval, func(pushes []ptt.Push) {
		runtime.EventsEmit(a.ctx, "pushes", pushes)
	})
	go monitor.Run(ctx)
	return nil
}

func (a *App) StopMonitor() {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.mu.Unlock()
}

func (a *App) OpenExternal(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

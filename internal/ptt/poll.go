package ptt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://www.ptt.cc"

// longPollWait 是單次長輪詢的最長等待時間，超時視為無新留言。
const longPollWait = 10 * time.Second

// errLongPollExpired：PTT 長輪詢到期（約 100 秒無新推文）後 id 立即失效，必須重抓文章換新 id。
var errLongPollExpired = errors.New("長輪詢已失效")

type Monitor struct {
	client   *Client
	article  *Article
	interval time.Duration
	seq      int
	consumed int
	emit     func([]Push)
}

func NewMonitor(c *Client, a *Article, interval time.Duration, emit func([]Push)) *Monitor {
	return &Monitor{
		client:   c,
		article:  a,
		interval: interval,
		seq:      len(a.Pushes),
		consumed: len(a.Pushes),
		emit:     emit,
	}
}

func (m *Monitor) Run(ctx context.Context) {
	fails := 0
	for {
		if ctx.Err() != nil {
			return
		}
		pushes, err := m.pollOnce(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[ptt] poll error: %v", err)
			if errors.Is(err, errLongPollExpired) {
				m.rebuild(ctx)
				if !sleep(ctx, m.interval) {
					return
				}
				continue
			}
			fails++
			if fails >= 5 {
				m.rebuild(ctx)
				fails = 0
			}
			if !sleep(ctx, m.interval) {
				return
			}
			continue
		}
		fails = 0
		if len(pushes) == 0 {
			if !sleep(ctx, m.interval) {
				return
			}
			continue
		}
		for i := range pushes {
			pushes[i].Seq = m.seq
			m.seq++
		}
		m.consumed += len(pushes)
		m.emit(pushes)
	}
}

func (m *Monitor) pollOnce(ctx context.Context) ([]Push, error) {
	if m.article.LongPollURL == "" || m.article.PollURL == "" {
		return nil, fmt.Errorf("文章缺少輪詢網址")
	}

	longBody, err := m.client.GetCtx(ctx, baseURL+m.article.LongPollURL, longPollWait)
	if err != nil {
		var se *statusError
		if errors.As(err, &se) {
			return nil, errLongPollExpired
		}
		if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
			return nil, nil
		}
		return nil, err
	}
	var longPoll struct {
		Size json.RawMessage `json:"size"`
		Sig  json.RawMessage `json:"sig"`
	}
	if err := json.Unmarshal([]byte(longBody), &longPoll); err != nil {
		return nil, err
	}

	pollURL := baseURL + m.article.PollURL +
		"&size=" + url.QueryEscape(rawToString(longPoll.Size)) +
		"&size-sig=" + url.QueryEscape(rawToString(longPoll.Sig))

	body, err := m.client.GetCtx(ctx, pollURL, 60*time.Second)
	if err != nil {
		return nil, err
	}
	var data struct {
		PollURL     string `json:"pollUrl"`
		ContentHTML string `json:"contentHtml"`
	}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil, err
	}
	if data.PollURL != "" {
		m.article.PollURL = data.PollURL
	}
	return ParsePushesHTML(data.ContentHTML)
}

// rebuild 在連續失敗後重新抓取文章，補回中斷期間的新推文
func (m *Monitor) rebuild(ctx context.Context) {
	log.Printf("[ptt] rebuild article")
	html, err := m.client.GetCtx(ctx, m.article.URL, 20*time.Second)
	if err != nil {
		log.Printf("[ptt] rebuild fetch error: %v", err)
		return
	}
	art, err := ParseArticle(m.article.URL, html)
	if err != nil {
		log.Printf("[ptt] rebuild parse error: %v", err)
		return
	}
	m.article.PollURL = art.PollURL
	m.article.LongPollURL = art.LongPollURL
	if len(art.Pushes) <= m.consumed {
		return
	}
	rest := art.Pushes[m.consumed:]
	for i := range rest {
		rest[i].Seq = m.seq
		m.seq++
	}
	m.consumed = len(art.Pushes)
	m.emit(rest)
}

func rawToString(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	return strings.Trim(s, `"`)
}

func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		d = time.Second
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

package ptt

import (
	"context"
	"os"
	"testing"
	"time"
)

const articleFixture = `<!DOCTYPE html>
<html><body>
<div id="main-content">
  <div class="article-metaline"><span class="article-meta-tag">作者</span><span class="article-meta-value">dispptt (dispptt)</span></div>
  <div class="article-metaline"><span class="article-meta-tag">看板</span><span class="article-meta-value">Stock</span></div>
  <div class="article-metaline"><span class="article-meta-tag">標題</span><span class="article-meta-value">[請益] 重壓的勇氣？</span></div>
  <div class="article-metaline"><span class="article-meta-tag">時間</span><span class="article-meta-value">Fri Jul 21 10:27:53 2023</span></div>
  <div class="push">
    <span class="push-tag">推 </span>
    <span class="push-userid">alice</span>
    <span class="push-content">: 頭推 https://i.imgur.com/abc.png</span>
    <span class="push-ipdatetime"> 123.45.67.89 07/21 10:28</span>
  </div>
  <div class="push">
    <span class="push-tag">噓 </span>
    <span class="push-userid">bob</span>
    <span class="push-content">: 低調</span>
    <span class="push-ipdatetime"> 07/21 10:29</span>
  </div>
  <div id="article-polling" data-pollurl="/bbs/Stock/M.1689906475.A.C13.html?poll=abc" data-longpollurl="/bbs/Stock/M.1689906475.A.C13.html?longpoll=xyz"></div>
</div>
</body></html>`

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"https://www.ptt.cc/bbs/Stock/M.1.A.1.html", "https://www.ptt.cc/bbs/Stock/M.1.A.1.html", false},
		{"http://www.ptt.cc/bbs/Stock/M.1.A.1.html", "https://www.ptt.cc/bbs/Stock/M.1.A.1.html", false},
		{"https://m.ptt.cc/bbs/Stock/M.1.A.1.html", "https://www.ptt.cc/bbs/Stock/M.1.A.1.html", false},
		{"www.ptt.cc/bbs/Stock/M.1.A.1.html", "https://www.ptt.cc/bbs/Stock/M.1.A.1.html", false},
		{"https://www.ptt.cc/bbs/Stock/M.1.A.1.html?x=1#y", "https://www.ptt.cc/bbs/Stock/M.1.A.1.html", false},
		{"https://www.ptt.cc/bbs/Stock/index.html", "", true},
		{"https://google.com/bbs/Stock/M.1.A.1.html", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		got, err := NormalizeURL(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("NormalizeURL(%q) = %q, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeURL(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseArticle(t *testing.T) {
	a, err := ParseArticle("https://www.ptt.cc/bbs/Stock/M.1689906475.A.C13.html", articleFixture)
	if err != nil {
		t.Fatal(err)
	}
	if a.Author != "dispptt (dispptt)" {
		t.Errorf("author = %q", a.Author)
	}
	if a.Board != "Stock" {
		t.Errorf("board = %q", a.Board)
	}
	if a.Title != "[請益] 重壓的勇氣？" {
		t.Errorf("title = %q", a.Title)
	}
	if a.PollURL == "" || a.LongPollURL == "" {
		t.Errorf("poll urls missing: %q %q", a.PollURL, a.LongPollURL)
	}
	if len(a.Pushes) != 2 {
		t.Fatalf("pushes = %d, want 2", len(a.Pushes))
	}

	p := a.Pushes[0]
	if p.Tag != "推" || p.User != "alice" {
		t.Errorf("push0 tag/user = %q/%q", p.Tag, p.User)
	}
	if p.IP != "123.45.67.89" || p.Date != "07-21" || p.Time != "10:28" {
		t.Errorf("push0 ip/date/time = %q/%q/%q", p.IP, p.Date, p.Time)
	}
	if !p.IsImage {
		t.Errorf("push0 should be image link, link=%q content=%q", p.Link, p.Content)
	}
	if p.Seq != 0 {
		t.Errorf("push0 seq = %d", p.Seq)
	}

	p2 := a.Pushes[1]
	if p2.IP != "" || p2.Date != "07-21" || p2.Time != "10:29" {
		t.Errorf("push1 ip/date/time = %q/%q/%q", p2.IP, p2.Date, p2.Time)
	}
	if p2.Content != "低調" {
		t.Errorf("push1 content = %q", p2.Content)
	}
	if p2.IsImage {
		t.Errorf("push1 should not be image")
	}
}

func TestParseArticleWithoutMetadata(t *testing.T) {
	html := `<html><body><div id="main-content" class="bbs-content">
	<span class="f6"> ▄▄ </span>
	<div class="push"><span class="push-tag">推 </span><span class="push-userid">dan</span><span class="push-content">: 圖王</span><span class="push-ipdatetime"> 11/10 11:59</span></div>
	</div></body></html>`
	a, err := ParseArticle("https://www.ptt.cc/bbs/Gossiping/M.1794217856.A.839.html", html)
	if err != nil {
		t.Fatal(err)
	}
	if a.Board != "Gossiping" {
		t.Errorf("board = %q, want Gossiping (from url)", a.Board)
	}
	if a.Title != "（無標題文章）" {
		t.Errorf("title = %q", a.Title)
	}
	if len(a.Pushes) != 1 || a.Pushes[0].User != "dan" {
		t.Errorf("pushes = %+v", a.Pushes)
	}
}

func TestParseArticleInvalidPage(t *testing.T) {
	if _, err := ParseArticle("https://www.ptt.cc/bbs/Gossiping/M.1.A.1.html", `<html><body>系統異常</body></html>`); err == nil {
		t.Fatal("want error for page without #main-content")
	}
}

func TestParsePushesHTML(t *testing.T) {	pushes, err := ParsePushesHTML(`<div class="push"><span class="push-tag">→ </span><span class="push-userid">carol</span><span class="push-content">: hi</span><span class="push-ipdatetime"> 07/22 09:00</span></div>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(pushes) != 1 || pushes[0].User != "carol" || pushes[0].Content != "hi" {
		t.Fatalf("unexpected pushes: %+v", pushes)
	}
}

// TestLiveArticle 需要網路，設定 FCP_LIVE=1 才會執行
func TestLiveArticle(t *testing.T) {
	if os.Getenv("FCP_LIVE") != "1" {
		t.Skip("set FCP_LIVE=1 to run live test")
	}
	raw := os.Getenv("FCP_URL")
	if raw == "" {
		raw = "https://www.ptt.cc/bbs/Gossiping/M.1794282737.A.9D3.html"
	}
	url, err := NormalizeURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	c := NewClient()
	html, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	art, err := ParseArticle(url, html)
	if err != nil {
		t.Fatal(err)
	}
	if art.Title == "" || art.Board == "" {
		t.Fatalf("parse failed: title=%q board=%q", art.Title, art.Board)
	}
	if art.PollURL == "" || art.LongPollURL == "" {
		t.Fatalf("poll urls missing: %q %q", art.PollURL, art.LongPollURL)
	}
	t.Logf("board=%s title=%s pushes=%d", art.Board, art.Title, len(art.Pushes))

	var total int
	m := NewMonitor(c, art, time.Second, func(p []Push) {
		total += len(p)
		t.Logf("emit %d (total %d)", len(p), total)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	m.Run(ctx)
	if total == 0 {
		t.Fatal("no pushes received within 3 minutes")
	}
}

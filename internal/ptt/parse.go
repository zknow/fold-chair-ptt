package ptt

import (
	"errors"
	neturl "net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	articlePathRe = regexp.MustCompile(`^/bbs/([^/]+)/M\.[^/]+\.html$`)
	urlRe         = regexp.MustCompile(`https?://[^\s]+`)
)

// NormalizeURL 將各種 PTT 文章網址正規化成 https://www.ptt.cc/bbs/<board>/<article>.html
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("請輸入 PTT 文章網址")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := neturl.Parse(raw)
	if err != nil {
		return "", errors.New("網址格式錯誤")
	}
	u.Host = strings.ToLower(u.Host)
	if u.Host == "m.ptt.cc" {
		u.Host = "www.ptt.cc"
	}
	if u.Host != "www.ptt.cc" {
		return "", errors.New("不是 PTT 文章網址")
	}
	u.Scheme = "https"
	u.RawQuery = ""
	u.Fragment = ""
	if !articlePathRe.MatchString(u.Path) {
		return "", errors.New("不是 PTT 文章網址")
	}
	return u.String(), nil
}

// ParseArticle 解析 PTT 文章頁面
func ParseArticle(urlStr, html string) (*Article, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	if doc.Find("#main-content").Length() == 0 {
		return nil, errors.New("找不到文章內容，網址可能已失效")
	}

	a := &Article{URL: urlStr}

	tags := doc.Find(".article-meta-tag")
	values := doc.Find(".article-meta-value")
	tags.Each(func(i int, s *goquery.Selection) {
		value := strings.TrimSpace(values.Eq(i).Text())
		switch strings.TrimSpace(s.Text()) {
		case "作者":
			a.Author = value
		case "看板":
			a.Board = value
		case "標題":
			a.Title = value
		case "時間":
			a.Date = value
		}
	})

	// 部分文章（如 ANSI 圖文）沒有作者/看板/標題等中繼資料，看板改由網址推得。
	if a.Board == "" {
		if u, err := neturl.Parse(urlStr); err == nil {
			if m := articlePathRe.FindStringSubmatch(u.Path); m != nil {
				a.Board = m[1]
			}
		}
	}
	if a.Title == "" {
		a.Title = "（無標題文章）"
	}

	a.PollURL, _ = doc.Find("#article-polling").Attr("data-pollurl")
	a.LongPollURL, _ = doc.Find("#article-polling").Attr("data-longpollurl")

	a.Pushes = parsePushes(doc.Find("#main-content .push"))
	if len(a.Pushes) == 0 {
		a.Pushes = parsePushes(doc.Find(".push"))
	}

	for i := range a.Pushes {
		a.Pushes[i].Seq = i
	}

	return a, nil
}

// ParsePushesHTML 解析長輪詢回傳的 contentHtml
func ParsePushesHTML(html string) ([]Push, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	return parsePushes(doc.Find(".push")), nil
}

func parsePushes(sel *goquery.Selection) []Push {
	out := make([]Push, 0, sel.Length())
	sel.Each(func(_ int, s *goquery.Selection) {
		ipdatetime := strings.TrimSpace(s.Find(".push-ipdatetime").Text())
		if ipdatetime == "" {
			return
		}

		fields := strings.Fields(strings.ReplaceAll(ipdatetime, "/", "-"))
		p := Push{}
		switch len(fields) {
		case 3:
			p.IP, p.Date, p.Time = fields[0], fields[1], fields[2]
		default:
			if len(fields) >= 2 {
				p.Date, p.Time = fields[0], fields[1]
			}
		}

		content := strings.TrimSpace(s.Find(".push-content").Text())
		content = strings.TrimSpace(strings.TrimPrefix(content, ":"))

		p.Tag = strings.TrimSpace(s.Find(".push-tag").Text())
		p.User = strings.TrimSpace(s.Find(".push-userid").Text())
		p.Content = content

		link := ""
		s.Find(".push-content a").EachWithBreak(func(_ int, a *goquery.Selection) bool {
			if href, ok := a.Attr("href"); ok {
				link = href
			} else {
				link = strings.TrimSpace(a.Text())
			}
			return false
		})
		if link == "" {
			link = urlRe.FindString(content)
		}
		p.Link = link
		p.IsImage = isImageLink(link)

		out = append(out, p)
	})
	return out
}

func isImageLink(link string) bool {
	l := strings.ToLower(link)
	if l == "" {
		return false
	}
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg"} {
		if strings.Contains(l, ext) {
			return true
		}
	}
	return strings.Contains(l, "imgur.com")
}

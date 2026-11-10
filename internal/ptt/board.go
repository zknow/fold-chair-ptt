package ptt

import (
	"errors"
	neturl "net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	boardNameRe  = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	boardIndexRe = regexp.MustCompile(`^https://www\.ptt\.cc/bbs/([A-Za-z0-9_]+)/index\d*\.html$`)
)

type BoardEntry struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Date   string `json:"date"`
	Score  int    `json:"score"`
	URL    string `json:"url"`
}

type BoardPage struct {
	Board   string       `json:"board"`
	URL     string       `json:"url"`
	PrevURL string       `json:"prevUrl"`
	NextURL string       `json:"nextUrl"`
	Entries []BoardEntry `json:"entries"`
}

// BoardIndexURL 回傳看板列表頁網址；pageURL 為空時取最新一頁。
func BoardIndexURL(board, pageURL string) (string, error) {
	if pageURL != "" {
		if !boardIndexRe.MatchString(pageURL) {
			return "", errors.New("不是 PTT 看板列表網址")
		}
		return pageURL, nil
	}
	if !boardNameRe.MatchString(board) {
		return "", errors.New("看板名稱錯誤")
	}
	return baseURL + "/bbs/" + board + "/index.html", nil
}

// ParseBoardIndex 解析看板列表頁，已刪除文章會略過，每頁依留言熱度由高到低排序。
func ParseBoardIndex(pageURL, html string) (*BoardPage, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	if doc.Find(".r-list-container").Length() == 0 {
		return nil, errors.New("找不到文章列表，看板可能不存在")
	}

	base, err := neturl.Parse(pageURL)
	if err != nil {
		return nil, err
	}
	resolve := func(href string) string {
		ref, err := neturl.Parse(href)
		if err != nil {
			return ""
		}
		return base.ResolveReference(ref).String()
	}

	page := &BoardPage{URL: pageURL}
	if m := boardIndexRe.FindStringSubmatch(pageURL); m != nil {
		page.Board = m[1]
	}

	doc.Find(".btn-group-paging a").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		text := s.Text()
		switch {
		case strings.Contains(text, "上頁"):
			page.PrevURL = resolve(href)
		case strings.Contains(text, "下頁"):
			page.NextURL = resolve(href)
		}
	})

	doc.Find(".r-ent").Each(func(_ int, s *goquery.Selection) {
		link := s.Find(".title a")
		href, ok := link.Attr("href")
		if !ok {
			return
		}
		page.Entries = append(page.Entries, BoardEntry{
			Title:  strings.TrimSpace(link.Text()),
			Author: strings.TrimSpace(s.Find(".meta .author").Text()),
			Date:   strings.TrimSpace(s.Find(".meta .date").Text()),
			Score:  parseScore(s.Find(".nrec").Text()),
			URL:    resolve(href),
		})
	})

	sort.SliceStable(page.Entries, func(i, j int) bool {
		return page.Entries[i].Score > page.Entries[j].Score
	})
	return page, nil
}

// parseScore 將 PTT 列表的推文數欄位換算成數值：爆為 100，X 開頭為負數，XX 為 -100。
func parseScore(s string) int {
	s = strings.TrimSpace(s)
	switch {
	case s == "":
		return 0
	case s == "爆":
		return 100
	case s == "XX":
		return -100
	case strings.HasPrefix(s, "X"):
		n, err := strconv.Atoi(s[1:])
		if err != nil {
			return 0
		}
		return -n * 10
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

package ptt

import "testing"

const boardFixture = `<!DOCTYPE html>
<html><body>
<div class="r-list-container bbs-screen">
  <div class="r-ent"><div class="nrec"><span class="hl f3">5</span></div><div class="title"><a href="/bbs/Stock/M.1.A.1.html">[新聞] 低熱度</a></div><div class="meta"><div class="author">amy</div><div class="date">11/10</div></div></div>
  <div class="r-ent"><div class="nrec"><span class="hl f1">爆</span></div><div class="title"><a href="/bbs/Stock/M.2.A.2.html">[閒聊] 最熱</a></div><div class="meta"><div class="author">bob</div><div class="date">11/09</div></div></div>
  <div class="r-ent"><div class="nrec"></div><div class="title">(本文已被刪除) [amy]</div><div class="meta"><div class="author">amy</div><div class="date">11/09</div></div></div>
  <div class="r-ent"><div class="nrec"><span class="f2">X2</span></div><div class="title"><a href="/bbs/Stock/M.3.A.3.html">[問卦] 負熱度</a></div><div class="meta"><div class="author">cat</div><div class="date">11/08</div></div></div>
  <div class="r-ent"><div class="nrec"><span class="hl f3">5</span></div><div class="title"><a href="/bbs/Stock/M.4.A.4.html">[心得] 同分</a></div><div class="meta"><div class="author">dan</div><div class="date">11/08</div></div></div>
</div>
<div class="btn-group btn-group-paging">
  <a class="btn wide" href="/bbs/Stock/index100.html">‹ 上頁</a>
  <a class="btn wide" href="/bbs/Stock/index1.html">最舊</a>
  <a class="btn wide disabled">下頁 ›</a>
</div>
</body></html>`

func TestBoardIndexURL(t *testing.T) {
	cases := []struct {
		board, page, want string
		wantErr           bool
	}{
		{"Stock", "", "https://www.ptt.cc/bbs/Stock/index.html", false},
		{"Gossiping", "", "https://www.ptt.cc/bbs/Gossiping/index.html", false},
		{"Stock", "https://www.ptt.cc/bbs/Stock/index100.html", "https://www.ptt.cc/bbs/Stock/index100.html", false},
		{"Stock", "https://www.ptt.cc/bbs/Stock/M.1.A.1.html", "", true},
		{"Stock", "https://www.ptt.cc/bbs/Stock/index/../../x.html", "", true},
		{"Stock/../x", "", "", true},
	}
	for _, c := range cases {
		got, err := BoardIndexURL(c.board, c.page)
		if c.wantErr {
			if err == nil {
				t.Errorf("BoardIndexURL(%q, %q) = %q, want error", c.board, c.page, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("BoardIndexURL(%q, %q) = %q, %v; want %q", c.board, c.page, got, err, c.want)
		}
	}
}

func TestParseBoardIndex(t *testing.T) {
	page, err := ParseBoardIndex("https://www.ptt.cc/bbs/Stock/index.html", boardFixture)
	if err != nil {
		t.Fatal(err)
	}
	if page.Board != "Stock" {
		t.Errorf("board = %q", page.Board)
	}
	if page.PrevURL != "https://www.ptt.cc/bbs/Stock/index100.html" {
		t.Errorf("prevUrl = %q", page.PrevURL)
	}
	if page.NextURL != "" {
		t.Errorf("nextUrl = %q, want empty on latest page", page.NextURL)
	}
	if len(page.Entries) != 4 {
		t.Fatalf("entries = %d, want 4 (deleted post skipped)", len(page.Entries))
	}

	wantOrder := []struct {
		title string
		score int
	}{
		{"[閒聊] 最熱", 100},
		{"[新聞] 低熱度", 5},
		{"[心得] 同分", 5},
		{"[問卦] 負熱度", -20},
	}
	for i, w := range wantOrder {
		e := page.Entries[i]
		if e.Title != w.title || e.Score != w.score {
			t.Errorf("entry %d = %q/%d, want %q/%d", i, e.Title, e.Score, w.title, w.score)
		}
	}
	if page.Entries[0].URL != "https://www.ptt.cc/bbs/Stock/M.2.A.2.html" {
		t.Errorf("entry url = %q", page.Entries[0].URL)
	}
	if page.Entries[0].Author != "bob" || page.Entries[0].Date != "11/09" {
		t.Errorf("entry meta = %q/%q", page.Entries[0].Author, page.Entries[0].Date)
	}
}

func TestParseBoardIndexInvalidPage(t *testing.T) {
	if _, err := ParseBoardIndex("https://www.ptt.cc/bbs/NoSuch/index.html", `<html><body>找不到看板</body></html>`); err == nil {
		t.Fatal("want error for page without article list")
	}
}

func TestParseScore(t *testing.T) {
	cases := map[string]int{"": 0, "爆": 100, "XX": -100, "X1": -10, "X9": -90, "7": 7, "0": 0}
	for in, want := range cases {
		if got := parseScore(in); got != want {
			t.Errorf("parseScore(%q) = %d, want %d", in, got, want)
		}
	}
}

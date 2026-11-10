package ptt

type Push struct {
	Tag     string `json:"tag"`
	User    string `json:"user"`
	Content string `json:"content"`
	IP      string `json:"ip"`
	Date    string `json:"date"`
	Time    string `json:"time"`
	Link    string `json:"link"`
	IsImage bool   `json:"isImage"`
	Seq     int    `json:"seq"`
}

type Article struct {
	URL         string `json:"url"`
	Board       string `json:"board"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Date        string `json:"date"`
	PollURL     string `json:"-"`
	LongPollURL string `json:"-"`
	Pushes      []Push `json:"pushes"`
}

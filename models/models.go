package models

type SongDetailed struct {
	Song
	ReleaseDate string `json:"release_date,omitempty"`
	Text        string `json:"text,omitempty"`
	Link        string `json:"link,omitempty"`
}

type Song struct {
	Group string `json:"group,omitempty"`
	Name  string `json:"name,omitempty"`
}

type Lyrics struct {
	Page int
	Text []string
}

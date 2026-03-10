package recipe

type BrewStep struct {
	Order           int    `json:"order"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	DurationSeconds *int   `json:"duration_seconds,omitempty"`
}

type Recipe struct {
	ID          string      `json:"id"`
	AuthorID    string      `json:"author_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	ImageURL    *string     `json:"image_url,omitempty"`
	BrewMethod  BrewMethod  `json:"brew_method"`
	Difficulty  Difficulty  `json:"difficulty"`
	RoastLevel  *RoastLevel `json:"roast_level,omitempty"`
	Beans       *string     `json:"beans,omitempty"`
	Steps       []BrewStep  `json:"steps"`
}

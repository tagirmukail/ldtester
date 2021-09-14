package url_item

type Item struct {
	Host string `json:"-"`
	Url  string `json:"url"`
}

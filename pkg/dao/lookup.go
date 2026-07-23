package dao

type Book struct {
	Title      string
	Authors    []string
	Year       int
	ISBN       string
	CoverURL   string
	Rating     float64
	SeriesName string
	SeriesPos  float64
	Owned      bool
	WorkKey    string
}

type Author struct {
	Name  string
	Bio   string
	Key   string
	Books []Book
}

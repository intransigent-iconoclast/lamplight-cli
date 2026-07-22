package constants

// Standard Newznab/Torznab categories included in book searches.
// 3030 = Audio/Audiobook is intentionally included so audiobooks pass the --books filter.
var BookCategories = []int{
	3030, // Audio/Audiobook
	7000, // Books
	7010, // Books/Mags
	7020, // Books/EBook
	7030, // Books/Comics
	7040, // Books/Technical
	7050, // Books/Other
	7060, // Books/Foreign
}

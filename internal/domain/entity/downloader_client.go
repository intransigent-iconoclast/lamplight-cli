package entity

type DownloaderType string

const (
	Deluge DownloaderType = "DELUGE"
)

type Downloader struct {
	ID         uint           `gorm:"primaryKey"`
	Name       string         `gorm:"size:75;not null;uniqueIndex"`
	ClientType DownloaderType `gorm:"size:50;not null"`
	Host       string         `gorm:"size:1024;not null"`
	Scheme     string         `gorm:"size:6;not null"`
	Port       int            `gorm:"not null"`
	BaseURL    string         `gorm:"size:1024"`
	Username   string         `gorm:"size:255"`
	Password   string         `gorm:"size:255"`
	Label      string         `gorm:"size:50"`
	Priority   int            `gorm:"default:42"`
	// maybe add certificate path and download directory
}

func (Downloader) TableName() string {
	return "downloader"
}

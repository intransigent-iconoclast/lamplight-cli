package entity

import "time"

type DownloadHistory struct {
	ID             uint      `gorm:"primaryKey"`
	Title          string    `gorm:"size:1024;not null"`
	Link           string    `gorm:"size:2048;not null"`
	IndexerName    string    `gorm:"size:255;not null"`
	DownloaderName string    `gorm:"size:75;not null"`
	SizeBytes      int64     `gorm:"default:0"`
	DownloadedAt   time.Time `gorm:"not null"`
}

func (DownloadHistory) TableName() string {
	return "download_history"
}

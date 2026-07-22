package entity

import "time"

type DownloadStatus string

const (
	StatusSnatched    DownloadStatus = "snatched"    // sent to client, not confirmed yet
	StatusDownloading DownloadStatus = "downloading" // client confirmed it's active
	StatusCompleted   DownloadStatus = "completed"   // done
	StatusFailed      DownloadStatus = "failed"      // something went wrong
)

type DownloadHistory struct {
	ID             uint           `gorm:"primaryKey"`
	Title          string         `gorm:"size:1024;not null"`
	Link           string         `gorm:"size:2048;not null"`
	IndexerName    string         `gorm:"size:255;not null"`
	DownloaderName string         `gorm:"size:75;not null"`
	SizeBytes      int64          `gorm:"default:0"`
	Status         DownloadStatus `gorm:"size:20;not null;default:'snatched'"`
	TorrentHash    string         `gorm:"size:64"`  // hash returned by deluge on add
	FilePath       string         `gorm:"size:2048"` // full path once completed
	DownloadedAt   time.Time      `gorm:"not null"`
}

func (DownloadHistory) TableName() string {
	return "download_history"
}

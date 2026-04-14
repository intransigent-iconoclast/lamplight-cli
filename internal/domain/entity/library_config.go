package entity

type LibraryConfig struct {
	ID          uint   `gorm:"primaryKey;check:id=1"`
	LibraryPath string `gorm:"size:1024;not null"`
	Template    string `gorm:"size:1024;not null"`
	// optional separate root for audiobooks. if empty, audiobooks go into LibraryPath.
	AudiobookPath string `gorm:"size:1024"`
	// when deluge runs in docker it reports paths inside the container.
	// set these two so lamplight can translate to the real host path.
	DelugePath string `gorm:"size:1024"` // path prefix deluge reports, e.g. /data
	HostPath   string `gorm:"size:1024"` // actual host path, e.g. /opt/docker/data/delugevpn/downloads
}

func (LibraryConfig) TableName() string {
	return "library_config"
}

const (
	DefaultLibraryPath = "~/lamplight"
	DefaultTemplate    = "{author}/{title} ({year})"
)

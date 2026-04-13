package entity

type ProviderType string

const (
	ProviderTypeJackett  ProviderType = "JACKETT"
	ProviderTypeProwlarr ProviderType = "PROWLARR"
)

type Provider struct {
	ID      uint         `gorm:"primaryKey"`
	Name    string       `gorm:"size:255;uniqueIndex;not null"`
	Type    ProviderType `gorm:"size:32;not null"`
	Host    string       `gorm:"size:255;not null"`
	Port    int          `gorm:"not null"`
	Scheme  string       `gorm:"size:8;not null;default:http"`
	APIKey  string       `gorm:"size:255;not null"`
	Enabled bool         `gorm:"not null;default:true"`
}

package models

import (
	"time"

	"github.com/lib/pq"
)

type TechnicalSignal struct {
	ID                uint `gorm:"primaryKey"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	PolyStartDuration string `gorm:"not null;"`
	PolyEndDuration   string `gorm:"not null;"`
	PolyTimeSpan      string `gorm:"not null;"`
	PolyMultiplier    int    `gorm:"not null;"`

	StartDate    time.Time `gorm:"not null;"`
	EndDate      time.Time `gorm:"not null;"`
	Interval     string    `gorm:"not null;"`
	WindowSize   int       `gorm:"not null;"`
	Ticker       string    `gorm:"not null;"`
	AnalysisType string    `gorm:"not null;"`

	Signals       pq.StringArray `gorm:"type:text[];not null"`
	FinalDecision string         `gorm:"default ''"`
	UserId        string         `gorm:"not null"`
}

type DeepSearchRequest struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	StartDate string `gorm:"not null;"`
	EndDate   string `gorm:"not null;"`
	Ticker    string `gorm:"not null;"`
	UserId    string `gorm:"not null;"`
}

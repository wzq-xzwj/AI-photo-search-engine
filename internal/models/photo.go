package models

import (
	"time"
)

type Photo struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Size        int64     `json:"size"`
	Format      string    `json:"format"`
	DateTaken   *time.Time `json:"date_taken,omitempty"`
	CameraMake  string    `json:"camera_make,omitempty"`
	CameraModel string    `json:"camera_model,omitempty"`
	GPSLat      *float64  `json:"gps_lat,omitempty"`
	GPSLng      *float64  `json:"gps_lng,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EXIFData struct {
	Make        string
	Model       string
	DateTaken   *time.Time
	GPSLat      *float64
	GPSLng      *float64
	Orientation int
}

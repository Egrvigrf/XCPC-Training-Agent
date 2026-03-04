package model

import (
	"time"

	"gorm.io/gorm"
)

type ContestRecord struct {
	StudentID string `gorm:"column:student_id;primaryKey"`
	Platform  string `gorm:"column:platform;primaryKey"` // CF / AC
	ContestID string `gorm:"column:contest_id;primaryKey"`

	ContestName string    `gorm:"column:contest_name"`
	ContestDate time.Time `gorm:"column:contest_date"`

	ContestRank  int `gorm:"column:contest_rank"`
	OldRating    int `gorm:"column:old_rating"`
	NewRating    int `gorm:"column:new_rating"`
	RatingChange int `gorm:"column:rating_change"`

	Performance int `gorm:"column:performance"`

	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	DeleteAt  gorm.DeletedAt `gorm:"column:delete_at"`
}

func (ContestRecord) TableName() string {
	return "contest_records"
}

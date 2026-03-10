package model

import "time"

type TeamMetricCache struct {
	StatDate time.Time `gorm:"column:stat_date;type:date"`
	Metric   string    `gorm:"column:metric;type:varchar(32)"`

	N    int     `gorm:"column:n"`
	Mean float64 `gorm:"column:mean"`
	Std  float64 `gorm:"column:std"`

	P05 float64 `gorm:"column:p05"`
	P95 float64 `gorm:"column:p95"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (TeamMetricCache) TableName() string {
	return "team_metric_cache"
}

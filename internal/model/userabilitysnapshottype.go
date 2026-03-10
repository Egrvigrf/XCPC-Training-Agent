package model

import "time"

type UserAbilitySnapshot struct {
	StudentID string    `gorm:"column:student_id;type:varchar(32)"`
	StatDate  time.Time `gorm:"column:stat_date;type:date"`

	AvgDifficulty30d float64 `gorm:"column:avg_difficulty_30d"`
	TotalCount30d    int     `gorm:"column:total_count_30d"`
	ContestMean5     float64 `gorm:"column:contest_mean_5"`
	ContestCnt180d   int     `gorm:"column:contest_cnt_180d"`

	ZAvgDiff    float64 `gorm:"column:z_avg_diff"`
	ZTotalCount float64 `gorm:"column:z_total_count"`
	ZContest    float64 `gorm:"column:z_contest"`

	ScoreRaw float64 `gorm:"column:score_raw"`
	Score100 float64 `gorm:"column:score_100"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (UserAbilitySnapshot) TableName() string { return "user_ability_snapshot" }

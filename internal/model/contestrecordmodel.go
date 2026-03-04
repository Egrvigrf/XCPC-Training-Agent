package model

import (
	"context"

	"gorm.io/gorm"
)

type (
	ContestRecordModel interface {
		// Insert 插入一场比赛记录。
		Insert(ctx context.Context, data *ContestRecord) error
		// FindByStudent 查询某用户的全部比赛历史。
		FindByStudent(ctx context.Context, studentID string) ([]*ContestRecord, error)
		// FindRecent 查询某用户最近 N 天的比赛记录。
		FindRecent(ctx context.Context, studentID string, days int) ([]*ContestRecord, error)
		// Delete 删除某场比赛记录。
		Delete(ctx context.Context, studentID, platform, contestID string) error
	}

	defaultContestRecord struct {
		db *gorm.DB
	}
)

func NewContestRecordModel(db *gorm.DB) ContestRecordModel {
	return &defaultContestRecord{db: db}
}

func (m *defaultContestRecord) model() *gorm.DB {
	return m.db.Model(&ContestRecord{})
}

func (m *defaultContestRecord) Insert(ctx context.Context, data *ContestRecord) error {
	return m.model().Create(data).Error
}

func (m *defaultContestRecord) FindByStudent(
	ctx context.Context,
	studentID string,
) ([]*ContestRecord, error) {

	var list []*ContestRecord
	err := m.model().
		Where("student_id = ?", studentID).
		Order("contest_date ASC").
		Find(&list).Error

	return list, err
}

func (m *defaultContestRecord) FindRecent(
	ctx context.Context,
	studentID string,
	days int,
) ([]*ContestRecord, error) {

	var list []*ContestRecord
	err := m.model().
		Where("student_id = ?", studentID).
		Where("contest_date >= NOW() - INTERVAL ? DAY", days).
		Order("contest_date ASC").
		Find(&list).Error

	return list, err
}

func (m *defaultContestRecord) Delete(
	ctx context.Context,
	studentID, platform, contestID string,
) error {

	return m.model().
		Where("student_id = ? AND platform = ? AND contest_id = ?",
			studentID, platform, contestID).
		Delete(&ContestRecord{}).Error
}

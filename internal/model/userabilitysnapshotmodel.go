package model

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserAbilitySnapshotModel interface {
	Upsert(ctx context.Context, data *UserAbilitySnapshot) error
	ListByDate(ctx context.Context, date string) ([]*UserAbilitySnapshot, error)
	FindByDate(ctx context.Context, studentID, date string) (*UserAbilitySnapshot, error)
}

type defaultUserAbilitySnapshotModel struct{ db *gorm.DB }

func NewUserAbilitySnapshotModel(db *gorm.DB) UserAbilitySnapshotModel {
	return &defaultUserAbilitySnapshotModel{db: db}
}

func (m *defaultUserAbilitySnapshotModel) model() *gorm.DB {
	return m.db.Model(&UserAbilitySnapshot{})
}

func (m *defaultUserAbilitySnapshotModel) Upsert(ctx context.Context, data *UserAbilitySnapshot) error {
	return m.model().
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "student_id"}, {Name: "stat_date"}},
			UpdateAll: true,
		}).
		Create(data).Error
}

func (m *defaultUserAbilitySnapshotModel) ListByDate(ctx context.Context, date string) ([]*UserAbilitySnapshot, error) {
	var list []*UserAbilitySnapshot
	err := m.model().WithContext(ctx).
		Where("stat_date = ?", date).
		Find(&list).Error
	return list, err
}

func (m *defaultUserAbilitySnapshotModel) FindByDate(
	ctx context.Context,
	studentID string,
	date string,
) (*UserAbilitySnapshot, error) {

	var data UserAbilitySnapshot

	err := m.model().
		WithContext(ctx).
		Where("student_id = ? AND stat_date = ?", studentID, date).
		First(&data).Error

	if err != nil {
		return nil, err
	}

	return &data, nil
}

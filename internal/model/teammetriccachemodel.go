package model

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TeamMetricCacheModel interface {
	Upsert(ctx context.Context, data *TeamMetricCache) error
	ListByDate(ctx context.Context, date string) ([]*TeamMetricCache, error)
}

type defaultTeamMetricCacheModel struct {
	db *gorm.DB
}

func NewTeamMetricCacheModel(db *gorm.DB) TeamMetricCacheModel {
	return &defaultTeamMetricCacheModel{db: db}
}

func (m *defaultTeamMetricCacheModel) model() *gorm.DB {
	return m.db.Model(&TeamMetricCache{})
}

func (m *defaultTeamMetricCacheModel) Upsert(ctx context.Context, data *TeamMetricCache) error {
	return m.model().
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "stat_date"},
				{Name: "metric"},
			},
			UpdateAll: true,
		}).
		Create(data).Error
}

func (m *defaultTeamMetricCacheModel) ListByDate(ctx context.Context, date string) ([]*TeamMetricCache, error) {
	var list []*TeamMetricCache
	err := m.model().
		WithContext(ctx).
		Where("stat_date = ?", date).
		Find(&list).Error
	return list, err
}

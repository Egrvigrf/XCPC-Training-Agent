package student_data

import (
	"context"
	"math"
	"sort"
	"time"

	"aATA/internal/domain"
	"aATA/internal/model"
	"aATA/internal/svc"
)

type ComputeLogic interface {
	RecomputeAll(ctx context.Context, cutoff time.Time) error
}

type computeLogic struct {
	usersModel     model.UsersModel
	dailyModel     model.DailyTrainingStatsModel
	contestModel   model.ContestRecordModel
	abilityModel   model.UserAbilitySnapshotModel
	teamCacheModel model.TeamMetricCacheModel
	loc            *time.Location
}

func NewComputeLogic(svcCtx *svc.ServiceContext, loc *time.Location) ComputeLogic {
	if loc == nil {
		loc = time.Local
	}
	return &computeLogic{
		usersModel:     svcCtx.UsersModel,
		dailyModel:     svcCtx.DailyModel,
		contestModel:   svcCtx.ContestModel,
		abilityModel:   svcCtx.AbilityModel,
		teamCacheModel: svcCtx.TeamCacheModel,
		loc:            loc,
	}
}

func (l *computeLogic) RecomputeAll(ctx context.Context, cutoff time.Time) error {
	cutoff = dateOnly(cutoff.In(l.loc), l.loc)
	from30 := cutoff.AddDate(0, 0, -30)
	from180 := cutoff.AddDate(0, 0, -180)

	users, _, err := l.usersModel.List(ctx, &domain.UserListReq{})
	if err != nil {
		return err
	}

	snaps := make([]*model.UserAbilitySnapshot, 0, len(users))

	for _, u := range users {
		if u.IsSystem == model.IsSystemUser {
			continue
		}
		if u.CFHandle == "" || u.ACHandle == "" {
			continue
		}

		daily30, _ := l.dailyModel.FindRange(ctx, u.Id, from30, cutoff)

		avgDiff, totalCnt := computeAvgDiffAndCount30d(daily30)
		contestMean5, contestCnt180 := computeContestMean5ByCutoff(ctx, l.contestModel, u.Id, from180, cutoff, avgDiff)

		snaps = append(snaps, &model.UserAbilitySnapshot{
			StudentID: u.Id,
			StatDate:  cutoff,

			AvgDifficulty30d: avgDiff,
			TotalCount30d:    totalCnt,
			ContestMean5:     contestMean5,
			ContestCnt180d:   contestCnt180,
		})
	}

	// 团队 Z-score + cache + score
	if err := computeZAndCacheV1(ctx, l.teamCacheModel, cutoff, snaps); err != nil {
		return err
	}

	// Upsert
	for _, s := range snaps {
		if err := l.abilityModel.Upsert(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func dateOnly(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// 直接按桶加权，不展开
func computeAvgDiffAndCount30d(daily []*model.DailyTrainingStats) (avg float64, total int) {
	var weighted float64
	var cnt int

	for _, d := range daily {
		total += d.CFNewTotal + d.ACNewTotal

		// CF buckets
		add := func(c int, repr float64) {
			if c <= 0 {
				return
			}
			cnt += c
			weighted += float64(c) * repr
		}

		add(d.CFNew800, 800)
		add(d.CFNew900, 900)
		add(d.CFNew1000, 1000)
		add(d.CFNew1100, 1100)
		add(d.CFNew1200, 1200)
		add(d.CFNew1300, 1300)
		add(d.CFNew1400, 1400)
		add(d.CFNew1500, 1500)
		add(d.CFNew1600, 1600)
		add(d.CFNew1700, 1700)
		add(d.CFNew1800, 1800)
		add(d.CFNew1900, 1900)
		add(d.CFNew2000, 2000)
		add(d.CFNew2100, 2100)
		add(d.CFNew2200, 2200)
		add(d.CFNew2300, 2300)
		add(d.CFNew2400, 2400)
		add(d.CFNew2500, 2500)
		add(d.CFNew2600, 2600)
		add(d.CFNew2700, 2700)
		add(d.CFNew2800Plus, 2850)

		// AC buckets (用区间中心值)
		add(d.ACNew0_399, 200)
		add(d.ACNew400_799, 600)
		add(d.ACNew800_1199, 1000)
		add(d.ACNew1200_1599, 1400)
		add(d.ACNew1600_1999, 1800)
		add(d.ACNew2000_2399, 2200)
		add(d.ACNew2400_2799, 2600)
		add(d.ACNew2800Plus, 2850)
	}

	if cnt == 0 {
		return 0, total
	}
	return weighted / float64(cnt), total
}

func computeContestMean5ByCutoff(
	ctx context.Context,
	contestModel model.ContestRecordModel,
	studentID string,
	from180 time.Time,
	cutoff time.Time,
	fallback float64,
) (mean5 float64, cnt180 int) {

	all, err := contestModel.FindByStudent(ctx, studentID) // 按 contest_date ASC :contentReference[oaicite:5]{index=5}
	if err != nil || len(all) == 0 {
		return fallback, 0
	}

	filtered := make([]*model.ContestRecord, 0, 16)
	for _, r := range all {
		if r.ContestDate.Before(from180) || r.ContestDate.After(cutoff) {
			continue
		}
		filtered = append(filtered, r)
	}
	cnt180 = len(filtered)
	if cnt180 == 0 {
		return fallback, 0
	}

	// 取最后 5 场
	k := 5
	if cnt180 < k {
		k = cnt180
	}
	recent := filtered[cnt180-k:]

	// 少于 3 场：不用比赛，避免缺数据导致偏差
	if k < 3 {
		return fallback, cnt180
	}

	var sum float64
	for _, r := range recent {
		sum += float64(r.Performance)
	}
	return sum / float64(k), cnt180
}

func meanStd(vals []float64) (mean, std float64) {
	if len(vals) == 0 {
		return 0, 1
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	mean = sum / float64(len(vals))
	var ss float64
	for _, v := range vals {
		d := v - mean
		ss += d * d
	}
	std = math.Sqrt(ss / float64(len(vals)))
	if std < 1e-6 {
		std = 1
	}
	return mean, std
}

func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	idx := int(math.Round(p * float64(len(vals)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(vals) {
		idx = len(vals) - 1
	}
	return vals[idx]
}

func computeZAndCacheV1(
	ctx context.Context,
	cache model.TeamMetricCacheModel,
	cutoff time.Time,
	snaps []*model.UserAbilitySnapshot,
) error {

	// metric: avg_diff
	{
		vals := make([]float64, 0, len(snaps))
		for _, s := range snaps {
			vals = append(vals, s.AvgDifficulty30d)
		}
		m, sd := meanStd(vals)
		p05 := percentile(append([]float64{}, vals...), 0.05)
		p95 := percentile(append([]float64{}, vals...), 0.95)
		for _, s := range snaps {
			s.ZAvgDiff = (s.AvgDifficulty30d - m) / sd
		}
		if err := cache.Upsert(ctx, &model.TeamMetricCache{StatDate: cutoff, Metric: "avg_diff_30d", N: len(vals), Mean: m, Std: sd, P05: p05, P95: p95}); err != nil {
			return err
		}
	}

	// metric: total_count
	{
		vals := make([]float64, 0, len(snaps))
		for _, s := range snaps {
			vals = append(vals, float64(s.TotalCount30d))
		}
		m, sd := meanStd(vals)
		p05 := percentile(append([]float64{}, vals...), 0.05)
		p95 := percentile(append([]float64{}, vals...), 0.95)
		for _, s := range snaps {
			s.ZTotalCount = (float64(s.TotalCount30d) - m) / sd
		}
		if err := cache.Upsert(ctx, &model.TeamMetricCache{StatDate: cutoff, Metric: "total_cnt_30d", N: len(vals), Mean: m, Std: sd, P05: p05, P95: p95}); err != nil {
			return err
		}
	}

	// metric: contest_mean
	{
		vals := make([]float64, 0, len(snaps))
		for _, s := range snaps {
			vals = append(vals, s.ContestMean5)
		}
		m, sd := meanStd(vals)
		p05 := percentile(append([]float64{}, vals...), 0.05)
		p95 := percentile(append([]float64{}, vals...), 0.95)
		for _, s := range snaps {
			s.ZContest = (s.ContestMean5 - m) / sd
		}
		if err := cache.Upsert(ctx, &model.TeamMetricCache{StatDate: cutoff, Metric: "contest_mean_5", N: len(vals), Mean: m, Std: sd, P05: p05, P95: p95}); err != nil {
			return err
		}
	}

	for _, s := range snaps {
		s.ScoreRaw = 0.6*s.ZAvgDiff + 0.3*s.ZTotalCount + 0.1*s.ZContest
		s.Score100 = 50 + 10*s.ScoreRaw
	}
	return nil
}

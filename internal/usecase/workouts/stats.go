package workouts

import (
	"context"
	"fmt"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// maxStatsDays bounds how far back the Progress tab asks: a year and the month it starts in.
const maxStatsDays = 400

var ErrStatsFrom = domain.Coded(domain.ErrInvalid, "stats_from_invalid", fmt.Sprintf("from must be a time within the last %d days", maxStatsDays))

func init() { ErrStatsFrom.Params = map[string]any{"days": maxStatsDays} }

// WorkoutStats is a workout in figures, for the Progress tab: the phone buckets
// them by its own weeks and days.
type WorkoutStats struct {
	ID        string    `json:"id"`
	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt"`
	VolumeG   int       `json:"volumeG"`
	Sets      int       `json:"sets"`
	// Muscles counts the working sets (warm-ups left out) of each primary muscle trained.
	Muscles map[string]int `json:"muscles"`
}

// Stats lists me's workouts started since from, oldest first, in figures.
func (s *Service) Stats(ctx context.Context, me domain.User, from time.Time) ([]WorkoutStats, error) {
	if from.IsZero() || time.Since(from) > maxStatsDays*24*time.Hour {
		return nil, ErrStatsFrom
	}
	ws, err := s.repo.WorkoutsSince(ctx, me.ID, from)
	if err != nil {
		return nil, err
	}
	out := make([]WorkoutStats, 0, len(ws))
	for _, w := range ws {
		s := WorkoutStats{ID: w.ID, StartedAt: w.StartedAt, EndedAt: w.EndedAt, VolumeG: w.VolumeG, Muscles: map[string]int{}}
		for _, e := range w.Exercises {
			s.Sets += len(e.Sets)
			if e.Exercise == nil {
				continue
			}
			for _, set := range e.Sets {
				if set.Type == domain.SetWarmup {
					continue
				}
				for _, m := range e.Exercise.PrimaryMuscles {
					s.Muscles[m]++
				}
			}
		}
		out = append(out, s)
	}
	return out, nil
}

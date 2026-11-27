package records

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// recordCase is one case of testdata/records.json, which web/src/lib/records.test.ts runs too, so the phone and the
// server agree on what a record is.
type recordCase struct {
	Name     string                `json:"name"`
	Kind     domain.Kind           `json:"kind"`
	Sessions [][]domain.WorkoutSet `json:"sessions"`
	Beaten   []map[RecordKind]int  `json:"beaten"`
	Bests    map[RecordKind]int    `json:"bests"`
}

func TestRecordsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/records.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []recordCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var rows []domain.LoggedSet
			for i, sets := range tc.Sessions {
				for j, s := range sets {
					rows = append(rows, domain.LoggedSet{WorkoutID: fmt.Sprint(i), StartedAt: start.AddDate(0, 0, i), Set: j,
						Type: s.Type, WeightG: s.WeightG, Reps: s.Reps, DurationS: s.DurationS, DistanceM: s.DistanceM})
				}
			}
			h := analyze(tc.Kind, rows)
			if len(h.sessions) != len(tc.Sessions) {
				t.Fatalf("%d sessions, want %d", len(h.sessions), len(tc.Sessions))
			}
			for i, s := range h.sessions {
				got := map[RecordKind]int{}
				for _, r := range s.Records {
					got[r.Kind] = r.Value
				}
				if !maps.Equal(got, tc.Beaten[i]) {
					t.Errorf("session %d beat %v, want %v", i, got, tc.Beaten[i])
				}
			}
			got := map[RecordKind]int{}
			for rk, r := range h.bests {
				got[rk] = r.Value
			}
			if !maps.Equal(got, tc.Bests) {
				t.Errorf("bests %v, want %v", got, tc.Bests)
			}
		})
	}
}

func TestE1RM(t *testing.T) {
	for _, tc := range []struct{ w, reps, want int }{
		{100000, 1, 103333}, {100000, 5, 116667}, {100000, 12, 140000}, {60000, 10, 80000},
	} {
		if got, ok := E1RM(tc.w, tc.reps); !ok || got != tc.want {
			t.Errorf("E1RM(%d, %d) = %d, %v; want %d", tc.w, tc.reps, got, ok, tc.want)
		}
	}
	for _, reps := range []int{0, 13, -1} {
		if _, ok := E1RM(100000, reps); ok {
			t.Errorf("E1RM at %d reps should not count", reps)
		}
	}
}

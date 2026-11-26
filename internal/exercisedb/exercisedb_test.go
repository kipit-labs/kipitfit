package exercisedb

import (
	"testing"

	"github.com/kipit-labs/kipitfit/internal/store"
)

func TestKind(t *testing.T) {
	for _, c := range []struct {
		name, category, equipment, force string
		want                             store.Kind
	}{
		{"Barbell Bench Press - Medium Grip", "strength", "barbell", "push", store.KindWeightReps},
		{"Pullups", "strength", "body_only", "pull", store.KindRepsOnly},
		{"Plank", "strength", "body_only", "static", store.KindDuration},
		{"Hamstring Stretch", "stretching", "none", "static", store.KindDuration},
		{"Running, Treadmill", "cardio", "machine", "", store.KindDistanceDuration},
		{"Band Assisted Pull-Up", "strength", "other", "pull", store.KindAssistedBodyweight},
		{"Weighted Pull Ups", "strength", "other", "pull", store.KindWeightedBodyweight},
		{"Weighted Jump Squat", "strength", "barbell", "push", store.KindWeightReps},
		{"Weighted Crunches", "strength", "medicine_ball", "pull", store.KindWeightReps},
	} {
		e := store.Exercise{Name: c.name, Category: c.category, Equipment: c.equipment, Force: c.force}
		if got := Kind(e); got != c.want {
			t.Errorf("Kind(%q) = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestLoad(t *testing.T) {
	exs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(exs) != 876 {
		t.Fatalf("got %d exercises, want 876", len(exs))
	}
	slugs := map[string]bool{}
	for _, e := range exs {
		if e.Slug == "" || slugs[e.Slug] || e.Name == "" || len(e.PrimaryMuscles) == 0 || e.Kind == "" {
			t.Fatalf("bad exercise %+v", e)
		}
		slugs[e.Slug] = true
	}
	if !slugs["Barbell_Bench_Press_-_Medium_Grip"] {
		t.Error("bench press missing")
	}
	if Version() == "" {
		t.Error("no version")
	}
}

package library

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

var builtin = fakeDataset{
	{Slug: "Barbell_Deadlift", Name: "Barbell Deadlift", Kind: domain.KindWeightReps},
	{Slug: "3_4_Sit-Up", Name: "3/4 Sit-Up", Kind: domain.KindRepsOnly},
	{Slug: "Barbell_Squat", Name: "Barbell Squat", Kind: domain.KindWeightReps},
}

func testPack() domain.ParsedPack {
	ex := func(slug, name string, kind domain.Kind, files ...domain.PackFile) domain.PackExercise {
		return domain.PackExercise{Exercise: domain.Exercise{Slug: slug, Name: name, Kind: kind, Category: "strength",
			Equipment: "barbell", PrimaryMuscles: []string{"glutes"}}, Files: files}
	}
	return domain.ParsedPack{Pack: domain.Pack{Name: "test", Version: "1", Credit: "Exercise data by Test"}, Exercises: []domain.PackExercise{
		ex("test:deadlift", "barbell  deadlift", domain.KindWeightReps, file("packs/test/1/deadlift-start.webp"), file("packs/test/1/deadlift-peak.webp")),
		ex("test:plank", "Plank", domain.KindDuration, file("packs/test/1/plank-main.webp")),
		// Same name as a built-in one, other kind: its sets wouldn't fit.
		ex("test:situp", "3/4 Sit-Up", domain.KindWeightReps),
	}}
}

func newService(lib *fakeLibrary, m domain.MediaStore) *Service {
	return NewService(lib, lib, builtin, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestSyncExercises(t *testing.T) {
	ctx := context.Background()
	lib := &fakeLibrary{}
	s := newService(lib, nil)
	if err := s.SyncExercises(ctx); err != nil || lib.syncs != 1 || len(lib.exercises) != len(builtin) || lib.version != "v1" {
		t.Fatalf("first start: %v, %d syncs of %d exercises at %q", err, lib.syncs, len(lib.exercises), lib.version)
	}
	if err := s.SyncExercises(ctx); err != nil || lib.syncs != 1 { // second start: no-op
		t.Fatalf("second start: %v, %d syncs", err, lib.syncs)
	}
	if l, err := s.Library(ctx); err != nil || l.Pack != nil || l.Media {
		t.Fatalf("library = %+v, %v", l, err)
	}
}

func TestImportPack(t *testing.T) {
	ctx := context.Background()
	lib := &fakeLibrary{}
	if err := newService(lib, nil).SyncExercises(ctx); err != nil {
		t.Fatal(err)
	}
	lib.exercises = append(lib.exercises, domain.Exercise{ID: "mine", OwnerID: "me", Name: "Landmine Press", Kind: domain.KindWeightReps, Custom: true})
	if _, err := newService(lib, nil).ImportPack(ctx, testPack()); err == nil {
		t.Fatal("import with media off")
	}
	m := &fakeMedia{objs: map[string]string{}}
	s := newService(lib, m)

	res, err := s.ImportPack(ctx, testPack())
	if err != nil {
		t.Fatal(err)
	}
	if res.Uploaded != 3 || res.Skipped != 0 || res.Exercises != 3 {
		t.Fatalf("import = %+v", res)
	}
	// Only the deadlift moves: the sit-up's kind differs, the squat isn't in the pack, a custom exercise is no library one.
	if len(lib.repoint) != 1 || lib.repoint["Barbell_Deadlift"] != "test:deadlift" {
		t.Fatalf("repoint = %v", lib.repoint)
	}
	if len(lib.replaced) != 3 || len(lib.replaced[0].Media) != 2 || lib.replaced[0].Media[1] != "packs/test/1/deadlift-peak.webp" ||
		lib.replaced[2].Media == nil {
		t.Fatalf("pack exercises = %+v", lib.replaced)
	}
	if l, _ := s.Library(ctx); l.Pack == nil || l.Pack.Credit != "Exercise data by Test" || !l.Media {
		t.Fatalf("library = %+v", l)
	}

	// The next start doesn't bring the built-in library back; a re-run skips what's uploaded.
	if err := s.SyncExercises(ctx); err != nil || lib.syncs != 1 {
		t.Fatalf("sync after import: %v, %d syncs", err, lib.syncs)
	}
	res, err = s.ImportPack(ctx, testPack())
	if err != nil || res.Uploaded != 0 || res.Skipped != 3 || len(lib.repoint) != 0 || m.puts != 3 {
		t.Fatalf("re-run = %+v, %v (puts %d)", res, err, m.puts)
	}

	bad := testPack()
	bad.Exercises[0].Files = append(bad.Exercises[0].Files, file("users/x.webp"))
	if _, err := s.ImportPack(ctx, bad); err == nil {
		t.Error("a key the API won't serve was imported")
	}
	if _, err := s.ImportPack(ctx, domain.ParsedPack{}); err == nil {
		t.Error("an empty pack was imported")
	}
}

func TestMedia(t *testing.T) {
	ctx := context.Background()
	if _, _, err := newService(&fakeLibrary{}, nil).Media(ctx, "packs/a.webp"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("media off: %v", err)
	}
	m := &fakeMedia{objs: map[string]string{"packs/a.webp": "webp", "users/b.webp": "private"}}
	s := newService(&fakeLibrary{}, m)
	obj, info, err := s.Media(ctx, "packs/a.webp")
	if err != nil || info.Size != 4 {
		t.Fatalf("get = %+v, %v", info, err)
	}
	_ = obj.Close()
	for _, key := range []string{"users/b.webp", "packs/missing.webp"} {
		if _, _, err := s.Media(ctx, key); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: %v", key, err)
		}
	}
}

func TestNormName(t *testing.T) {
	if normName("Pull-Up") != normName(" pull up ") || normName("Pull-Up") == normName("Push-Up") {
		t.Fatal("normName")
	}
}

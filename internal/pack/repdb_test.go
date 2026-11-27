package pack_test

import (
	"archive/zip"
	"bytes"
	"io"
	"slices"
	"testing"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/pack"
)

// A made-up pack in RepDB's free-tier shape; no real RepDB data is in the repo.
const freeJSON = `{"schema_version":3,"generated_at":"2026-11-14T12:11:04Z","exercises":[
{"id":"squat","name_en":"Barbell Back Squat","category":"strength","force_type":"push","mechanic":"compound","difficulty":"advanced",
 "equipment":"barbell","primary_muscles":["gluteus_maximus","quadriceps"],"secondary_muscles":["erector_spinae","gluteus_medius"],
 "instructions_en":["Squat."],"images":{"flat":["start","peak"]}},
{"id":"plank","name_en":"Plank","category":"strength","force_type":"static","difficulty":"beginner","equipment":null,
 "primary_muscles":["rectus_abdominis"],"secondary_muscles":null,"images":{"flat":["main"]}},
{"id":"pull-up","name_en":"Pull-Up","category":"strength","force_type":"pull","equipment":"pull_up_bar",
 "primary_muscles":["latissimus_dorsi","mystery_muscle"],"images":{"flat":["start","peak"]}},
{"id":"ball-plank","name_en":"Ball Plank","category":"strength","force_type":"static","equipment":"bosu_ball","primary_muscles":["obliques"]},
{"id":"row","name_en":"Rowing Machine","category":"cardio","force_type":"dynamic","equipment":"rower","primary_muscles":["latissimus_dorsi"]},
{"id":"clean","name_en":"Power Clean","category":"olympic","equipment":"sled_machine","primary_muscles":["hip_flexors"]},
{"id":"../evil","name_en":"Evil","category":"strength","equipment":"sandbag","primary_muscles":["trapezius"]}
]}`

func testZip(t *testing.T, files map[string]string) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, _ := w.Create(name)
		_, _ = f.Write([]byte(body))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return z
}

func TestRepDBFree(t *testing.T) {
	z := testZip(t, map[string]string{
		"free.json":                    freeJSON,
		"images/flat/squat-start.webp": "s0", "images/flat/squat-peak.webp": "s1",
		"images/flat/plank-main.webp": "p0",
		// pull-up-peak.webp is missing: the exercise keeps the frames that exist.
		"images/flat/pull-up-start.webp": "u0",
	})
	p, err := pack.Adapters["repdb-free"].Read(z)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "repdb-free" || p.Version != "2026-11-14T121104Z" || p.Credit != "Exercise data by RepDB (repdb.co)" || len(p.Exercises) != 7 {
		t.Fatalf("pack = %+v (%d exercises)", p.Pack, len(p.Exercises))
	}
	by := map[string]domain.PackExercise{}
	for _, e := range p.Exercises {
		by[e.Slug] = e
		for _, list := range [][]string{e.PrimaryMuscles, e.SecondaryMuscles} {
			for _, m := range list {
				if !slices.Contains(domain.Muscles, m) {
					t.Errorf("%s: muscle %q isn't a kipitfit key", e.Slug, m)
				}
			}
		}
		if !slices.Contains(domain.Equipment, e.Equipment) || !slices.Contains(domain.Categories, e.Category) || !slices.Contains(domain.Kinds, e.Kind) {
			t.Errorf("%s: unknown key in %+v", e.Slug, e.Exercise)
		}
		for _, f := range e.Files {
			if !domain.ValidMediaKey(f.Key) {
				t.Errorf("%s: key %q not servable", e.Slug, f.Key)
			}
		}
	}
	sq := by["repdb:squat"]
	if sq.Name != "Barbell Back Squat" || sq.Kind != domain.KindWeightReps || sq.Level != "expert" || sq.Force != "push" ||
		!slices.Equal(sq.PrimaryMuscles, []string{"glutes", "quadriceps"}) || !slices.Equal(sq.SecondaryMuscles, []string{"lower_back"}) ||
		len(sq.Files) != 2 || sq.Files[0].Key != "packs/repdb-free/2026-11-14T121104Z/squat-start.webp" || sq.Files[0].ContentType != "image/webp" {
		t.Errorf("squat = %+v", sq)
	}
	r, _ := sq.Files[1].Open()
	if b, _ := io.ReadAll(r); string(b) != "s1" {
		t.Errorf("squat peak = %q", b)
	}
	for slug, want := range map[string]struct {
		kind      domain.Kind
		equipment string
		files     int
	}{
		"repdb:plank":      {domain.KindDuration, "body_only", 1},
		"repdb:pull-up":    {domain.KindRepsOnly, "body_only", 1},
		"repdb:ball-plank": {domain.KindDuration, "exercise_ball", 0},
		"repdb:row":        {domain.KindDistanceDuration, "machine", 0},
		"repdb:clean":      {domain.KindWeightReps, "machine", 0},
		"repdb:..evil":     {domain.KindWeightReps, "other", 0}, // "/" dropped from the id
	} {
		e := by[slug]
		if e.Kind != want.kind || e.Equipment != want.equipment || len(e.Files) != want.files {
			t.Errorf("%s = kind %s, equipment %s, %d files; want %+v", slug, e.Kind, e.Equipment, len(e.Files), want)
		}
	}
	if by["repdb:clean"].Category != "olympic_weightlifting" || by["repdb:row"].Force != "" || len(by["repdb:pull-up"].PrimaryMuscles) != 1 {
		t.Errorf("mapping: %+v %+v", by["repdb:clean"].Exercise, by["repdb:pull-up"].Exercise)
	}

	if _, err := pack.Adapters["repdb-free"].Read(testZip(t, map[string]string{"other.json": "{}"})); err == nil {
		t.Error("a ZIP without free.json was read")
	}
}

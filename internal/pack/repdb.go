package pack

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/kipit-labs/kipitfit/internal/exercisedb"
	"github.com/kipit-labs/kipitfit/internal/store"
)

// RepDB's muscles, equipment, categories and levels onto kipitfit's keys.
var (
	repdbMuscles = map[string]string{
		"gluteus_maximus": "glutes", "gluteus_medius": "glutes",
		"anterior_deltoid": "shoulders", "lateral_deltoid": "shoulders", "posterior_deltoid": "shoulders",
		"infraspinatus_teres_minor": "shoulders",
		"hamstrings":                "hamstrings", "quadriceps": "quadriceps", "adductors": "adductors", "abductors": "abductors",
		"hip_flexors":     "hip_flexors",
		"triceps_brachii": "triceps", "biceps_brachii": "biceps", "brachialis": "biceps",
		"brachioradialis": "forearms", "forearm_flexors": "forearms", "forearm_extensors": "forearms", "forearms": "forearms",
		"erector_spinae": "lower_back", "quadratus_lumborum": "lower_back",
		"trapezius": "traps", "rhomboids": "middle_back", "latissimus_dorsi": "lats",
		"pectoralis_major": "chest", "serratus_anterior": "chest",
		"rectus_abdominis": "abdominals", "obliques": "abdominals", "transverse_abdominis": "abdominals",
		"gastrocnemius": "calves", "soleus": "calves", "tibialis_anterior": "calves",
	}
	repdbEquipment = map[string]string{
		"barbell": "barbell", "trap_bar": "barbell", "dumbbell": "dumbbell", "kettlebell": "kettlebells",
		"cable": "cable", "ez_bar": "ez_curl_bar", "resistance_band": "bands", "loop_band": "bands",
		"medicine_ball": "medicine_ball", "wall_ball": "medicine_ball", "slam_ball": "medicine_ball",
		"stability_ball": "exercise_ball", "bosu_ball": "exercise_ball",
		// Things you hang from or stand on: the exercise is body weight.
		"pull_up_bar": "body_only", "dip_station": "body_only", "rings": "body_only", "suspension_trainer": "body_only",
		"flat_bench": "body_only", "plyo_box": "body_only",
	}
	// RepDB names machines one by one; any unknown one ending in _machine is a machine too.
	repdbMachines = []string{"smith_machine", "leg_press", "leg_curl", "leg_extension", "hack_squat", "pec_deck",
		"glute_ham_developer", "treadmill", "stationary_bike", "elliptical", "air_bike", "rower", "ski_erg", "stair_climber"}
	repdbCategories = map[string]string{"strength": "strength", "olympic": "olympic_weightlifting",
		"plyometrics": "plyometrics", "stretching": "stretching", "cardio": "cardio"}
	repdbLevels = map[string]string{"beginner": "beginner", "intermediate": "intermediate", "advanced": "expert"}
)

type repdbFile struct {
	SchemaVersion int    `json:"schema_version"`
	GeneratedAt   string `json:"generated_at"`
	Exercises     []struct {
		ID           string   `json:"id"`
		Name         string   `json:"name_en"`
		Category     string   `json:"category"`
		Force        string   `json:"force_type"`
		Mechanic     string   `json:"mechanic"`
		Difficulty   string   `json:"difficulty"`
		Equipment    *string  `json:"equipment"`
		Primary      []string `json:"primary_muscles"`
		Secondary    []string `json:"secondary_muscles"`
		Instructions []string `json:"instructions_en"`
		Images       struct {
			Flat []string `json:"flat"`
		} `json:"images"`
	} `json:"exercises"`
}

// readRepDBFree reads RepDB's free tier: free.json and images/flat/<id>-<frame>.webp.
func readRepDBFree(z *zip.Reader) (Pack, error) {
	f, ok := file(z, "free.json", "")
	if !ok {
		return Pack{}, fmt.Errorf("free.json not in the ZIP: is this RepDB's free download?")
	}
	r, err := f.Open()
	if err != nil {
		return Pack{}, err
	}
	defer r.Close()
	var data repdbFile
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return Pack{}, fmt.Errorf("free.json: %w", err)
	}
	version := keySafe(data.GeneratedAt)
	if version == "" || len(data.Exercises) == 0 {
		return Pack{}, fmt.Errorf("free.json: no version or no exercises")
	}
	p := Pack{Pack: store.Pack{Name: "repdb-free", Version: version,
		Credit: "Exercise data by RepDB (repdb.co)", CreditURL: "https://repdb.co"}}
	for _, x := range data.Exercises {
		id := keySafe(x.ID)
		if id == "" || x.Name == "" {
			continue
		}
		e := Exercise{Exercise: store.Exercise{
			Slug: "repdb:" + id, Name: x.Name, Category: repdbCategories[x.Category],
			Mechanic: x.Mechanic, Level: repdbLevels[x.Difficulty], Instructions: x.Instructions,
		}}
		if e.Category == "" {
			e.Category = "strength"
		}
		if x.Force == "push" || x.Force == "pull" || x.Force == "static" {
			e.Force = x.Force
		}
		raw := ""
		if x.Equipment != nil {
			raw = *x.Equipment
		}
		e.Equipment = repdbEquipmentKey(raw)
		e.PrimaryMuscles = repdbMuscleKeys(x.Primary, nil)
		e.SecondaryMuscles = repdbMuscleKeys(x.Secondary, e.PrimaryMuscles)
		k := e.Exercise
		if raw == "bosu_ball" || raw == "stability_ball" || raw == "ab_wheel" {
			k.Equipment = "body_only" // a plank on a ball is still a timed hold
		}
		e.Kind = exercisedb.Kind(k)
		for _, frame := range x.Images.Flat {
			name := id + "-" + keySafe(frame) + ".webp"
			if f, ok := file(z, "images/flat/"+name, "packs/repdb-free/"+version+"/"+name); ok {
				e.Files = append(e.Files, f)
			}
		}
		p.Exercises = append(p.Exercises, e)
	}
	return p, nil
}

func repdbEquipmentKey(raw string) string {
	switch {
	case raw == "":
		return "body_only"
	case repdbEquipment[raw] != "":
		return repdbEquipment[raw]
	case slices.Contains(repdbMachines, raw) || len(raw) > 8 && raw[len(raw)-8:] == "_machine":
		return "machine"
	default:
		return "other" // sled, sandbag, battle rope, ab wheel…
	}
}

// repdbMuscleKeys maps muscles to kipitfit's keys, sorted, without repeats
// and without those already in skip.
func repdbMuscleKeys(in, skip []string) []string {
	out := []string{}
	for _, m := range in {
		if k := repdbMuscles[m]; k != "" && !slices.Contains(skip, k) {
			out = append(out, k)
		}
	}
	return slices.Compact(slices.Sorted(slices.Values(out)))
}

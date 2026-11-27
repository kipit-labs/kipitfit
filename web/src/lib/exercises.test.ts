import { describe, expect, it } from "vitest";
import type { TFunction } from "i18next";

import type { ExerciseSummary } from "./api";
import { categories, equipment, filterExercises, kinds, muscles } from "./exercises";
import en from "@/locales/en.json";
import fr from "@/locales/fr.json";

const ex = (name: string, more: Partial<ExerciseSummary> = {}): ExerciseSummary => ({
  id: name,
  name,
  kind: "weight_reps",
  category: "strength",
  equipment: "barbell",
  primaryMuscles: ["chest"],
  secondaryMuscles: [],
  custom: false,
  archived: false,
  ...more,
});
const labels: Record<string, string> = { "exercises.muscles.chest": "Pectoraux", "exercises.muscles.quadriceps": "Quadriceps" };
const t = ((k: string) => labels[k] ?? k) as unknown as TFunction;
const none = { q: "", muscle: "", equipment: "", category: "" };

describe("filterExercises", () => {
  const list = [
    ex("Barbell Bench Press - Medium Grip"),
    ex("Écarté couché", { equipment: "dumbbell" }),
    ex("Barbell Squat", { primaryMuscles: ["quadriceps"], category: "powerlifting" }),
  ];
  const names = (f: Partial<typeof none>) => filterExercises(list, { ...none, ...f }, t).map((e) => e.name);

  it("matches words in any order, ignoring case and accents", () => {
    expect(names({ q: "bench" })).toEqual(["Barbell Bench Press - Medium Grip"]);
    expect(names({ q: "GRIP barbell" })).toEqual(["Barbell Bench Press - Medium Grip"]);
    expect(names({ q: "ecarte" })).toEqual(["Écarté couché"]);
  });
  it("matches translated muscle labels", () => {
    expect(names({ q: "pectoraux" })).toHaveLength(2);
  });
  it("applies every filter", () => {
    expect(names({ muscle: "chest" })).toHaveLength(2);
    expect(names({ muscle: "chest", equipment: "dumbbell" })).toEqual(["Écarté couché"]);
    expect(names({ category: "powerlifting", q: "squat" })).toEqual(["Barbell Squat"]);
  });
});

describe("exercise labels", () => {
  it("has a label for every value in both catalogs", () => {
    const groups = { kinds, categories, equipment, muscles };
    for (const cat of [en, fr]) {
      for (const [group, values] of Object.entries(groups)) {
        const have = (cat.exercises as unknown as Record<string, Record<string, string>>)[group];
        expect(values.filter((v) => !have[v])).toEqual([]);
      }
    }
  });
});

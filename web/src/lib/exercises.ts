import {
  Cable,
  CircleDot,
  Cog,
  Cylinder,
  Dumbbell,
  PersonStanding,
  Shapes,
  Spline,
  Volleyball,
  Waves,
  Weight,
  type LucideIcon,
} from "lucide-react";
import type { TFunction } from "i18next";

import { api, type ExerciseKind, type ExerciseSummary } from "./api";

// Same values as internal/core/exercises.go; labels are exercises.<group>.<value>.
export const kinds: ExerciseKind[] = [
  "weight_reps",
  "reps_only",
  "weighted_bodyweight",
  "assisted_bodyweight",
  "duration",
  "distance_duration",
];
export const categories = [
  "strength",
  "stretching",
  "cardio",
  "plyometrics",
  "powerlifting",
  "olympic_weightlifting",
  "strongman",
];
export const muscles = [
  "abdominals",
  "abductors",
  "adductors",
  "biceps",
  "calves",
  "chest",
  "forearms",
  "glutes",
  "hamstrings",
  "lats",
  "lower_back",
  "middle_back",
  "neck",
  "quadriceps",
  "shoulders",
  "traps",
  "triceps",
];

const icons: Record<string, LucideIcon> = {
  barbell: Dumbbell,
  dumbbell: Dumbbell,
  ez_curl_bar: Spline,
  kettlebells: Weight,
  cable: Cable,
  machine: Cog,
  bands: Waves,
  body_only: PersonStanding,
  none: PersonStanding,
  exercise_ball: Volleyball,
  medicine_ball: CircleDot,
  foam_roll: Cylinder,
};
export const equipment = Object.keys(icons).concat("other");
export const equipmentIcon = (e: string): LucideIcon => icons[e] ?? Shapes;

export const exercisesQuery = { queryKey: ["exercises"], queryFn: api.exercises, staleTime: 5 * 60_000 };

/** Lower case without accents, so "écarté" matches "ecarte". */
export const fold = (s: string) =>
  s
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase();

export type Filters = { q: string; muscle: string; equipment: string; category: string };

/**
 * The exercises matching every word of q, in any order, against the name and
 * the muscle and equipment labels in the user's language, and every filter set.
 */
export function filterExercises(list: ExerciseSummary[], f: Filters, t: TFunction): ExerciseSummary[] {
  const words = fold(f.q).split(/\s+/).filter(Boolean);
  return list.filter((e) => {
    if (f.muscle && !e.primaryMuscles.includes(f.muscle)) return false;
    if (f.equipment && e.equipment !== f.equipment) return false;
    if (f.category && e.category !== f.category) return false;
    if (!words.length) return true;
    const text = fold(
      [e.name, t(`exercises.equipment.${e.equipment}`), ...e.primaryMuscles.map((m) => t(`exercises.muscles.${m}`))].join(" "),
    );
    return words.every((w) => text.includes(w));
  });
}

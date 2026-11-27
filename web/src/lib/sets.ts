import type { ExerciseKind, SetType } from "./api";

export const setTypes: SetType[] = ["normal", "warmup", "drop", "failure"];

/** What the set-number button shows: the count among normal sets, or the type's letter. */
export function setLabels(types: SetType[], letters: Record<SetType, string>): string[] {
  let n = 0;
  return types.map((t) => (t === "normal" ? String(++n) : letters[t]));
}

/** The values a set of an exercise kind takes; the server checks the same. */
export function fieldsOf(kind: ExerciseKind) {
  return {
    weight: kind === "weight_reps" || kind === "weighted_bodyweight" || kind === "assisted_bodyweight",
    reps: kind === "weight_reps" || kind === "weighted_bodyweight" || kind === "assisted_bodyweight" || kind === "reps_only",
    distance: kind === "distance_duration",
    duration: kind === "duration" || kind === "distance_duration",
  };
}

/** The weight column's sign: added weight, or assistance taken off. */
export const weightSign = (kind: ExerciseKind) =>
  kind === "weighted_bodyweight" ? "+" : kind === "assisted_bodyweight" ? "−" : "";

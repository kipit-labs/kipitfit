import type { ExerciseKind, SetType, Unit } from "./api";
import { formatDuration, formatKm, formatWeight } from "./units";

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

/** A logged set in a line: "100 × 5" ("100 kg × 5" withUnit), "+20 × 8", "5 km · 25:00", "1:30". */
export function setText(
  p: { weightG?: number | null; reps?: number | null; durationS?: number | null; distanceM?: number | null },
  kind: ExerciseKind,
  unit: Unit,
  withUnit = false,
): string {
  const f = fieldsOf(kind);
  if (f.distance) return [p.distanceM != null && `${formatKm(p.distanceM)} km`, p.durationS != null && formatDuration(p.durationS)].filter(Boolean).join(" · ");
  if (f.duration) return p.durationS == null ? "—" : formatDuration(p.durationS);
  const w = f.weight && p.weightG != null ? weightSign(kind) + formatWeight(p.weightG, unit) + (withUnit ? ` ${unit}` : "") : "";
  return `${w}${w ? " " : ""}× ${p.reps ?? "—"}`;
}

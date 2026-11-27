import type { ExerciseKind, RecordKind, Unit } from "./api";
import { formatDuration, formatKm, formatWeight } from "./units";

/** The records an exercise kind keeps, in the order the page shows them; internal/core/records.go has the same. */
export function recordKinds(k: ExerciseKind): RecordKind[] {
  switch (k) {
    case "weight_reps":
      return ["e1rm", "weight", "set_volume", "session_volume", "reps"];
    case "weighted_bodyweight":
      return ["weight", "set_volume", "session_volume", "reps"];
    case "reps_only":
    case "assisted_bodyweight":
      return ["reps", "session_reps"];
    case "duration":
      return ["duration"];
    case "distance_duration":
      return ["distance", "duration"];
  }
}

/** What the exercise page can chart, per workout. */
export const chartKinds = (k: ExerciseKind): RecordKind[] => recordKinds(k).filter((rk) => rk !== "set_volume");

const weighs = (rk: RecordKind) => rk === "e1rm" || rk === "weight" || rk === "set_volume" || rk === "session_volume";

/** A record's value with its unit: "111 kg", "12", "1:30", "5.2 km". */
export function formatRecord(rk: RecordKind, value: number, unit: Unit): string {
  if (weighs(rk)) return `${formatWeight(value, unit)} ${unit}`;
  if (rk === "duration") return formatDuration(value);
  if (rk === "distance") return `${formatKm(value)} km`;
  return String(value);
}

/** A record's value as a number to chart: weights in unit, distance in km, the rest as is. */
export function chartValue(rk: RecordKind, value: number, unit: Unit): number {
  if (weighs(rk)) return Number(formatWeight(value, unit));
  if (rk === "distance") return value / 1000;
  return value;
}

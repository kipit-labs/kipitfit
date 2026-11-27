import type { ExerciseKind, RecordKind, SetType, Unit } from "./api";
import { formatDuration, formatKm, formatNumber, formatWeight, weightIn } from "./units";

/** The records an exercise kind keeps, in the order the page shows them; internal/usecase/records/records.go has the same. */
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

/** A record's value with its unit: "111.0 kg" (an estimate, to 0.1 kg), "100 kg", "12", "1:30", "5.2 km". */
export function formatRecord(rk: RecordKind, value: number, unit: Unit): string {
  if (rk === "e1rm" && unit === "kg") return `${formatNumber(Math.round(value / 100) / 10, { minimumFractionDigits: 1, maximumFractionDigits: 1 })} kg`;
  if (weighs(rk)) return `${formatWeight(value, unit)} ${unit}`;
  if (rk === "duration") return formatDuration(value);
  if (rk === "distance") return `${formatKm(value)} km`;
  return String(value);
}

/** A record's value as a number to chart: weights in unit, distance in km, the rest as is. */
export function chartValue(rk: RecordKind, value: number, unit: Unit): number {
  if (weighs(rk)) return weightIn(value, unit);
  if (rk === "distance") return value / 1000;
  return value;
}

/** A set as records see it: a logged one, or a live one, which counts once ticked done. */
type SetValues = {
  type: SetType;
  done?: boolean;
  weightG?: number | null;
  reps?: number | null;
  durationS?: number | null;
  distanceM?: number | null;
};

const e1rmMaxReps = 12;

/** Epley's estimated one-rep max in grams, rounded half up; undefined outside 1 to 12 reps. */
export function e1rm(weightG: number, reps: number): number | undefined {
  if (reps < 1 || reps > e1rmMaxReps || weightG <= 0) return undefined;
  return Math.floor((weightG * (30 + reps) + 15) / 30);
}

const pos = (v: number | null | undefined): v is number => v != null && v > 0;

/** What one set counts for a record kind; undefined when it doesn't (warm-ups never do). */
function setValue(rk: RecordKind, s: SetValues): number | undefined {
  if (s.type === "warmup" || s.done === false) return undefined;
  switch (rk) {
    case "e1rm":
      return s.weightG != null && s.reps != null ? e1rm(s.weightG, s.reps) : undefined;
    case "weight":
      return pos(s.weightG) ? s.weightG : undefined;
    case "set_volume":
    case "session_volume":
      return pos(s.weightG) && pos(s.reps) ? s.weightG * s.reps : undefined;
    case "reps":
    case "session_reps":
      return pos(s.reps) ? s.reps : undefined;
    case "duration":
      return pos(s.durationS) ? s.durationS : undefined;
    case "distance":
      return pos(s.distanceM) ? s.distanceM : undefined;
  }
}

const summed = (rk: RecordKind) => rk === "session_volume" || rk === "session_reps";

export type Beaten = { kind: RecordKind; value: number; previous: number; /** -1 for a workout total. */ set: number };

/**
 * The records one workout's sets of an exercise beat, given the bests before it, as internal/usecase/records/records.go
 * works them out: its best set (the first on a tie) or its total, strictly above the best. An exercise without
 * bests was never done, and a first workout sets none.
 */
export function beaten(kind: ExerciseKind, sets: SetValues[], bests: Partial<Record<RecordKind, number>> | undefined): Beaten[] {
  const out: Beaten[] = [];
  for (const rk of recordKinds(kind)) {
    const prev = bests?.[rk];
    const b = sessionBest(rk, sets);
    if (b && prev !== undefined && b.value > prev) out.push({ kind: rk, value: b.value, previous: prev, set: b.set });
  }
  return out;
}

/** A workout's value for a record kind and the set holding it; undefined when no set counts. */
export function sessionBest(rk: RecordKind, sets: SetValues[]): { value: number; set: number } | undefined {
  let best: { value: number; set: number } | undefined;
  sets.forEach((s, j) => {
    const v = setValue(rk, s);
    if (v === undefined) return;
    if (summed(rk)) best = { value: (best?.value ?? 0) + v, set: -1 };
    else if (!best || v > best.value) best = { value: v, set: j };
  });
  return best;
}

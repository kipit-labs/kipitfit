// The live workout: a pure model the device keeps in localStorage after every change.
import type { ExerciseKind, ExerciseSummary, RecordKind, Routine, RoutineInput, RoutineSet, SetType } from "@/lib/api";
import { beaten } from "@/lib/records";
import { defaultRestS, leaveSuperset, move, relabel, supersetWithNext } from "@/lib/routines";
import { fieldsOf } from "@/lib/sets";

export type LiveSet = {
  type: SetType;
  weightG: number | null;
  reps: number | null;
  /** 6 to 10 by 0.5; undefined until the set's menu turns it on. */
  rpe?: number | null;
  durationS: number | null;
  distanceM: number | null;
  done: boolean;
};

/** A set as the server returns it, for "previous". */
export type LoggedSet = {
  type: SetType;
  weightG: number | null;
  reps: number | null;
  rpe: number | null;
  durationS: number | null;
  distanceM: number | null;
};

export type LiveExercise = {
  key: number;
  exercise: ExerciseSummary;
  superset?: string;
  notes: string;
  /** Rest after each set in seconds, null for the default. */
  restS: number | null;
  sets: LiveSet[];
  /** The routine's targets, by set position. */
  targets?: RoutineSet[];
};

export type Live = {
  /** The ULID the finished workout is saved under, made at start. */
  id: string;
  /** Whose workout it is: a device can be shared. */
  userId: string;
  /** The routine it started from, as it was then. */
  routine?: Routine;
  startedAt: number;
  exercises: LiveExercise[];
  /** Each exercise's sets from last time, by exercise id. */
  previous: Record<string, LoggedSet[]>;
  /** Each exercise's bests before this workout, by exercise id; missing when never done or not loaded yet. */
  bests?: Record<string, Partial<Record<RecordKind, number>>>;
  rest?: { endsAt: number; totalS: number };
  nextKey: number;
};

const blankSet = (type: SetType = "normal"): LiveSet => ({
  type,
  weightG: null,
  reps: null,
  durationS: null,
  distanceM: null,
  done: false,
});

export function start(id: string, userId: string, now: number, routine?: Routine): Live {
  const live: Live = { id, userId, routine, startedAt: now, exercises: [], previous: {}, nextKey: 0 };
  if (!routine) return live;
  return {
    ...live,
    nextKey: routine.exercises.length,
    exercises: routine.exercises.map((e, i) => ({
      key: i,
      exercise: e.exercise!,
      superset: e.superset,
      notes: e.notes,
      restS: e.restS,
      targets: e.sets,
      // Targets fill the fields as hints; the lifter types or ticks.
      sets: e.sets.length ? e.sets.map((s) => blankSet(s.type)) : [blankSet()],
    })),
  };
}

export type Action =
  | { type: "previous"; previous: Record<string, LoggedSet[]>; bests: NonNullable<Live["bests"]> }
  | { type: "addExercises"; exercises: ExerciseSummary[] }
  | { type: "removeExercise"; i: number }
  | { type: "moveExercise"; i: number; by: -1 | 1 }
  | { type: "swapExercise"; i: number; exercise: ExerciseSummary }
  | { type: "superset"; i: number }
  | { type: "exerciseNotes"; i: number; notes: string }
  | { type: "addSet"; i: number }
  | { type: "removeSet"; i: number; j: number }
  | { type: "updateSet"; i: number; j: number; patch: Partial<LiveSet> }
  | { type: "toggleSet"; i: number; j: number; now: number }
  | { type: "restAdjust"; by: number; now: number }
  | { type: "restEnd" };

export function reduce(live: Live, a: Action): Live {
  const exs = live.exercises;
  const at = (i: number, f: (e: LiveExercise) => LiveExercise) => exs.map((e, k) => (k === i ? f(e) : e));
  switch (a.type) {
    case "previous":
      return { ...live, previous: { ...live.previous, ...a.previous }, bests: { ...live.bests, ...a.bests } };
    case "addExercises":
      return {
        ...live,
        nextKey: live.nextKey + a.exercises.length,
        exercises: [
          ...exs,
          ...a.exercises.map((exercise, n) => ({ key: live.nextKey + n, exercise, notes: "", restS: null, sets: [blankSet()] })),
        ],
      };
    case "removeExercise":
      return { ...live, exercises: relabel(exs.filter((_, k) => k !== a.i)) };
    case "moveExercise":
      return { ...live, exercises: relabel(move(exs, a.i, a.by)) };
    case "swapExercise":
      // A new exercise keeps the slot and the set types; its values and targets don't carry over.
      return {
        ...live,
        exercises: at(a.i, (e) => ({ ...e, exercise: a.exercise, targets: undefined, sets: e.sets.map((s) => blankSet(s.type)) })),
      };
    case "superset":
      return { ...live, exercises: exs[a.i].superset ? leaveSuperset(exs, a.i) : supersetWithNext(exs, a.i) };
    case "exerciseNotes":
      return { ...live, exercises: at(a.i, (e) => ({ ...e, notes: a.notes })) };
    case "addSet":
      return { ...live, exercises: at(a.i, (e) => ({ ...e, sets: [...e.sets, blankSet()] })) };
    case "removeSet":
      return { ...live, exercises: at(a.i, (e) => ({ ...e, sets: e.sets.filter((_, k) => k !== a.j) })) };
    case "updateSet":
      return { ...live, exercises: at(a.i, (e) => ({ ...e, sets: e.sets.map((s, k) => (k === a.j ? { ...s, ...a.patch } : s)) })) };
    case "toggleSet": {
      const e = exs[a.i];
      const s = e.sets[a.j];
      if (s.done) return reduce(live, { type: "updateSet", i: a.i, j: a.j, patch: { done: false } });
      const filled = fill(s, hint(live, a.i, a.j), e.exercise.kind);
      if (!filled) return live;
      const next = { ...live, exercises: at(a.i, (x) => ({ ...x, sets: x.sets.map((y, k) => (k === a.j ? filled : y)) })) };
      // In a superset, the rest comes after its last exercise.
      const restS = e.restS ?? defaultRestS;
      if (e.superset && exs[a.i + 1]?.superset === e.superset) return next;
      return restS > 0 ? { ...next, rest: { endsAt: a.now + restS * 1000, totalS: restS } } : { ...next, rest: undefined };
    }
    case "restAdjust": {
      if (!live.rest) return live;
      const endsAt = live.rest.endsAt + a.by * 1000;
      if (endsAt <= a.now) return { ...live, rest: undefined };
      return { ...live, rest: { endsAt, totalS: Math.max(live.rest.totalS + a.by, 1) } };
    }
    case "restEnd":
      return { ...live, rest: undefined };
  }
}

/** What the fields of set j show greyed: last time's set at that position, else the routine's target. */
export function hint(live: Live, i: number, j: number): Partial<LoggedSet> {
  const e = live.exercises[i];
  const prev = live.previous[e.exercise.id]?.[j];
  if (prev) return prev;
  const t = e.targets?.[j];
  if (!t) return {};
  return { weightG: t.weightG, reps: t.repsMax ?? t.repsMin, durationS: t.durationS, distanceM: t.distanceM };
}

/** s ticked done, its empty fields taken from the hint; null when a needed value is still missing. */
function fill(s: LiveSet, h: Partial<LoggedSet>, kind: ExerciseKind): LiveSet | null {
  const f = fieldsOf(kind);
  const pick = (on: boolean, v: number | null, hv: number | null | undefined) => (on ? (v ?? hv ?? null) : null);
  const out: LiveSet = {
    ...s,
    done: true,
    weightG: pick(f.weight, s.weightG, h.weightG),
    reps: pick(f.reps, s.reps, h.reps),
    durationS: pick(f.duration, s.durationS, h.durationS),
    distanceM: pick(f.distance, s.distanceM, h.distanceM),
  };
  const missing =
    (kind === "weight_reps" && out.weightG == null) ||
    (f.reps && out.reps == null) ||
    (f.duration && out.durationS == null) ||
    (f.distance && out.distanceM == null);
  return missing ? null : out;
}

/** Weight × reps of the done sets that aren't warm-ups, as the server counts it. */
export function volumeOf(live: Live): number {
  let g = 0;
  for (const e of live.exercises) {
    if (e.exercise.kind !== "weight_reps" && e.exercise.kind !== "weighted_bodyweight") continue;
    for (const s of e.sets) if (s.done && s.type !== "warmup") g += (s.weightG ?? 0) * (s.reps ?? 0);
  }
  return g;
}

/**
 * The records exercise i's done sets beat so far.
 * shortcut: an exercise done twice in a workout is judged per card, where the server sums both; fine until it shows.
 */
export const recordsOf = (live: Live, i: number) => {
  const e = live.exercises[i];
  return beaten(e.exercise.kind, e.sets, live.bests?.[e.exercise.id]);
};

export const doneSets = (live: Live) => live.exercises.reduce((n, e) => n + e.sets.filter((s) => s.done).length, 0);

/** The exercises and done sets to save; exercises with none are left out. */
function doneExercises(live: Live) {
  return relabel(
    live.exercises
      .map((e) => ({ ...e, sets: e.sets.filter((s) => s.done) }))
      .filter((e) => e.sets.length > 0),
  );
}

export type WorkoutBody = {
  title: string;
  notes: string;
  routineId?: string;
  startedAt: string;
  endedAt: string;
  exercises: {
    exerciseId: string;
    superset?: string;
    notes: string;
    sets: (Omit<LoggedSet, "rpe"> & { rpe?: number | null })[];
  }[];
};

/** The PUT /api/workouts/{id} body of the finished workout. */
export function toWorkout(live: Live, title: string, notes: string, endedAt: number): WorkoutBody {
  return {
    title,
    notes,
    routineId: live.routine?.id,
    startedAt: new Date(live.startedAt).toISOString(),
    endedAt: new Date(Math.max(endedAt, live.startedAt)).toISOString(),
    exercises: doneExercises(live).map((e) => ({
      exerciseId: e.exercise.id,
      superset: e.superset,
      notes: e.notes,
      sets: e.sets.map((s) => ({
        type: s.type,
        weightG: s.weightG,
        reps: s.reps,
        rpe: s.rpe ?? null,
        durationS: s.durationS,
        distanceM: s.distanceM,
      })),
    })),
  };
}

export type RoutineUpdate = "keep" | "weights" | "sets";

/**
 * The routine rewritten from the finished workout, or null to leave it:
 * "weights" copies each done working set's weight onto the routine set at the
 * same position of the same exercise; "sets" replaces its exercises and sets.
 */
export function updatedRoutine(live: Live, mode: RoutineUpdate): RoutineInput | null {
  const r = live.routine;
  if (!r || mode === "keep") return null;
  const base = { name: r.name, notes: r.notes, folderId: r.folderId };
  const strip = (e: Routine["exercises"][number]) => ({
    exerciseId: e.exerciseId,
    superset: e.superset,
    restS: e.restS,
    notes: e.notes,
    sets: e.sets,
  });
  if (mode === "sets") {
    return {
      ...base,
      exercises: doneExercises(live).map((e) => ({
        exerciseId: e.exercise.id,
        superset: e.superset,
        restS: e.restS,
        notes: e.notes,
        sets: e.sets.map((s) => ({
          type: s.type,
          weightG: s.weightG,
          repsMin: s.reps,
          repsMax: s.reps,
          durationS: s.durationS,
          distanceM: s.distanceM,
        })),
      })),
    };
  }
  // The n-th time an exercise appears in the routine matches its n-th time in the workout.
  const seen: Record<string, number> = {};
  return {
    ...base,
    exercises: r.exercises.map((re) => {
      const n = (seen[re.exerciseId] = (seen[re.exerciseId] ?? -1) + 1);
      const le = live.exercises.filter((e) => e.exercise.id === re.exerciseId)[n];
      return {
        ...strip(re),
        sets: re.sets.map((t, j) => {
          const s = le?.sets[j];
          return s?.done && s.weightG != null && t.type !== "warmup" && s.type !== "warmup" ? { ...t, weightG: s.weightG } : t;
        }),
      };
    }),
  };
}

import { describe, expect, it } from "vitest";

import type { ExerciseSummary, Routine } from "@/lib/api";

import { doneSets, editing, elapsedS, recordsOf, reduce, start, toWorkout, updatedRoutine, volumeOf, type Live } from "./model";
import { ulid } from "./ulid";

const ex = (id: string, kind: ExerciseSummary["kind"] = "weight_reps"): ExerciseSummary => ({
  id,
  name: id,
  kind,
  category: "strength",
  equipment: "barbell",
  primaryMuscles: ["chest"],
  secondaryMuscles: [],
  custom: false,
  archived: false,
});
const bench = ex("bench"), fly = ex("fly"), plank = ex("plank", "duration");

const routine: Routine = {
  id: "r1",
  name: "Push",
  notes: "",
  folderId: "f1",
  exercises: [
    {
      exerciseId: "bench",
      exercise: bench,
      restS: 90,
      notes: "",
      sets: [
        { type: "warmup", weightG: 40000, repsMin: 10, repsMax: 10 },
        { type: "normal", weightG: 80000, repsMin: 5, repsMax: 8 },
      ],
    },
    { exerciseId: "fly", exercise: fly, superset: "A", restS: null, notes: "", sets: [{ type: "normal", weightG: 20000, repsMin: 12, repsMax: 12 }] },
    { exerciseId: "plank", exercise: plank, superset: "A", restS: 0, notes: "", sets: [{ type: "normal", durationS: 60 }] },
  ],
};

const t0 = 1_700_000_000_000;
const run = (live: Live, ...actions: Parameters<typeof reduce>[1][]) => actions.reduce(reduce, live);

describe("live workout", () => {
  it("starts from a routine with its targets as hints", () => {
    const l = start("id", "u", t0, routine);
    expect(l.exercises.map((e) => e.sets.map((s) => s.type))).toEqual([["warmup", "normal"], ["normal"], ["normal"]]);
    expect(l.exercises[0].sets[0].weightG).toBeNull();
  });

  it("ticks a set with the hint's values and starts the rest", () => {
    let l = start("id", "u", t0, routine);
    l = run(l, { type: "toggleSet", i: 0, j: 0, now: t0 + 1000 });
    expect(l.exercises[0].sets[0]).toMatchObject({ done: true, weightG: 40000, reps: 10 });
    expect(l.rest).toEqual({ endsAt: t0 + 1000 + 90_000, totalS: 90 });
    // Last time's set wins over the target.
    l = run(l, { type: "previous", previous: { bench: [{ type: "warmup", weightG: 40000, reps: 10, rpe: null, durationS: null, distanceM: null }, { type: "normal", weightG: 85000, reps: 6, rpe: 80, durationS: null, distanceM: null }] }, bests: {} });
    l = run(l, { type: "updateSet", i: 0, j: 1, patch: { reps: 7 } }, { type: "toggleSet", i: 0, j: 1, now: t0 });
    expect(l.exercises[0].sets[1]).toMatchObject({ done: true, weightG: 85000, reps: 7 });
    // Unticking keeps the values.
    l = run(l, { type: "toggleSet", i: 0, j: 1, now: t0 });
    expect(l.exercises[0].sets[1]).toMatchObject({ done: false, weightG: 85000 });
  });

  it("badges the done sets that beat a best, never a first time", () => {
    let l = run(start("id", "u", t0), { type: "addExercises", exercises: [bench, plank] });
    l = run(
      l,
      { type: "previous", previous: {}, bests: { bench: { e1rm: 116667, weight: 100000, set_volume: 500000, session_volume: 500000, reps: 5 } } },
      { type: "updateSet", i: 0, j: 0, patch: { weightG: 100000, reps: 6 } },
      { type: "toggleSet", i: 0, j: 0, now: t0 },
      { type: "addSet", i: 0 },
      { type: "updateSet", i: 0, j: 1, patch: { weightG: 110000, reps: 3 } },
      { type: "updateSet", i: 1, j: 0, patch: { durationS: 60 } },
      { type: "toggleSet", i: 1, j: 0, now: t0 },
    );
    // Set 2 isn't ticked yet: only set 1 counts.
    expect(recordsOf(l, 0)).toEqual([
      { kind: "e1rm", value: 120000, previous: 116667, set: 0 },
      { kind: "set_volume", value: 600000, previous: 500000, set: 0 },
      { kind: "session_volume", value: 600000, previous: 500000, set: -1 },
      { kind: "reps", value: 6, previous: 5, set: 0 },
    ]);
    l = run(l, { type: "toggleSet", i: 0, j: 1, now: t0 });
    expect(recordsOf(l, 0).map((r) => `${r.kind}@${r.set}`)).toEqual(["e1rm@1", "weight@1", "set_volume@0", "session_volume@-1", "reps@0"]);
    expect(recordsOf(l, 1)).toEqual([]); // the plank was never done
  });

  it("edits a saved workout: same times, title and routine, no rest, no badges", () => {
    const set = { type: "normal" as const, weightG: 100000, reps: 5, rpe: null, durationS: null, distanceM: null };
    let l = editing(
      {
        id: "w1",
        title: "Push",
        notes: "felt good",
        routineId: "r1",
        startedAt: "2026-11-20T18:00:00Z",
        endedAt: "2026-11-20T19:00:00Z",
        volumeG: 500000,
        exercises: [{ exerciseId: "bench", notes: "", exercise: bench, sets: [set] }],
        records: [],
      },
      "u",
    );
    l = run(l, { type: "previous", previous: {}, bests: { bench: { weight: 90000 } } }, { type: "addSet", i: 0 });
    l = run(l, { type: "updateSet", i: 0, j: 1, patch: { weightG: 105000, reps: 3 } }, { type: "toggleSet", i: 0, j: 1, now: t0 });
    expect(l.rest).toBeUndefined();
    expect(recordsOf(l, 0)).toEqual([]);
    expect(elapsedS(l, t0)).toBe(3600);
    const w = toWorkout(l, l.edit!.title, l.edit!.notes, l.edit!.endedAt);
    expect(w).toMatchObject({ title: "Push", notes: "felt good", routineId: "r1", startedAt: "2026-11-20T18:00:00.000Z", endedAt: "2026-11-20T19:00:00.000Z" });
    expect(w.exercises[0].sets.map((s) => s.weightG)).toEqual([100000, 105000]);
  });

  it("won't tick a set missing a value with no hint", () => {
    const l = run(start("id", "u", t0), { type: "addExercises", exercises: [bench] }, { type: "toggleSet", i: 0, j: 0, now: t0 });
    expect(l.exercises[0].sets[0].done).toBe(false);
  });

  it("rests after the last exercise of a superset only, and not when rest is off", () => {
    let l = start("id", "u", t0, routine);
    l = run(l, { type: "toggleSet", i: 1, j: 0, now: t0 });
    expect(l.rest).toBeUndefined();
    l = run(l, { type: "toggleSet", i: 2, j: 0, now: t0 });
    expect(l.rest).toBeUndefined(); // plank's rest is 0
  });

  it("adjusts and ends the rest", () => {
    let l = run(start("id", "u", t0, routine), { type: "toggleSet", i: 0, j: 0, now: t0 });
    l = run(l, { type: "restAdjust", by: 15, now: t0 });
    expect(l.rest).toEqual({ endsAt: t0 + 105_000, totalS: 105 });
    l = run(l, { type: "restAdjust", by: -15, now: t0 + 100_000 });
    expect(l.rest).toBeUndefined();
  });

  it("edits exercises: add, swap, move, superset, remove", () => {
    let l = run(start("id", "u", t0), { type: "addExercises", exercises: [bench, fly, plank] });
    expect(l.exercises.map((e) => e.key)).toEqual([0, 1, 2]);
    l = run(l, { type: "updateSet", i: 0, j: 0, patch: { weightG: 1, type: "drop" } }, { type: "swapExercise", i: 0, exercise: fly });
    expect(l.exercises[0].exercise.id).toBe("fly");
    expect(l.exercises[0].sets[0]).toMatchObject({ type: "drop", weightG: null });
    l = run(l, { type: "superset", i: 1 });
    expect(l.exercises.map((e) => e.superset ?? "-").join("")).toBe("-AA");
    l = run(l, { type: "moveExercise", i: 2, by: -1 }, { type: "removeExercise", i: 0 });
    expect(l.exercises.map((e) => e.exercise.id)).toEqual(["plank", "fly"]);
  });

  it("builds the workout from the done sets and counts volume", () => {
    let l = start("id", "u", t0, routine);
    l = run(
      l,
      { type: "toggleSet", i: 0, j: 0, now: t0 },
      { type: "updateSet", i: 0, j: 1, patch: { weightG: 102058, reps: 5, rpe: 8.5 } },
      { type: "toggleSet", i: 0, j: 1, now: t0 },
      { type: "toggleSet", i: 2, j: 0, now: t0 },
    );
    expect(volumeOf(l)).toBe(102058 * 5);
    expect(doneSets(l)).toBe(3);
    const w = toWorkout(l, "Push", "", t0 + 3_600_000);
    expect(w.routineId).toBe("r1");
    expect(w.endedAt).toBe(new Date(t0 + 3_600_000).toISOString());
    // fly has no done set: left out, and plank's superset of one is no superset.
    expect(w.exercises.map((e) => [e.exerciseId, e.superset, e.sets.length])).toEqual([["bench", undefined, 2], ["plank", undefined, 1]]);
    expect(w.exercises[0].sets[1]).toEqual({ type: "normal", weightG: 102058, reps: 5, rpe: 8.5, durationS: null, distanceM: null });
  });

  it("updates the routine's weights or sets", () => {
    let l = start("id", "u", t0, routine);
    l = run(
      l,
      { type: "updateSet", i: 0, j: 0, patch: { weightG: 50000 } },
      { type: "toggleSet", i: 0, j: 0, now: t0 },
      { type: "updateSet", i: 0, j: 1, patch: { weightG: 90000 } },
      { type: "toggleSet", i: 0, j: 1, now: t0 },
      { type: "addSet", i: 0 },
    );
    expect(updatedRoutine(l, "keep")).toBeNull();
    const w = updatedRoutine(l, "weights")!;
    expect(w.folderId).toBe("f1");
    // The warm-up keeps its target; the working set takes today's weight; structure unchanged.
    expect(w.exercises[0].sets).toEqual([routine.exercises[0].sets[0], { ...routine.exercises[0].sets[1], weightG: 90000 }]);
    expect(w.exercises).toHaveLength(3);
    const s = updatedRoutine(l, "sets")!;
    expect(s.exercises).toHaveLength(1);
    expect(s.exercises[0]).toMatchObject({ exerciseId: "bench", restS: 90 });
    expect(s.exercises[0].sets[1]).toEqual({ type: "normal", weightG: 90000, repsMin: 8, repsMax: 8, durationS: null, distanceM: null });
  });
});

describe("ulid", () => {
  it("makes a sortable ULID the server accepts", () => {
    const a = ulid(t0), b = ulid(t0 + 1);
    expect(a).toMatch(/^[0-9A-HJKMNP-TV-Z]{26}$/);
    expect(a.slice(0, 10) < b.slice(0, 10)).toBe(true);
    expect(ulid(t0)).not.toBe(a);
  });
});

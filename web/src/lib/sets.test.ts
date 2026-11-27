import { describe, expect, it } from "vitest";

import { fieldsOf, setLabels } from "./sets";

describe("sets", () => {
  it("numbers normal sets and letters the others", () => {
    const letters = { normal: "", warmup: "W", drop: "D", failure: "F" };
    expect(setLabels(["warmup", "normal", "normal", "drop", "failure", "normal"], letters)).toEqual(["W", "1", "2", "D", "F", "3"]);
  });
  it("gives each kind its fields", () => {
    expect(fieldsOf("reps_only")).toEqual({ weight: false, reps: true, distance: false, duration: false });
    expect(fieldsOf("distance_duration")).toEqual({ weight: false, reps: false, distance: true, duration: true });
  });
});

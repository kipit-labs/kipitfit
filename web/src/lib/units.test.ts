import i18n from "i18next";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

import { formatDuration, formatKm, formatNumber, formatReps, formatWeight, parseDuration, parseKm, parseReps, parseWeight } from "./units";

// Formatting follows i18next's language, English until switched.
beforeAll(() => i18n.init({ lng: "en", resources: {} }));
afterEach(() => i18n.changeLanguage("en"));

describe("weights", () => {
  it("stores grams from kg and lb", () => {
    expect(parseWeight("92.5", "kg")).toBe(92500);
    expect(parseWeight("92,5", "kg")).toBe(92500);
    expect(parseWeight("225", "lb")).toBe(102058);
    expect(parseWeight("  ", "kg")).toBeNull();
    expect(parseWeight("-5", "kg")).toBeUndefined();
    expect(parseWeight("abc", "lb")).toBeUndefined();
  });
  it("shows kg to 0.01 and lb to 0.5", () => {
    expect(formatWeight(92500, "kg")).toBe("92.5");
    expect(formatWeight(1250, "kg")).toBe("1.25");
    expect(formatWeight(102058, "lb")).toBe("225");
    expect(formatWeight(20000, "lb")).toBe("44");
    expect(formatWeight(21000, "lb")).toBe("46.5");
  });
  it("writes numbers the UI language's way; fields skip grouping so they parse back", async () => {
    expect(formatWeight(6505500, "kg")).toBe("6,505.5");
    expect(formatWeight(1000000, "lb", true)).toBe("2204.5");
    await i18n.changeLanguage("fr");
    // Intl groups French thousands with a narrow no-break space.
    expect(formatWeight(6505500, "kg")).toBe("6\u202f505,5");
    expect(formatWeight(92500, "kg", true)).toBe("92,5");
    expect(parseWeight(formatWeight(92500, "kg", true), "kg")).toBe(92500);
    expect(formatKm(5200, true)).toBe("5,2");
    expect(formatNumber(0.25)).toBe("0,25");
  });
  it("round-trips lb without drift", () => {
    for (const lb of ["45", "135", "225", "2.5", "47.5"]) expect(formatWeight(parseWeight(lb, "lb")!, "lb")).toBe(lb);
  });
});


describe("reps", () => {
  it("parses one count or a range", () => {
    expect(parseReps("8")).toEqual({ min: 8, max: 8 });
    expect(parseReps("8-12")).toEqual({ min: 8, max: 12 });
    expect(parseReps("8 – 12")).toEqual({ min: 8, max: 12 });
    expect(parseReps("")).toEqual({ min: null, max: null });
    expect(parseReps("12-8")).toBeUndefined();
    expect(parseReps("x")).toBeUndefined();
  });
  it("formats", () => {
    expect(formatReps(8, 8)).toBe("8");
    expect(formatReps(8, 12)).toBe("8–12");
    expect(formatReps(null, null)).toBe("");
  });
});

describe("durations and distances", () => {
  it("parses m:ss, h:mm:ss and plain seconds", () => {
    expect(parseDuration("1:30")).toBe(90);
    expect(parseDuration("90")).toBe(90);
    expect(parseDuration("1:02:03")).toBe(3723);
    expect(parseDuration("")).toBeNull();
    expect(parseDuration("1:x")).toBeUndefined();
  });
  it("formats", () => {
    expect(formatDuration(90)).toBe("1:30");
    expect(formatDuration(3723)).toBe("1:02:03");
    expect(formatDuration(5)).toBe("0:05");
  });
  it("keeps kilometres in metres", () => {
    expect(parseKm("5,2")).toBe(5200);
    expect(formatKm(5200)).toBe("5.2");
  });
});

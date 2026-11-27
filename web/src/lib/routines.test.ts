import { describe, expect, it } from "vitest";

import { leaveSuperset, move, relabel, supersetWithNext } from "./routines";

const ex = (...letters: (string | undefined)[]) => letters.map((superset, i) => ({ id: i, superset }));
const letters = (l: { superset?: string }[]) => l.map((e) => e.superset ?? "-").join("");

describe("move", () => {
  it("swaps with a neighbour and stops at the ends", () => {
    expect(move([1, 2, 3], 0, 1)).toEqual([2, 1, 3]);
    expect(move([1, 2, 3], 2, -1)).toEqual([1, 3, 2]);
    expect(move([1, 2, 3], 0, -1)).toEqual([1, 2, 3]);
    expect(move([1, 2, 3], 2, 1)).toEqual([1, 2, 3]);
  });
});

describe("supersets", () => {
  it("relabels runs A, B… and drops lone letters", () => {
    expect(letters(relabel(ex("C", "C", undefined, "Z", "Z", "Z", "Q")))).toBe("AA-BBB-");
  });
  it("groups with the next and extends a group", () => {
    let l = supersetWithNext(ex(undefined, undefined, undefined), 0);
    expect(letters(l)).toBe("AA-");
    l = supersetWithNext(l, 1);
    expect(letters(l)).toBe("AAA");
  });
  it("leaves a group, splitting it when needed", () => {
    expect(letters(leaveSuperset(ex("A", "A"), 0))).toBe("--");
    expect(letters(leaveSuperset(ex("A", "A", "A"), 2))).toBe("AA-");
    expect(letters(leaveSuperset(ex("A", "A", "A", "A", "A"), 2))).toBe("AA-BB");
  });
  it("drops a superset a move splits", () => {
    expect(letters(relabel(move(ex("A", "A", undefined), 1, 1)))).toBe("---");
  });
});

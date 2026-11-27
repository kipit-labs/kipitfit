import { api } from "./api";

export const routinesQuery = { queryKey: ["routines"], queryFn: api.routines } as const;

/** Rest after each set when the routine sets none. */
export const defaultRestS = 120;

/** list with item i moved by one place up (-1) or down (+1); unchanged at an end. */
export function move<T>(list: T[], i: number, by: -1 | 1): T[] {
  const j = i + by;
  if (j < 0 || j >= list.length) return list;
  const next = [...list];
  [next[i], next[j]] = [next[j], next[i]];
  return next;
}

type Grouped = { superset?: string };

/**
 * Relabels supersets A, B, C… in list order. A superset is a run of neighbours
 * sharing a letter; a run of one is no superset.
 */
export function relabel<T extends Grouped>(list: T[]): T[] {
  let next = 0;
  return list.map((e, i) => {
    const s = e.superset;
    const withPrev = !!s && list[i - 1]?.superset === s;
    const withNext = !!s && list[i + 1]?.superset === s;
    if (!withPrev && !withNext) return { ...e, superset: undefined };
    if (!withPrev) next++;
    return { ...e, superset: String.fromCharCode(64 + next) };
  });
}

/** Joins item i with the next one into a superset (the next joins i's group). */
export function supersetWithNext<T extends Grouped>(list: T[], i: number): T[] {
  if (i + 1 >= list.length) return list;
  // A letter no one uses until relabel names it.
  const letter = list[i].superset ?? "~";
  return relabel(list.map((e, j) => (j === i || j === i + 1 ? { ...e, superset: letter } : e)));
}

/** Takes item i out of its superset. */
export function leaveSuperset<T extends Grouped>(list: T[], i: number): T[] {
  const s = list[i].superset;
  if (!s) return list;
  // Splitting a run in the middle leaves two groups: give the part after i its own letter.
  return relabel(
    list.map((e, j) => (j === i ? { ...e, superset: undefined } : j > i && e.superset === s ? { ...e, superset: `${s}'` } : e)),
  );
}

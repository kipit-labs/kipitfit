// The live workout and the finishes waiting to be sent, both kept in localStorage so a reload, a dead battery or
// a gym without signal loses nothing.
import { useSyncExternalStore } from "react";

import { api, ApiError, type RoutineInput } from "@/lib/api";

import { reduce, type Action, type Live, type WorkoutBody } from "./model";

const liveKey = "kipitfit.live";
const outboxKey = "kipitfit.outbox";

function read<T>(key: string): T | null {
  try {
    const raw = localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as T) : null;
  } catch {
    return null;
  }
}

function write(key: string, v: unknown) {
  try {
    if (v == null) localStorage.removeItem(key);
    else localStorage.setItem(key, JSON.stringify(v));
  } catch {
    // Storage full or blocked: the state still lives in memory until the page closes.
  }
}

const listeners = new Set<() => void>();
const subscribe = (f: () => void) => {
  listeners.add(f);
  return () => void listeners.delete(f);
};
const emit = () => listeners.forEach((f) => f());

let live = read<Live>(liveKey);

/** The device's live workout, if it is userId's. */
export function useLive(userId: string): Live | null {
  const l = useSyncExternalStore(subscribe, () => live);
  return l?.userId === userId ? l : null;
}

export function getLive(): Live | null {
  return live;
}

export function setLive(next: Live | null) {
  live = next;
  write(liveKey, next);
  emit();
}

export function dispatch(a: Action) {
  if (live) setLive(reduce(live, a));
}

/** Fetches last time's sets of exercises into the live workout; offline, the column stays empty. */
export async function loadPrevious(ids: string[]) {
  if (!ids.length) return;
  try {
    dispatch({ type: "previous", previous: await api.previous(ids) });
  } catch {
    // No signal: "previous" is a nicety.
  }
}

// The outbox: finished workouts not yet saved, sent in order and retried until the server takes them.

export type Finish = {
  id: string;
  userId: string;
  workout: WorkoutBody;
  /** The routine rewritten on finish, saved once the workout is. */
  routine?: { id: string; body: RoutineInput };
  /** Why the server refused it, when retrying won't help. */
  error?: unknown;
};

export type SendState = "idle" | "saving" | "retry";

let outbox = read<Finish[]>(outboxKey) ?? [];
let state: SendState = "idle";
let timer: ReturnType<typeof setTimeout> | undefined;
let delay = 0;
let onSaved: (f: Finish) => void = () => {};

const setOutbox = (next: Finish[]) => {
  outbox = next;
  write(outboxKey, next);
  emit();
};
const setState = (s: SendState) => {
  state = s;
  emit();
};

export function useOutbox(userId: string): { pending: Finish[]; state: SendState } {
  const box = useSyncExternalStore(subscribe, () => outbox);
  const s = useSyncExternalStore(subscribe, () => state);
  return { pending: box.filter((f) => f.userId === userId), state: s };
}

/** Queues a finished workout and starts sending. */
export function finish(f: Finish, userId: string) {
  setOutbox([...outbox.filter((x) => x.id !== f.id), f]);
  setLive(null);
  delay = 0;
  void flush(userId);
}

export function discardFinish(id: string) {
  setOutbox(outbox.filter((f) => f.id !== id));
}

/** Calls f each time a finish is saved, e.g. to refresh the routines. */
export function onFinishSaved(f: (x: Finish) => void) {
  onSaved = f;
}

const transient = (err: unknown) =>
  !(err instanceof ApiError) || err.status >= 500 || err.status === 408 || err.status === 429;

/**
 * Sends userId's queued finishes. A network error or a 5xx retries with backoff
 * (2 s doubling to a minute); a signed-out 401 waits for the next call; any other
 * refusal stays queued with its error until the user retries or discards it.
 */
export async function flush(userId: string) {
  clearTimeout(timer);
  if (state === "saving") return;
  for (const f of outbox.filter((x) => x.userId === userId && !x.error)) {
    setState("saving");
    try {
      // PUT is idempotent: a finish that landed before the answer got lost saves once.
      await api.putWorkout(f.id, f.workout);
      if (f.routine) {
        await api.updateRoutine(f.routine.id, f.routine.body).catch((err) => {
          // The routine was deleted meanwhile: the workout is what matters.
          if (!(err instanceof ApiError && err.status === 404)) throw err;
        });
      }
      setOutbox(outbox.filter((x) => x.id !== f.id));
      onSaved(f);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) return setState("retry");
      if (transient(err)) {
        delay = Math.min(delay ? delay * 2 : 2000, 60_000);
        timer = setTimeout(() => void flush(userId), delay);
        return setState("retry");
      }
      setOutbox(outbox.map((x) => (x.id === f.id ? { ...x, error: err instanceof ApiError ? { code: err.code, message: err.message, params: err.params } : String(err) } : x)));
    }
  }
  delay = 0;
  setState("idle");
}

/** Clears a refused finish's error and sends it again. */
export function retryFinish(id: string, userId: string) {
  setOutbox(outbox.map((x) => (x.id === id ? { ...x, error: undefined } : x)));
  delay = 0;
  void flush(userId);
}

export class ApiError extends Error {
  readonly status: number;
  /** Names the error for the UI, which shows errors.<code> in its language. */
  readonly code?: string;
  /** Values the message for code is built from, e.g. a length limit. */
  readonly params?: Record<string, string | number>;

  constructor(status: number, message: string, code?: string, params?: Record<string, string | number>) {
    super(message);
    this.status = status;
    this.code = code;
    this.params = params;
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body) headers.set("Content-Type", "application/json");
  const res = await fetch(`/api${path}`, { ...init, headers, credentials: "same-origin" });
  if (!res.ok) {
    let body: { error?: string; code?: string; params?: Record<string, string | number> } = {};
    try {
      body = await res.json();
    } catch {
      // Non-JSON error body.
    }
    throw new ApiError(res.status, body.error ?? res.statusText, body.code, body.params);
  }
  return res.status === 204 ? (undefined as T) : ((await res.json()) as T);
}

const send = <T>(method: string, path: string, body?: unknown) =>
  request<T>(path, { method, body: body === undefined ? undefined : JSON.stringify(body) });

export type Unit = "kg" | "lb";

export type User = {
  id: string;
  email: string;
  name: string;
  isAdmin: boolean;
  /** "" until the user picks one. */
  language: "" | "en" | "fr";
  unit: Unit;
  createdAt: string;
};

export type AuthState = { setupRequired: boolean; user?: User };

export type Invite = { id: string; email: string; createdBy: string; createdAt: string; expiresAt: string };

export type ExerciseKind =
  | "weight_reps"
  | "reps_only"
  | "weighted_bodyweight"
  | "assisted_bodyweight"
  | "duration"
  | "distance_duration";

/** An exercise as the list returns it. */
export type ExerciseSummary = {
  id: string;
  name: string;
  kind: ExerciseKind;
  category: string;
  equipment: string;
  primaryMuscles: string[];
  secondaryMuscles: string[];
  /** The user's own exercise, which they can change; the rest is the shared library. */
  custom: boolean;
};

export type Exercise = ExerciseSummary & { level?: string; force?: string; mechanic?: string; instructions?: string[] };

export type ExerciseInput = Partial<
  Pick<Exercise, "name" | "kind" | "category" | "equipment" | "primaryMuscles" | "secondaryMuscles">
>;

export const api = {
  authState: () => request<AuthState>("/auth/state"),
  setup: (b: { setupToken: string; email: string; name: string; password: string }) => send<User>("POST", "/auth/setup", b),
  login: (b: { email: string; password: string }) => send<User>("POST", "/auth/login", b),
  logout: () => send<void>("POST", "/auth/logout"),
  acceptInvite: (b: { token: string; name: string; password: string }) => send<User>("POST", "/invites/accept", b),
  updateMe: (b: Partial<Pick<User, "name" | "language" | "unit">>) => send<User>("PATCH", "/me", b),
  invites: () => request<Invite[]>("/admin/invites"),
  createInvite: (email: string) => send<Invite & { link: string }>("POST", "/admin/invites", { email }),
  deleteInvite: (id: string) => send<void>("DELETE", `/admin/invites/${id}`),
  exercises: () => request<ExerciseSummary[]>("/exercises"),
  exercise: (id: string) => request<Exercise>(`/exercises/${id}`),
  createExercise: (b: ExerciseInput) => send<Exercise>("POST", "/exercises", b),
  updateExercise: (id: string, b: ExerciseInput) => send<Exercise>("PATCH", `/exercises/${id}`, b),
  deleteExercise: (id: string) => send<void>("DELETE", `/exercises/${id}`),
};

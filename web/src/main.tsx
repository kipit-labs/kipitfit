import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "next-themes";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, RouterProvider } from "react-router";
import { toast } from "sonner";
import { Toaster } from "@kipit-labs/kipitidy/sonner";
import { TooltipProvider } from "@kipit-labs/kipitidy/tooltip";

import "./index.css";
import i18n from "./i18n";
import { App } from "./App";
import { ApiError } from "./lib/api";
import { errorMessage } from "./lib/errors";
import { ExerciseDetail } from "./features/exercises/ExerciseDetail";
import { Exercises } from "./features/exercises/Exercises";
import { History, WorkoutDetail } from "./features/history/History";
import { Me } from "./features/me/Me";
import { Live } from "./features/live/Live";
import { Feed } from "./features/feed/Placeholders";
import { Progress } from "./features/progress/Progress";
import { RoutineEditor } from "./features/routines/RoutineEditor";
import { Workout } from "./features/workout/Workout";

// A 401 anywhere means the session is gone: re-check auth to show the login.
const onError = (err: Error) => {
  if (err instanceof ApiError && err.status === 401) void queryClient.invalidateQueries({ queryKey: ["auth"] });
};
declare module "@tanstack/react-query" {
  interface Register {
    // error: the toast title when the mutation fails. false when the caller shows the error itself.
    mutationMeta: { error?: string | false };
  }
}

const queryClient: QueryClient = new QueryClient({
  queryCache: new QueryCache({ onError }),
  mutationCache: new MutationCache({
    onError: (err, _vars, _ctx, mutation) => {
      onError(err);
      const title = mutation.meta?.error;
      if (title !== false) toast.error(title ?? i18n.t("errors.generic"), { description: errorMessage(err) });
    },
  }),
  defaultOptions: {
    queries: { retry: (n, err) => !(err instanceof ApiError && err.status < 500) && n < 2 },
  },
});

const router = createBrowserRouter([
  {
    path: "/",
    element: <App />,
    children: [
      { index: true, element: <Navigate to="/workout" replace /> },
      { path: "feed", element: <Feed /> },
      { path: "workout", element: <Workout /> },
      { path: "live", element: <Live /> },
      { path: "routines/new", element: <RoutineEditor /> },
      { path: "routines/:id", element: <RoutineEditor /> },
      { path: "exercises", element: <Exercises /> },
      { path: "exercises/:id", element: <ExerciseDetail /> },
      { path: "history", element: <History /> },
      { path: "workouts/:id", element: <WorkoutDetail /> },
      { path: "progress", element: <Progress /> },
      { path: "me", element: <Me /> },
      // App renders the invite page itself.
      { path: "invite", element: null },
      { path: "*", element: <Navigate to="/" replace /> },
    ],
  },
]);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>
          <RouterProvider router={router} />
          <Toaster />
        </TooltipProvider>
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>,
);

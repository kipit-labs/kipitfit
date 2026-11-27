import { useQueryClient } from "@tanstack/react-query";
import { Dumbbell, ChartLine, ChevronRight, Rss, User } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Link, NavLink, Outlet, useLocation } from "react-router";
import { toast } from "sonner";
import { cn } from "cn";

import { useMe } from "@/lib/auth";
import { routinesQuery } from "@/lib/routines";
import { formatDuration } from "@/lib/units";
import type { Live } from "@/live/model";
import { flush, onFinishSaved, useLive } from "@/live/store";
import { useNow } from "@/live/useNow";

import { RestBar } from "./RestBar";

const tabs = [
  { to: "/feed", icon: Rss, label: "nav.feed" },
  { to: "/workout", icon: Dumbbell, label: "nav.workout" },
  { to: "/progress", icon: ChartLine, label: "nav.progress" },
  { to: "/me", icon: User, label: "nav.me" },
] as const;

/** The signed-in app: one screen at a time over a bottom tab bar. */
export function Layout() {
  const { t } = useTranslation();
  // The exercise library and the routine editor are reached from the Workout tab.
  const path = useLocation().pathname;
  const inLibrary = path.startsWith("/exercises") || path.startsWith("/routines") || path === "/live";
  const me = useMe();
  const live = useLive(me.id);
  const qc = useQueryClient();
  // Finishes saved offline go out when the app opens and when the signal comes back.
  useEffect(() => {
    onFinishSaved(() => {
      toast.success(t("live.saved"));
      void qc.invalidateQueries({ queryKey: routinesQuery.queryKey });
    });
    const send = () => void flush(me.id);
    send();
    addEventListener("online", send);
    return () => removeEventListener("online", send);
  }, [me.id, qc, t]);
  const resume = live && path !== "/live";
  return (
    <div className="flex min-h-svh flex-col">
      <main
        className={cn(
          "mx-auto w-full max-w-lg flex-1 px-4 pt-[max(1.5rem,env(safe-area-inset-top))]",
          live?.rest || resume ? "pb-44" : "pb-24",
        )}
      >
        <Outlet />
      </main>
      <div className="fixed inset-x-0 bottom-0">
        {resume && <InProgress live={live} />}
        {live?.rest && <RestBar rest={live.rest} />}
        <nav className="border-t bg-background/95 pb-[env(safe-area-inset-bottom)] backdrop-blur">
          <ul className="mx-auto flex max-w-lg">
            {tabs.map(({ to, icon: Icon, label }) => (
              <li key={to} className="flex-1">
                <NavLink
                  to={to}
                  className={({ isActive }) =>
                    cn(
                      "flex h-16 flex-col items-center justify-center gap-1 text-xs text-muted-foreground transition-colors",
                      (isActive || (inLibrary && to === "/workout")) && "font-medium text-primary",
                    )
                  }
                >
                  <Icon className="size-5" aria-hidden />
                  {t(label)}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
      </div>
    </div>
  );
}

/** Shown over the tabs while a workout runs and its screen isn't open. */
function InProgress({ live }: { live: Live }) {
  const { t } = useTranslation();
  const now = useNow();
  return (
    <Link to="/live" className="block border-t bg-primary px-4 py-2 text-primary-foreground">
      <span className="mx-auto flex max-w-lg items-center gap-2 text-sm">
        <span className="min-w-0 flex-1 truncate font-medium">{live.routine?.name ?? t("live.defaultTitle")}</span>
        <span className="tabular-nums">{formatDuration(Math.max(0, Math.floor((now - live.startedAt) / 1000)))}</span>
        <span className="font-medium">{t("live.resume")}</span>
        <ChevronRight className="size-4" aria-hidden />
      </span>
    </Link>
  );
}

export function PageTitle({ children, className }: { children: React.ReactNode; className?: string }) {
  return <h1 className={cn("mb-6 font-heading text-2xl font-semibold", className)}>{children}</h1>;
}

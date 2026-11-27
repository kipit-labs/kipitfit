import { Dumbbell, ChartLine, Rss, User } from "lucide-react";
import { useTranslation } from "react-i18next";
import { NavLink, Outlet, useLocation } from "react-router";
import { cn } from "cn";

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
  const inLibrary = path.startsWith("/exercises") || path.startsWith("/routines");
  return (
    <div className="flex min-h-svh flex-col">
      <main className="mx-auto w-full max-w-lg flex-1 px-4 pt-[max(1.5rem,env(safe-area-inset-top))] pb-24">
        <Outlet />
      </main>
      <nav className="fixed inset-x-0 bottom-0 border-t bg-background/95 pb-[env(safe-area-inset-bottom)] backdrop-blur">
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
  );
}

export function PageTitle({ children, className }: { children: React.ReactNode; className?: string }) {
  return <h1 className={cn("mb-6 font-heading text-2xl font-semibold", className)}>{children}</h1>;
}

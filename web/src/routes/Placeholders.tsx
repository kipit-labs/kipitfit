import { Dumbbell, ChartLine, ChevronRight, Library, Rss } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { EmptyState } from "@kipit-labs/kipitidy/empty-state";
import { IconTile } from "@kipit-labs/kipitidy/icon-tile";

import { PageTitle } from "./Layout";

// Screens still to build (see docs/design.md).

export function Feed() {
  const { t } = useTranslation();
  return (
    <>
      <PageTitle>{t("nav.feed")}</PageTitle>
      <EmptyState icon={Rss} title={t("feed.emptyTitle")} description={t("feed.emptyDescription")} />
    </>
  );
}

export function Workout() {
  const { t } = useTranslation();
  return (
    <>
      <PageTitle>{t("nav.workout")}</PageTitle>
      <Link to="/exercises" className="mb-6 flex items-center gap-3 rounded-xl border bg-card p-4 transition-colors hover:bg-muted">
        <IconTile icon={Library} />
        <div className="min-w-0 flex-1">
          <p className="font-medium">{t("workout.library")}</p>
          <p className="text-sm text-muted-foreground">{t("workout.libraryDescription")}</p>
        </div>
        <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      </Link>
      <EmptyState icon={Dumbbell} title={t("workout.emptyTitle")} description={t("workout.emptyDescription")} />
    </>
  );
}

export function Progress() {
  const { t } = useTranslation();
  return (
    <>
      <PageTitle>{t("nav.progress")}</PageTitle>
      <EmptyState icon={ChartLine} title={t("progress.emptyTitle")} description={t("progress.emptyDescription")} />
    </>
  );
}

import { ChartLine, Rss } from "lucide-react";
import { useTranslation } from "react-i18next";
import { EmptyState } from "@kipit-labs/kipitidy/empty-state";

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

export function Progress() {
  const { t } = useTranslation();
  return (
    <>
      <PageTitle>{t("nav.progress")}</PageTitle>
      <EmptyState icon={ChartLine} title={t("progress.emptyTitle")} description={t("progress.emptyDescription")} />
    </>
  );
}

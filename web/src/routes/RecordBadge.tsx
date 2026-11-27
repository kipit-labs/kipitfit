import { Trophy } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { RecordKind, Unit } from "@/lib/api";
import { formatRecord } from "@/lib/records";

/** "PR · e1RM 111 kg": a record a set or a workout beat. */
export function RecordBadge({ kind, value, unit }: { kind: RecordKind; value?: number; unit: Unit }) {
  const { t } = useTranslation();
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-record px-2 py-0.5 text-xs font-medium whitespace-nowrap text-record-foreground">
      <Trophy className="size-3" aria-hidden />
      {t("records.pr")} · {t(`records.short.${kind}`)}
      {value !== undefined && ` ${formatRecord(kind, value, unit)}`}
    </span>
  );
}

/** A small yellow trophy next to a set that holds a record. */
export function RecordMark() {
  const { t } = useTranslation();
  return (
    <span className="inline-flex size-4 shrink-0 items-center justify-center self-center rounded-full bg-record text-record-foreground">
      <Trophy className="size-2.5" role="img" aria-label={t("records.pr")} />
    </span>
  );
}

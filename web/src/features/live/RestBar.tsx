import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@kipit-labs/kipitidy/button";

import { formatDuration } from "@/lib/units";
import type { Live } from "@/features/live/model";
import { restOver } from "@/features/live/signal";
import { dispatch } from "@/features/live/store";
import { useNow } from "@/features/live/useNow";

/** The rest timer over the tab bar: what's left, −15 s, +15 s, skip; beeps at the end. */
export function RestBar({ rest }: { rest: NonNullable<Live["rest"]> }) {
  const { t } = useTranslation();
  const now = useNow(250);
  const left = Math.ceil((rest.endsAt - now) / 1000);
  // A rest already over when the app opens (a reload, a locked phone) ends quietly.
  const [stale] = useState(() => rest.endsAt <= Date.now());
  useEffect(() => {
    if (left > 0) return;
    if (!stale) restOver();
    dispatch({ type: "restEnd" });
  }, [left, stale]);
  if (left <= 0) return null;
  return (
    <div role="timer" aria-label={t("live.rest")} className="border-t bg-background/95 px-4 py-2 backdrop-blur">
      <div className="mx-auto flex max-w-lg items-center gap-2">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium tabular-nums">
            {t("live.rest")} {formatDuration(left)} <span className="text-muted-foreground">/ {formatDuration(rest.totalS)}</span>
          </p>
          <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-muted">
            <div className="h-full bg-primary transition-[width]" style={{ width: `${Math.min(100, (left / rest.totalS) * 100)}%` }} />
          </div>
        </div>
        <Button size="sm" variant="outline" onClick={() => dispatch({ type: "restAdjust", by: -15, now: Date.now() })}>
          {t("live.minus15")}
        </Button>
        <Button size="sm" variant="outline" onClick={() => dispatch({ type: "restAdjust", by: 15, now: Date.now() })}>
          {t("live.plus15")}
        </Button>
        <Button size="sm" onClick={() => dispatch({ type: "restEnd" })}>
          {t("live.skip")}
        </Button>
      </div>
    </div>
  );
}

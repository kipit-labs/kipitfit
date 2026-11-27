import { Pause, Play } from "lucide-react";
import { useEffect, useState, useSyncExternalStore } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "cn";
import { Button } from "@kipit-labs/kipitidy/button";
import { IconTile } from "@kipit-labs/kipitidy/icon-tile";

import type { ExerciseSummary } from "@/lib/api";
import { equipmentIcon, mediaUrl } from "@/lib/exercises";

// How long each frame (start, peak) shows.
const frameMs = 900;

/** The exercise's first frame as a 40px tile, or its equipment icon when it has none (or it won't load). */
export function ExerciseThumb({ exercise: e, className }: { exercise: ExerciseSummary; className?: string }) {
  const key = e.media?.[0];
  const [failed, setFailed] = useState<string>();
  if (!key || failed === key) return <IconTile icon={equipmentIcon(e.equipment)} className={className} />;
  return (
    <img
      src={mediaUrl(key)}
      alt=""
      loading="lazy"
      decoding="async"
      onError={() => setFailed(key)}
      className={cn("size-10 shrink-0 rounded-lg bg-muted object-cover", className)}
    />
  );
}

const motionQuery = "(prefers-reduced-motion: reduce)";
const subscribeMotion = (cb: () => void) => {
  const mq = matchMedia(motionQuery);
  mq.addEventListener("change", cb);
  return () => mq.removeEventListener("change", cb);
};
const useReducedMotion = () => useSyncExternalStore(subscribeMotion, () => matchMedia(motionQuery).matches);

/**
 * The exercise's frames looping, with a pause button; one still frame under reduced motion. Nothing when it has
 * none or they won't load (onFail tells the page, which then shows the equipment icon).
 */
export function ExerciseFrames({ exercise: e, onFail }: { exercise: ExerciseSummary; onFail?: () => void }) {
  const { t } = useTranslation();
  const frames = e.media ?? [];
  const reduced = useReducedMotion();
  const [paused, setPaused] = useState(false);
  const [shown, setShown] = useState(0);
  const [failed, setFailed] = useState(false);
  const moving = frames.length > 1 && !reduced;
  const playing = moving && !paused;
  useEffect(() => {
    if (!playing) return;
    const id = setInterval(() => setShown((i) => (i + 1) % frames.length), frameMs);
    return () => clearInterval(id);
  }, [playing, frames.length]);
  if (!frames.length || failed) return null;
  const current = reduced ? 0 : shown % frames.length;
  return (
    <div className="relative aspect-square w-full overflow-hidden rounded-xl bg-muted">
      {frames.map((key, i) => (
        <img
          key={key}
          src={mediaUrl(key)}
          alt={i === 0 ? e.name : ""}
          aria-hidden={i !== 0}
          decoding="async"
          onError={() => {
            if (i > 0) return;
            setFailed(true);
            onFail?.();
          }}
          className={cn(
            "absolute inset-0 size-full object-contain transition-opacity duration-300 motion-reduce:transition-none",
            i === current ? "opacity-100" : "opacity-0",
          )}
        />
      ))}
      {moving && (
        <Button
          variant="secondary"
          size="icon-sm"
          className="absolute top-2 right-2 rounded-full opacity-80"
          aria-label={paused ? t("exercises.play") : t("exercises.pause")}
          aria-pressed={paused}
          onClick={() => setPaused((p) => !p)}
        >
          {paused ? <Play /> : <Pause />}
        </Button>
      )}
    </div>
  );
}

import { useState } from "react";
import { cn } from "cn";
import { Input } from "@kipit-labs/kipitidy/input";

/**
 * A set-table field. It keeps what is typed ("92." on the way to "92.5") and
 * reports each value that parses; one that doesn't is marked invalid.
 */
export function CellInput<T>({
  value,
  parse,
  onValue,
  className,
  ...props
}: Omit<React.ComponentProps<"input">, "value" | "onChange"> & {
  /** The value as text, read once when the field mounts. */
  value: string;
  parse: (text: string) => T | undefined;
  onValue: (v: T) => void;
}) {
  const [text, setText] = useState(value);
  const bad = parse(text) === undefined;
  return (
    <Input
      autoComplete="off"
      aria-invalid={bad || undefined}
      className={cn("h-8 px-1 text-center tabular-nums", className)}
      value={text}
      onChange={(e) => {
        setText(e.target.value);
        const v = parse(e.target.value);
        if (v !== undefined) onValue(v);
      }}
      {...props}
    />
  );
}

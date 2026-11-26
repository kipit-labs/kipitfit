# Exercise data

`exercises.json` is derived from [free-exercise-db](https://github.com/yuhonas/free-exercise-db) by yuhonas,
`dist/exercises.json` at commit `f00c92c7dcf1216a928a52c3706c7ce8e2f71ed5`, released under
[The Unlicense](https://github.com/yuhonas/free-exercise-db/blob/main/LICENSE.md) (public domain).

Trimmed to the fields kipitfit uses (no image paths), with category, equipment, force, level, mechanic and muscle
values turned into snake_case keys (`body only` → `body_only`, `e-z curl bar` → `ez_curl_bar`, missing equipment →
`none`, missing force or mechanic → empty). Names and instructions are unchanged.

package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/uptrace/bun"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/ids"
)

func (s *Store) LibraryPack(ctx context.Context) (domain.Pack, error) {
	var raw string
	err := s.db.NewSelect().Table("meta").Column("value").Where("key = ?", libraryPackKey).Scan(ctx, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Pack{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Pack{}, err
	}
	var p domain.Pack
	return p, json.Unmarshal([]byte(raw), &p)
}

func (s *Store) ReplaceLibrary(ctx context.Context, p domain.Pack, exs []domain.Exercise, repoint map[string]string) (domain.Replaced, error) {
	res := domain.Replaced{Exercises: len(exs)}
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		rows := make([]exerciseRow, len(exs))
		slugs := make([]string, len(exs))
		t := now()
		for i, e := range exs {
			e.ID, e.OwnerID, e.CreatedAt, e.UpdatedAt = ids.New(), "", t, t
			rows[i], slugs[i] = newExerciseRow(e), e.Slug
		}
		// An existing slug keeps its id and comes back if an earlier import archived it.
		_, err := tx.NewInsert().Model(&rows).
			On("CONFLICT (slug) DO UPDATE").
			Set("name = EXCLUDED.name, kind = EXCLUDED.kind, category = EXCLUDED.category, equipment = EXCLUDED.equipment").
			Set("primary_muscles = EXCLUDED.primary_muscles, secondary_muscles = EXCLUDED.secondary_muscles").
			Set("force = EXCLUDED.force, level = EXCLUDED.level, mechanic = EXCLUDED.mechanic").
			Set("instructions = EXCLUDED.instructions, media = EXCLUDED.media, updated_at = EXCLUDED.updated_at").
			Set("archived_at = NULL").
			Exec(ctx)
		if err != nil {
			return mapErr(err)
		}
		for from, to := range repoint {
			for _, table := range []string{"routine_exercises", "workout_exercises"} {
				r, err := tx.ExecContext(ctx, "UPDATE "+table+" SET exercise_id = (SELECT id FROM exercises WHERE slug = ?) "+
					"WHERE exercise_id = (SELECT id FROM exercises WHERE slug = ?)", to, from)
				if err != nil {
					return err
				}
				n, _ := r.RowsAffected()
				res.Repointed += int(n)
			}
		}
		// The rest of the old library is archived, not deleted: a workout under
		// way on a phone still holds its ids, and must still save.
		r, err := tx.NewUpdate().Model((*exerciseRow)(nil)).Set("archived_at = ?", t).
			Where("owner_id IS NULL").Where("slug NOT IN (?)", bun.List(slugs)).Where("archived_at IS NULL").Exec(ctx)
		if err != nil {
			return err
		}
		n, _ := r.RowsAffected()
		res.Archived = int(n)
		raw, err := json.Marshal(p)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = excluded.value",
			libraryPackKey, string(raw))
		return err
	})
	return res, err
}

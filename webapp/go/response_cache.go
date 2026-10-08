package main

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

type responseCacheKey struct{}
type responseCache struct {
	generation       uint64
	streamGeneration uint64
	pendingStreams   map[int64]streamMetadata
	users            map[int64]User
	streams          map[int64]Livestream
	tags             map[int64][]Tag
}

func responseCacheMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, _, generation := cachedIconHash("")
		cache := &responseCache{generation: generation, streamGeneration: currentStreamGeneration(), pendingStreams: make(map[int64]streamMetadata), users: make(map[int64]User), streams: make(map[int64]Livestream), tags: make(map[int64][]Tag)}
		c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), responseCacheKey{}, cache)))
		err := next(c)
		if err == nil && c.Response().Status >= 200 && c.Response().Status < 300 {
			publishStreamMetadata(cache.pendingStreams, cache.streamGeneration)
		}
		return err
	}
}
func responses(ctx context.Context) *responseCache {
	return ctx.Value(responseCacheKey{}).(*responseCache)
}

// SQL reads use the caller transaction. Immutable metadata is published only
// after the handler succeeds; mutable owner/icon metadata is loaded separately.
func prefetchUsers(ctx context.Context, tx *sqlx.Tx, ids []int64) error {
	cache := responses(ctx)
	seen := make(map[int64]bool)
	wanted := make([]int64, 0, len(ids))
	for _, id := range ids {
		if user, ok := cachedUserMetadata(id); ok {
			cache.users[id] = user
		}
		if _, ok := cache.users[id]; !ok && !seen[id] {
			seen[id] = true
			wanted = append(wanted, id)
		}
	}
	if len(wanted) == 0 {
		return nil
	}
	query, args, err := sqlx.In("SELECT * FROM users WHERE id IN (?)", wanted)
	if err != nil {
		return err
	}
	var users []UserModel
	if err := tx.SelectContext(ctx, &users, query, args...); err != nil {
		return err
	}
	query, args, err = sqlx.In("SELECT * FROM themes WHERE user_id IN (?) ORDER BY id", wanted)
	if err != nil {
		return err
	}
	var themes []ThemeModel
	if err := tx.SelectContext(ctx, &themes, query, args...); err != nil {
		return err
	}
	themeByUser := make(map[int64]ThemeModel)
	for _, theme := range themes {
		if _, ok := themeByUser[theme.UserID]; !ok {
			themeByUser[theme.UserID] = theme
		}
	}
	var icons []struct {
		UserID int64  `db:"user_id"`
		Hash   string `db:"image_hash"`
	}
	query, args, err = sqlx.In("SELECT user_id,image_hash FROM icons WHERE user_id IN (?) ORDER BY id", wanted)
	if err != nil {
		return err
	}
	if err := tx.SelectContext(ctx, &icons, query, args...); err != nil {
		return err
	}
	hashes := make(map[int64]string)
	for _, icon := range icons {
		if _, ok := hashes[icon.UserID]; !ok {
			hashes[icon.UserID] = icon.Hash
		}
	}
	fallback := ""
	for _, model := range users {
		theme, ok := themeByUser[model.ID]
		if !ok {
			return sql.ErrNoRows
		}
		hash, ok := hashes[model.ID]
		if !ok {
			if fallback == "" {
				var err error
				fallback, err = defaultIconHash()
				if err != nil {
					return err
				}
			}
			hash = fallback
		}
		cache.users[model.ID] = User{ID: model.ID, Name: model.Name, DisplayName: model.DisplayName, Description: model.Description, Theme: Theme{ID: theme.ID, DarkMode: theme.DarkMode}, IconHash: hash}
		rememberUserMetadata(cache.users[model.ID], cache.generation)
	}
	return nil
}
func loadUserResponse(ctx context.Context, tx *sqlx.Tx, id int64) (User, error) {
	cache := responses(ctx)
	if user, ok := cache.users[id]; ok {
		return user, nil
	}
	if err := prefetchUsers(ctx, tx, []int64{id}); err != nil {
		return User{}, err
	}
	user, ok := cache.users[id]
	if !ok {
		return User{}, sql.ErrNoRows
	}
	return user, nil
}
func prefetchStreams(ctx context.Context, tx *sqlx.Tx, models []*LivestreamModel) error {
	cache := responses(ctx)
	var ids, owners []int64
	for _, model := range models {
		if _, ok := cache.streams[model.ID]; !ok {
			owners = append(owners, model.UserID)
			if entry, ok := cachedStreamMetadata(model.ID, cache.streamGeneration); ok {
				cache.tags[model.ID] = entry.tags
			} else {
				ids = append(ids, model.ID)
			}
		}
	}
	if err := prefetchUsers(ctx, tx, owners); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	var tags []struct {
		StreamID int64          `db:"livestream_id"`
		ID       sql.NullInt64  `db:"id"`
		Name     sql.NullString `db:"name"`
	}
	query, args, err := sqlx.In(`SELECT lt.livestream_id,t.id,t.name FROM livestream_tags lt LEFT JOIN tags t ON t.id=lt.tag_id WHERE lt.livestream_id IN (?) ORDER BY lt.id`, ids)
	if err != nil {
		return err
	}
	if err := tx.SelectContext(ctx, &tags, query, args...); err != nil {
		return err
	}
	for _, id := range ids {
		cache.tags[id] = make([]Tag, 0)
	}
	for _, tag := range tags {
		if !tag.ID.Valid || !tag.Name.Valid {
			return sql.ErrNoRows
		}
		cache.tags[tag.StreamID] = append(cache.tags[tag.StreamID], Tag{ID: tag.ID.Int64, Name: tag.Name.String})
	}
	for _, model := range models {
		cache.pendingStreams[model.ID] = streamMetadata{model: *model, tags: cache.tags[model.ID]}
	}
	return nil
}
func loadLivestreamResponse(ctx context.Context, tx *sqlx.Tx, id int64) (Livestream, error) {
	if stream, ok := responses(ctx).streams[id]; ok {
		return stream, nil
	}
	if entry, ok := cachedStreamMetadata(id, responses(ctx).streamGeneration); ok {
		responses(ctx).tags[id] = entry.tags
		return fillLivestreamResponse(ctx, tx, entry.model)
	}
	var model LivestreamModel
	if err := tx.GetContext(ctx, &model, "SELECT * FROM livestreams WHERE id = ?", id); err != nil {
		return Livestream{}, err
	}
	return fillLivestreamResponse(ctx, tx, model)
}

// Warm both stream owners and comment/reaction authors in one user batch.
func prefetchReferencedResponses(ctx context.Context, tx *sqlx.Tx, streamIDs, userIDs []int64) error {
	if len(streamIDs) == 0 {
		return nil
	}
	unique := make(map[int64]bool)
	ids := make([]int64, 0, len(streamIDs))
	var models []*LivestreamModel
	for _, id := range streamIDs {
		if !unique[id] {
			unique[id] = true
			if entry, ok := cachedStreamMetadata(id, responses(ctx).streamGeneration); ok {
				model := entry.model
				models = append(models, &model)
				responses(ctx).tags[id] = entry.tags
			} else {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) > 0 {
		query, args, err := sqlx.In("SELECT * FROM livestreams WHERE id IN (?)", ids)
		if err != nil {
			return err
		}
		var missing []*LivestreamModel
		if err := tx.SelectContext(ctx, &missing, query, args...); err != nil {
			return err
		}
		models = append(models, missing...)
	}
	for _, model := range models {
		userIDs = append(userIDs, model.UserID)
	}
	if err := prefetchUsers(ctx, tx, userIDs); err != nil {
		return err
	}
	if err := prefetchStreams(ctx, tx, models); err != nil {
		return err
	}
	for _, model := range models {
		if _, err := fillLivestreamResponse(ctx, tx, *model); err != nil {
			return err
		}
	}
	return nil
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type ReactionModel struct {
	ID           int64  `db:"id"`
	EmojiName    string `db:"emoji_name"`
	UserID       int64  `db:"user_id"`
	LivestreamID int64  `db:"livestream_id"`
	CreatedAt    int64  `db:"created_at"`
}

type Reaction struct {
	ID         int64      `json:"id"`
	EmojiName  string     `json:"emoji_name"`
	User       User       `json:"user"`
	Livestream Livestream `json:"livestream"`
	CreatedAt  int64      `json:"created_at"`
}

type PostReactionRequest struct {
	EmojiName string `json:"emoji_name"`
}

func getReactionsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	if err := verifyUserSession(c); err != nil {
		// echo.NewHTTPErrorが返っているのでそのまま出力
		return err
	}

	livestreamID, err := strconv.Atoi(c.Param("livestream_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "livestream_id in path must be integer")
	}

	tx, err := dbConn.BeginTxx(ctx, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to begin transaction: "+err.Error())
	}
	defer tx.Rollback()

	query := "SELECT * FROM reactions WHERE livestream_id = ? ORDER BY created_at DESC"
	if c.QueryParam("limit") != "" {
		limit, err := strconv.Atoi(c.QueryParam("limit"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "limit query parameter must be integer")
		}
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	reactionModels := []ReactionModel{}
	if err := tx.SelectContext(ctx, &reactionModels, query, livestreamID); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "failed to get reactions")
	}

	var streamIDs, userIDs []int64
	for _, model := range reactionModels {
		streamIDs = append(streamIDs, model.LivestreamID)
		userIDs = append(userIDs, model.UserID)
	}
	if err := prefetchReferencedResponses(ctx, tx, streamIDs, userIDs); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to prefetch responses: "+err.Error())
	}
	reactions := make([]Reaction, len(reactionModels))
	for i := range reactionModels {
		reaction, err := fillReactionResponse(ctx, tx, reactionModels[i])
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to fill reaction: "+err.Error())
		}

		reactions[i] = reaction
	}

	if err := tx.Commit(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to commit: "+err.Error())
	}

	return c.JSON(http.StatusOK, reactions)
}

func postReactionHandler(c echo.Context) error {
	ctx := c.Request().Context()
	livestreamID, err := strconv.Atoi(c.Param("livestream_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "livestream_id in path must be integer")
	}

	if err := verifyUserSession(c); err != nil {
		// echo.NewHTTPErrorが返っているのでそのまま出力
		return err
	}

	// error already checked
	sess, _ := session.Get(defaultSessionIDKey, c)
	// existence already checked
	userID := sess.Values[defaultUserIDKey].(int64)

	var req *PostReactionRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to decode the request body as json")
	}

	value, err := enqueueWrite(&writeJob{ctx: ctx, epoch: responses(ctx).writeEpoch, userID: userID, streamID: int64(livestreamID), reaction: true, emoji: req.EmojiName})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, value)
}

func fillReactionResponse(ctx context.Context, tx *sqlx.Tx, reactionModel ReactionModel) (Reaction, error) {
	user, err := loadUserResponse(ctx, tx, reactionModel.UserID)
	if err != nil {
		return Reaction{}, err
	}
	livestream, err := loadLivestreamResponse(ctx, tx, reactionModel.LivestreamID)
	if err != nil {
		return Reaction{}, err
	}

	reaction := Reaction{
		ID:         reactionModel.ID,
		EmojiName:  reactionModel.EmojiName,
		User:       user,
		Livestream: livestream,
		CreatedAt:  reactionModel.CreatedAt,
	}

	return reaction, nil
}

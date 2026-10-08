package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

type writeJob struct {
	ctx              context.Context
	epoch            uint64
	userID, streamID int64
	comment          string
	tip              int64
	emoji            string
	reaction         bool
	done             chan writeResult
}
type writeResult struct {
	value interface{}
	err   error
}

var writeQueue = make(chan *writeJob, 1024)
var writeLifecycle sync.RWMutex
var writeEpoch atomic.Uint64
var writeBatches, writeRows, writeMaxBatch atomic.Int64

func enqueueWrite(job *writeJob) (interface{}, error) {
	job.done = make(chan writeResult, 1)
	select {
	case writeQueue <- job:
	case <-job.ctx.Done():
		return nil, job.ctx.Err()
	}
	select {
	case result := <-job.done:
		return result.value, result.err
	case <-job.ctx.Done():
		return nil, job.ctx.Err()
	}
}
func writeBatchLoop() {
	for {
		jobs := []*writeJob{<-writeQueue}
		timer := time.NewTimer(time.Millisecond)
	collect:
		for len(jobs) < 64 {
			select {
			case job := <-writeQueue:
				jobs = append(jobs, job)
			case <-timer.C:
				break collect
			}
		}
		timer.Stop()
		writeLifecycle.RLock()
		active := make([]*writeJob, 0, len(jobs))
		for _, job := range jobs {
			if job.epoch != writeEpoch.Load() {
				job.done <- writeResult{err: echo.NewHTTPError(503, "request predates initialization")}
			} else if job.ctx.Err() != nil {
				job.done <- writeResult{err: job.ctx.Err()}
			} else {
				active = append(active, job)
			}
		}
		if len(active) > 0 {
			writeBatches.Add(1)
			for old := writeMaxBatch.Load(); int64(len(active)) > old; old = writeMaxBatch.Load() {
				if writeMaxBatch.CompareAndSwap(old, int64(len(active))) {
					break
				}
			}
			results, err := commitWriteBatch(active)
			if err != nil {
				for i := range results {
					if results[i].err == nil {
						results[i] = writeResult{err: echo.NewHTTPError(500, "failed to commit write batch: "+err.Error())}
					}
				}
			}
			for i, job := range active {
				job.done <- results[i]
			}
		}
		writeLifecycle.RUnlock()
	}
}
func commitWriteBatch(jobs []*writeJob) (results []writeResult, err error) {
	results = make([]writeResult, len(jobs))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, generation := cachedIconHash("")
	cache := &responseCache{generation: generation, users: make(map[int64]User), streams: make(map[int64]Livestream), tags: make(map[int64][]Tag)}
	ctx = context.WithValue(ctx, responseCacheKey{}, cache)
	tx, err := dbConn.BeginTxx(ctx, nil)
	if err != nil {
		return results, err
	}
	defer tx.Rollback()
	ids := make([]int64, 0, len(jobs))
	seen := make(map[int64]bool)
	for _, job := range jobs {
		if !seen[job.streamID] {
			seen[job.streamID] = true
			ids = append(ids, job.streamID)
		}
	}
	// Lock in primary-key order before any consistent read, including NG-word reads.
	query, args, err := sqlx.In("SELECT * FROM livestreams WHERE id IN (?) ORDER BY id FOR UPDATE", ids)
	if err != nil {
		return results, err
	}
	var streams []*LivestreamModel
	if err = tx.SelectContext(ctx, &streams, query, args...); err != nil {
		return results, err
	}
	byID := make(map[int64]*LivestreamModel)
	for _, stream := range streams {
		byID[stream.ID] = stream
	}
	var comments []LivecommentModel
	var reactions []ReactionModel
	var commentJobs, reactionJobs []int
	var authors []int64
	for i, job := range jobs {
		stream, ok := byID[job.streamID]
		if !ok {
			status := 404
			if job.reaction {
				status = 500
			}
			results[i].err = echo.NewHTTPError(status, "livestream not found")
			continue
		}
		if !job.reaction {
			var spam int
			query = `SELECT EXISTS(SELECT 1 FROM ng_words WHERE user_id=? AND livestream_id=?
    AND CONVERT(? USING utf8mb4) COLLATE utf8mb4_general_ci
    LIKE CONCAT('%',CONVERT(word USING utf8mb4) COLLATE utf8mb4_general_ci,'%'))`
			if err = tx.GetContext(ctx, &spam, query, stream.UserID, stream.ID, job.comment); err != nil {
				return results, err
			}
			if spam != 0 {
				results[i].err = echo.NewHTTPError(400, "このコメントがスパム判定されました")
				continue
			}
			comments = append(comments, LivecommentModel{UserID: job.userID, LivestreamID: job.streamID, Comment: job.comment, Tip: job.tip, CreatedAt: time.Now().Unix()})
			commentJobs = append(commentJobs, i)
		} else {
			reactions = append(reactions, ReactionModel{UserID: job.userID, LivestreamID: job.streamID, EmojiName: job.emoji, CreatedAt: time.Now().Unix()})
			reactionJobs = append(reactionJobs, i)
		}
		authors = append(authors, job.userID)
	}
	if len(authors) == 0 {
		return results, nil
	}
	// Fixed deployment requires auto_increment_increment=1, verified at startup.
	// Plain multi-row INSERT reserves contiguous IDs; no IGNORE/upsert or triggers.
	if len(comments) > 0 {
		args = make([]interface{}, 0, len(comments)*5)
		for _, model := range comments {
			args = append(args, model.UserID, model.LivestreamID, model.Comment, model.Tip, model.CreatedAt)
		}
		query = "INSERT INTO livecomments(user_id,livestream_id,comment,tip,created_at) VALUES " + strings.TrimSuffix(strings.Repeat("(?,?,?,?,?),", len(comments)), ",")
		result, e := tx.ExecContext(ctx, query, args...)
		if e != nil {
			return results, e
		}
		first, e := result.LastInsertId()
		if e != nil {
			return results, e
		}
		for i := range comments {
			comments[i].ID = first + int64(i)
		}
	}
	if len(reactions) > 0 {
		args = make([]interface{}, 0, len(reactions)*4)
		for _, model := range reactions {
			args = append(args, model.UserID, model.LivestreamID, model.EmojiName, model.CreatedAt)
		}
		query = "INSERT INTO reactions(user_id,livestream_id,emoji_name,created_at) VALUES " + strings.TrimSuffix(strings.Repeat("(?,?,?,?),", len(reactions)), ",")
		result, e := tx.ExecContext(ctx, query, args...)
		if e != nil {
			return results, e
		}
		first, e := result.LastInsertId()
		if e != nil {
			return results, e
		}
		for i := range reactions {
			reactions[i].ID = first + int64(i)
		}
	}
	if err = prefetchUsers(ctx, tx, authors); err != nil {
		return results, err
	}
	if err = prefetchStreams(ctx, tx, streams); err != nil {
		return results, err
	}
	for _, stream := range streams {
		if _, err = fillLivestreamResponse(ctx, tx, *stream); err != nil {
			return results, err
		}
	}
	for i, model := range comments {
		value, e := fillLivecommentResponse(ctx, tx, model)
		if e != nil {
			return results, e
		}
		results[commentJobs[i]].value = value
	}
	for i, model := range reactions {
		value, e := fillReactionResponse(ctx, tx, model)
		if e != nil {
			return results, e
		}
		results[reactionJobs[i]].value = value
	}
	if err = tx.Commit(); err != nil {
		return results, err
	}
	writeRows.Add(int64(len(authors)))
	// Results are published only by the caller, after this durable commit.
	return results, nil
}
func startWriteBatcher() error {
	var increment int
	if err := dbConn.Get(&increment, "SELECT @@auto_increment_increment"); err != nil {
		return err
	}
	if increment != 1 {
		return fmt.Errorf("batch insert requires auto_increment_increment=1, got%d", increment)
	}
	go writeBatchLoop()
	go writeBatchLoop()
	return nil
}

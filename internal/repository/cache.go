package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"task_tracker/internal/model"
)

// CacheRepository provides Redis caching for task lists.
type CacheRepository struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// NewCacheRepository creates a new CacheRepository.
func NewCacheRepository(rdb *redis.Client, logger *slog.Logger) *CacheRepository {
	return &CacheRepository{
		rdb:    rdb,
		logger: logger,
	}
}

// CacheTTL is the time-to-live for cached task lists.
const CacheTTL = 5 * time.Minute

// buildCacheKey creates a unique cache key based on team and filters.
func buildCacheKey(filter model.TaskFilter) string {
	key := fmt.Sprintf("tasks:%d", filter.TeamID)

	if filter.Status != nil {
		key += fmt.Sprintf(":status:%s", *filter.Status)
	}

	if filter.AssigneeID != nil {
		key += fmt.Sprintf(":assignee:%d", *filter.AssigneeID)
	}

	if filter.Limit > 0 {
		key += fmt.Sprintf(":limit:%d", filter.Limit)
	}

	if filter.Offset > 0 {
		key += fmt.Sprintf(":offset:%d", filter.Offset)
	}

	return key
}

// GetTaskList retrieves cached task list for the given filter.
// Returns (tasks, true, nil) on cache hit (tasks may be empty),
// (nil, false, nil) on cache miss, and (nil, false, err) on Redis error.
func (r *CacheRepository) GetTaskList(ctx context.Context, filter model.TaskFilter) ([]model.Task, bool, error) {
	key := buildCacheKey(filter)

	data, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			r.logger.Debug("cache miss", slog.String("key", key))
			return nil, false, nil // Cache miss
		}
		r.logger.Error("cache get error", slog.String("key", key), slog.Any("error", err))
		return nil, false, err
	}

	var tasks []model.Task
	if err := json.Unmarshal([]byte(data), &tasks); err != nil {
		r.logger.Error("cache unmarshal error", slog.String("key", key), slog.Any("error", err))
		return nil, false, err
	}

	r.logger.Debug("cache hit", slog.String("key", key), slog.Int("task_count", len(tasks)))
	return tasks, true, nil
}

// SetTaskList caches task list for the given filter.
func (r *CacheRepository) SetTaskList(ctx context.Context, filter model.TaskFilter, tasks []model.Task) error {
	key := buildCacheKey(filter)

	data, err := json.Marshal(tasks)
	if err != nil {
		r.logger.Error("cache marshal error", slog.String("key", key), slog.Any("error", err))
		return err
	}

	if err := r.rdb.Set(ctx, key, data, CacheTTL).Err(); err != nil {
		r.logger.Error("cache set error", slog.String("key", key), slog.Any("error", err))
		return err
	}

	r.logger.Debug("cache set", slog.String("key", key), slog.Int("task_count", len(tasks)))
	return nil
}

// InvalidateTeamTasks invalidates all cached task lists for a team.
// Called when tasks are created or updated.
func (r *CacheRepository) InvalidateTeamTasks(ctx context.Context, teamID int64) error {
	// Use pattern matching to find all keys for this team
	pattern := fmt.Sprintf("tasks:%d:*", teamID)

	// Also invalidate the base key (without filters)
	baseKey := fmt.Sprintf("tasks:%d", teamID)
	if err := r.rdb.Del(ctx, baseKey).Err(); err != nil {
		r.logger.Error("cache invalidate error", slog.Int64("team_id", teamID), slog.Any("error", err))
		return err
	}

	// Find and delete all matching keys
	deletedCount := 0
	iter := r.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := r.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			r.logger.Error("cache invalidate error", slog.String("key", iter.Val()), slog.Any("error", err))
			return err
		}
		deletedCount++
	}

	if err := iter.Err(); err != nil {
		r.logger.Error("cache scan error", slog.Int64("team_id", teamID), slog.Any("error", err))
		return err
	}

	r.logger.Debug("cache invalidated", slog.Int64("team_id", teamID), slog.Int("deleted_keys", deletedCount+1))
	return nil
}

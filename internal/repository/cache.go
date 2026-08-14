package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"task_tracker/internal/model"
)

// CacheRepository provides Redis caching for task lists.
type CacheRepository struct {
	rdb *redis.Client
}

// NewCacheRepository creates a new CacheRepository.
func NewCacheRepository(rdb *redis.Client) *CacheRepository {
	return &CacheRepository{rdb: rdb}
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
// Returns nil if not found in cache.
func (r *CacheRepository) GetTaskList(ctx context.Context, filter model.TaskFilter) ([]model.Task, error) {
	key := buildCacheKey(filter)

	data, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var tasks []model.Task
	if err := json.Unmarshal([]byte(data), &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

// SetTaskList caches task list for the given filter.
func (r *CacheRepository) SetTaskList(ctx context.Context, filter model.TaskFilter, tasks []model.Task) error {
	key := buildCacheKey(filter)

	data, err := json.Marshal(tasks)
	if err != nil {
		return err
	}

	return r.rdb.Set(ctx, key, data, CacheTTL).Err()
}

// InvalidateTeamTasks invalidates all cached task lists for a team.
// Called when tasks are created or updated.
func (r *CacheRepository) InvalidateTeamTasks(ctx context.Context, teamID int64) error {
	// Use pattern matching to find all keys for this team
	pattern := fmt.Sprintf("tasks:%d:*", teamID)

	// Also invalidate the base key (without filters)
	baseKey := fmt.Sprintf("tasks:%d", teamID)
	if err := r.rdb.Del(ctx, baseKey).Err(); err != nil {
		return err
	}

	// Find and delete all matching keys
	iter := r.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := r.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}

	if err := iter.Err(); err != nil {
		return err
	}

	return nil
}

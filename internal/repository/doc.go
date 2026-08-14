// Package repository contains data-access code: SQL queries, transactions,
// and Redis cache operations.
//
// Layout (to be implemented):
//   - user.go    — users table
//   - team.go    — teams + team_members (atomic creation via transactions)
//   - task.go    — tasks + task_history (atomic update + history write,
//                  optimistic locking via version column)
//   - comment.go — task_comments
//   - stats.go   — single-report SQL: JOINs, GROUP BY, aggregates, date math
//   - cache.go   — Redis task-list cache (TTL 5m, filter-aware, invalidation)
package repository

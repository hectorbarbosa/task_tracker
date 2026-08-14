// Package service contains business logic: authorization checks, transaction
// orchestration, Redis cache invalidation, and anything that doesn't belong
// in the HTTP or data-access layers.
//
// Layout (to be implemented):
//   - auth.go     — registration, login, JWT issuance
//   - team.go     — team creation, invitations, role management
//   - task.go     — task CRUD with history recording and cache invalidation
//   - comment.go  — comment CRUD
//   - stats.go    — team statistics (single CTE/JOIN query, no N+1)
package service

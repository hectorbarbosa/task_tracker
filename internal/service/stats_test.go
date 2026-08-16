package service

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"task_tracker/internal/model"
	"task_tracker/internal/repository"
)

// TestIntegration_StatsService tests the stats service with real database.
// Requires MySQL database with test data.
func TestIntegration_StatsService(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Database connection
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "task_user:task_pass@tcp(127.0.0.1:3306)/task_tracker?parseTime=true"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Clean up before test
	cleanupTestData(t, db)

	// Create test data
	testData := createTestData(t, db)

	// Initialize repositories and service
	statsRepo := repository.NewStatsRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	statsService := NewStatsService(statsRepo, teamRepo)

	ctx := context.Background()

	// Test 1: Owner can view stats
	t.Run("OwnerCanViewStats", func(t *testing.T) {
		stats, err := statsService.GetTeamStats(ctx, testData.teamID, testData.ownerID)
		if err != nil {
			t.Fatalf("Owner should be able to view stats: %v", err)
		}

		// Verify stats
		if stats.ByStatus[model.TaskStatusTodo] != 2 {
			t.Errorf("Expected 2 todo tasks, got %d", stats.ByStatus[model.TaskStatusTodo])
		}
		if stats.ByStatus[model.TaskStatusInProgress] != 1 {
			t.Errorf("Expected 1 in_progress task, got %d", stats.ByStatus[model.TaskStatusInProgress])
		}
		if stats.ByStatus[model.TaskStatusDone] != 3 {
			t.Errorf("Expected 3 done tasks, got %d", stats.ByStatus[model.TaskStatusDone])
		}

		if len(stats.TopAssignees) == 0 {
			t.Error("Expected top assignees, got empty list")
		}

		if stats.TotalComments != 5 {
			t.Errorf("Expected 5 comments, got %d", stats.TotalComments)
		}

		if stats.AvgCloseSec <= 0 {
			t.Errorf("Expected positive avg close time, got %f", stats.AvgCloseSec)
		}
	})

	// Test 2: Admin can view stats
	t.Run("AdminCanViewStats", func(t *testing.T) {
		stats, err := statsService.GetTeamStats(ctx, testData.teamID, testData.adminID)
		if err != nil {
			t.Fatalf("Admin should be able to view stats: %v", err)
		}

		if stats.ByStatus[model.TaskStatusDone] != 3 {
			t.Errorf("Expected 3 done tasks, got %d", stats.ByStatus[model.TaskStatusDone])
		}
	})

	// Test 3: Member cannot view stats
	t.Run("MemberCannotViewStats", func(t *testing.T) {
		_, err := statsService.GetTeamStats(ctx, testData.teamID, testData.memberID)
		if err == nil {
			t.Fatal("Member should not be able to view stats")
		}
		if err.Error() != "only owner or admin can view team statistics" {
			t.Errorf("Expected permission error, got: %v", err)
		}
	})

	// Test 4: Non-member cannot view stats
	t.Run("NonMemberCannotViewStats", func(t *testing.T) {
		_, err := statsService.GetTeamStats(ctx, testData.teamID, 999)
		if err == nil {
			t.Fatal("Non-member should not be able to view stats")
		}
	})

	// Test 5: Empty team returns zero counts
	t.Run("EmptyTeamReturnsZeroCounts", func(t *testing.T) {
		emptyTeamID := createEmptyTeam(t, db, testData.ownerID)

		stats, err := statsService.GetTeamStats(ctx, emptyTeamID, testData.ownerID)
		if err != nil {
			t.Fatalf("Failed to get stats for empty team: %v", err)
		}

		if len(stats.ByStatus) != 0 {
			t.Errorf("Expected empty status map, got %v", stats.ByStatus)
		}

		if len(stats.TopAssignees) != 0 {
			t.Errorf("Expected empty top assignees, got %v", stats.TopAssignees)
		}

		if stats.TotalComments != 0 {
			t.Errorf("Expected 0 comments, got %d", stats.TotalComments)
		}
	})

	// Clean up
	cleanupTestData(t, db)
}

type testData struct {
	ownerID  int64
	adminID  int64
	memberID int64
	teamID   int64
}

func createTestData(t *testing.T, db *sql.DB) testData {
	// Create users
	ownerID := createUser(t, db, "owner@test.com", "owner")
	adminID := createUser(t, db, "admin@test.com", "admin")
	memberID := createUser(t, db, "member@test.com", "member")
	assignee1ID := createUser(t, db, "assignee1@test.com", "Assignee One")
	assignee2ID := createUser(t, db, "assignee2@test.com", "Assignee Two")

	// Create team
	teamID := createTeam(t, db, ownerID, "Test Team")

	// Add members
	addTeamMember(t, db, teamID, adminID, model.RoleAdmin)
	addTeamMember(t, db, teamID, memberID, model.RoleMember)
	addTeamMember(t, db, teamID, assignee1ID, model.RoleMember)
	addTeamMember(t, db, teamID, assignee2ID, model.RoleMember)

	// Create tasks
	task1ID := createTask(t, db, teamID, ownerID, &assignee1ID, model.TaskStatusTodo, "Task 1")
	task2ID := createTask(t, db, teamID, ownerID, &assignee2ID, model.TaskStatusTodo, "Task 2")
	createTask(t, db, teamID, adminID, &assignee1ID, model.TaskStatusInProgress, "Task 3")
	task4ID := createTask(t, db, teamID, memberID, &assignee1ID, model.TaskStatusDone, "Task 4")
	task5ID := createTask(t, db, teamID, ownerID, &assignee2ID, model.TaskStatusDone, "Task 5")
	task6ID := createTask(t, db, teamID, adminID, &assignee1ID, model.TaskStatusDone, "Task 6")

	// Update created_at + closed_at for done tasks so closed_at > created_at.
	// created_at must be before closed_at for TIMESTAMPDIFF to produce a positive value.
	setTaskTimes(t, db, task4ID, time.Now().Add(-72*time.Hour), time.Now().Add(-48*time.Hour)) // open 24h
	setTaskTimes(t, db, task5ID, time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour)) // open 24h
	setTaskTimes(t, db, task6ID, time.Now().Add(-36*time.Hour), time.Now().Add(-12*time.Hour)) // open 24h

	// Create comments
	createComment(t, db, task1ID, ownerID, "Comment 1")
	createComment(t, db, task1ID, adminID, "Comment 2")
	createComment(t, db, task2ID, memberID, "Comment 3")
	createComment(t, db, task4ID, assignee1ID, "Comment 4")
	createComment(t, db, task5ID, assignee2ID, "Comment 5")

	return testData{
		ownerID:  ownerID,
		adminID:  adminID,
		memberID: memberID,
		teamID:   teamID,
	}
}

func createUser(t *testing.T, db *sql.DB, email, name string) int64 {
	query := `INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`
	result, err := db.Exec(query, email, "hash", name)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get user ID: %v", err)
	}
	return id
}

func createTeam(t *testing.T, db *sql.DB, ownerID int64, name string) int64 {
	query := `INSERT INTO teams (name, created_by) VALUES (?, ?)`
	result, err := db.Exec(query, name, ownerID)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}
	teamID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get team ID: %v", err)
	}

	// Add owner as member
	query = `INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`
	_, err = db.Exec(query, teamID, ownerID, model.RoleOwner)
	if err != nil {
		t.Fatalf("Failed to add owner to team: %v", err)
	}

	return teamID
}

func addTeamMember(t *testing.T, db *sql.DB, teamID, userID int64, role model.Role) {
	query := `INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`
	_, err := db.Exec(query, teamID, userID, role)
	if err != nil {
		t.Fatalf("Failed to add team member: %v", err)
	}
}

func createTask(t *testing.T, db *sql.DB, teamID, createdBy int64, assigneeID *int64, status model.TaskStatus, title string) int64 {
	query := `INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := db.Exec(query, teamID, title, "Description", status, createdBy, assigneeID)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get task ID: %v", err)
	}
	return id
}

func setTaskTimes(t *testing.T, db *sql.DB, taskID int64, createdAt, closedAt time.Time) {
	t.Helper()
	query := `UPDATE tasks SET created_at = ?, closed_at = ? WHERE id = ?`
	_, err := db.Exec(query, createdAt, closedAt, taskID)
	if err != nil {
		t.Fatalf("Failed to update task times: %v", err)
	}
}

func createComment(t *testing.T, db *sql.DB, taskID, userID int64, content string) {
	query := `INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`
	_, err := db.Exec(query, taskID, userID, content)
	if err != nil {
		t.Fatalf("Failed to create comment: %v", err)
	}
}

func createEmptyTeam(t *testing.T, db *sql.DB, ownerID int64) int64 {
	return createTeam(t, db, ownerID, "Empty Team")
}

func cleanupTestData(t *testing.T, db *sql.DB) {
	// Delete in reverse order to respect foreign keys
	tables := []string{
		"task_comments",
		"task_history",
		"tasks",
		"team_members",
		"teams",
		"users",
	}

	for _, table := range tables {
		_, err := db.Exec("DELETE FROM " + table)
		if err != nil {
			t.Logf("Warning: failed to clean up table %s: %v", table, err)
		}
	}
}

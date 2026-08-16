package service

import (
	"context"
	"errors"
	"testing"

	"task_tracker/internal/model"
	"task_tracker/internal/repository"
)

// ─── Mock implementations ─────────────────────────────────────────────────────

// mockTaskStore implements taskStore for unit tests.
type mockTaskStore struct {
	tasks       map[int64]*model.Task
	history     map[int64][]model.TaskHistory
	createErr   error
	getErr      error
	updateErr   error
	historyErr  error
	lastChanges map[string]interface{} // captured from last UpdateTask call
}

func newMockTaskStore() *mockTaskStore {
	return &mockTaskStore{
		tasks:   make(map[int64]*model.Task),
		history: make(map[int64][]model.TaskHistory),
	}
}

func (m *mockTaskStore) CreateTask(_ context.Context, task *model.Task) error {
	if m.createErr != nil {
		return m.createErr
	}
	if task.ID == 0 {
		task.ID = int64(len(m.tasks) + 1)
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskStore) GetTaskByID(_ context.Context, taskID int64) (*model.Task, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	task, ok := m.tasks[taskID]
	if !ok {
		return nil, nil
	}
	return task, nil
}

func (m *mockTaskStore) ListTasks(_ context.Context, filter model.TaskFilter) ([]model.Task, error) {
	var result []model.Task
	for _, t := range m.tasks {
		if t.TeamID == filter.TeamID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTaskStore) UpdateTask(_ context.Context, task *model.Task, changes map[string]interface{}) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.tasks[task.ID] = task
	m.lastChanges = changes
	task.Version++
	return nil
}

func (m *mockTaskStore) ListTaskHistory(_ context.Context, taskID int64) ([]model.TaskHistory, error) {
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	return m.history[taskID], nil
}

// mockTeamStore implements teamStore for unit tests.
type mockTeamStore struct {
	teams      map[int64]*model.Team
	members    map[int64]map[int64]model.Role // teamID -> userID -> role
	inviteErr  error
}

func newMockTeamStore() *mockTeamStore {
	return &mockTeamStore{
		teams:   make(map[int64]*model.Team),
		members: make(map[int64]map[int64]model.Role),
	}
}

func (m *mockTeamStore) addTeam(team *model.Team) {
	m.teams[team.ID] = team
	if m.members[team.ID] == nil {
		m.members[team.ID] = make(map[int64]model.Role)
	}
	m.members[team.ID][team.CreatedBy] = model.RoleOwner
}

func (m *mockTeamStore) addMember(teamID, userID int64, role model.Role) {
	if m.members[teamID] == nil {
		m.members[teamID] = make(map[int64]model.Role)
	}
	m.members[teamID][userID] = role
}

func (m *mockTeamStore) CreateTeam(_ context.Context, team *model.Team, ownerID int64) error {
	team.CreatedBy = ownerID
	if team.ID == 0 {
		team.ID = int64(len(m.teams) + 1)
	}
	m.addTeam(team)
	return nil
}

func (m *mockTeamStore) GetTeamByID(_ context.Context, teamID int64) (*model.Team, error) {
	team, ok := m.teams[teamID]
	if !ok {
		return nil, nil
	}
	return team, nil
}

func (m *mockTeamStore) ListUserTeams(_ context.Context, userID int64) ([]model.Team, error) {
	var result []model.Team
	for teamID, members := range m.members {
		if _, ok := members[userID]; ok {
			result = append(result, *m.teams[teamID])
		}
	}
	return result, nil
}

func (m *mockTeamStore) GetUserRole(_ context.Context, teamID, userID int64) (model.Role, error) {
	members, ok := m.members[teamID]
	if !ok {
		return "", nil
	}
	role, ok := members[userID]
	if !ok {
		return "", nil
	}
	return role, nil
}

func (m *mockTeamStore) IsUserMember(_ context.Context, teamID, userID int64) (bool, error) {
	members, ok := m.members[teamID]
	if !ok {
		return false, nil
	}
	_, isMember := members[userID]
	return isMember, nil
}

func (m *mockTeamStore) InviteUser(_ context.Context, teamID, userID int64, role model.Role) error {
	if m.inviteErr != nil {
		return m.inviteErr
	}
	if m.members[teamID] == nil {
		m.members[teamID] = make(map[int64]model.Role)
	}
	if _, exists := m.members[teamID][userID]; exists {
		return errors.New("user is already a member of this team")
	}
	m.members[teamID][userID] = role
	return nil
}

// mockCacheStore implements cacheStore for unit tests.
type mockCacheStore struct {
	invalidated bool
}

func (m *mockCacheStore) GetTaskList(_ context.Context, _ model.TaskFilter) ([]model.Task, bool, error) {
	return nil, false, nil // always cache miss
}

func (m *mockCacheStore) SetTaskList(_ context.Context, _ model.TaskFilter, _ []model.Task) error {
	return nil
}

func (m *mockCacheStore) InvalidateTeamTasks(_ context.Context, _ int64) error {
	m.invalidated = true
	return nil
}

// mockCommentStore implements commentStore for unit tests.
type mockCommentStore struct {
	comments map[int64][]model.TaskComment
	addErr   error
}

func newMockCommentStore() *mockCommentStore {
	return &mockCommentStore{
		comments: make(map[int64][]model.TaskComment),
	}
}

func (m *mockCommentStore) AddComment(_ context.Context, comment *model.TaskComment) error {
	if m.addErr != nil {
		return m.addErr
	}
	if comment.ID == 0 {
		comment.ID = int64(len(m.comments[comment.TaskID]) + 1)
	}
	m.comments[comment.TaskID] = append(m.comments[comment.TaskID], *comment)
	return nil
}

func (m *mockCommentStore) ListComments(_ context.Context, taskID int64) ([]model.TaskComment, error) {
	return m.comments[taskID], nil
}

// mockUserStore implements userStore for unit tests.
type mockUserStore struct {
	users map[int64]*model.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{
		users: make(map[int64]*model.User),
	}
}

func (m *mockUserStore) addUser(user *model.User) {
	if user.ID == 0 {
		user.ID = int64(len(m.users) + 1)
	}
	m.users[user.ID] = user
}

func (m *mockUserStore) Create(_ context.Context, user *model.User) error {
	if user.ID == 0 {
		user.ID = int64(len(m.users) + 1)
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserStore) GetByEmail(_ context.Context, email string) (*model.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserStore) GetByID(_ context.Context, id int64) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func int64Ptr(v int64) *int64 { return &v }

func strPtr(v string) *string { return &v }

func statusPtr(v model.TaskStatus) *model.TaskStatus { return &v }

// setupTaskTest creates a team with owner + members and returns a configured TaskService.
// Returns: service, teamStore, taskStore, cacheStore.
func setupTaskTest(t *testing.T) (*TaskService, *mockTeamStore, *mockTaskStore, *mockCacheStore) {
	t.Helper()
	team := newMockTeamStore()
	task := newMockTaskStore()
	cache := &mockCacheStore{}
	svc := NewTaskService(task, team, cache)
	return svc, team, task, cache
}

// ─── TaskService: CreateTask access rights ───────────────────────────────────

func TestCreateTask_NonMemberCannotCreate(t *testing.T) {
	svc, team, _, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	// userID 999 is not a member of team 1

	_, err := svc.CreateTask(context.Background(), model.CreateTaskInput{
		TeamID: 1,
		Title:  "Test Task",
	}, 999)

	if err == nil {
		t.Fatal("expected error for non-member creating task, got nil")
	}
	if err.Error() != "you are not a member of this team" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateTask_MemberCanCreate(t *testing.T) {
	svc, team, _, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember)

	task, err := svc.CreateTask(context.Background(), model.CreateTaskInput{
		TeamID: 1,
		Title:  "My task",
	}, 200)

	if err != nil {
		t.Fatalf("member should be able to create task: %v", err)
	}
	if task.Title != "My task" {
		t.Errorf("expected title 'My task', got %q", task.Title)
	}
	if task.CreatedBy != 200 {
		t.Errorf("expected CreatedBy=200, got %d", task.CreatedBy)
	}
}

func TestCreateTask_AssigneeMustBeTeamMember(t *testing.T) {
	svc, team, _, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember)

	// Try to assign to user 999 who is not in team 1
	_, err := svc.CreateTask(context.Background(), model.CreateTaskInput{
		TeamID:     1,
		Title:      "Task",
		AssigneeID: int64Ptr(999),
	}, 200)

	if err == nil {
		t.Fatal("expected error when assignee is not a team member")
	}
	if err.Error() != "assignee must be a member of the team" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateTask_CanAssignTeamMember(t *testing.T) {
	svc, team, _, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember)
	team.addMember(1, 300, model.RoleMember)

	task, err := svc.CreateTask(context.Background(), model.CreateTaskInput{
		TeamID:     1,
		Title:      "Task",
		AssigneeID: int64Ptr(300),
	}, 200)

	if err != nil {
		t.Fatalf("should be able to assign team member: %v", err)
	}
	if task.AssigneeID == nil || *task.AssigneeID != 300 {
		t.Errorf("expected assignee_id=300, got %v", task.AssigneeID)
	}
}

// ─── TaskService: ListTasks access rights ────────────────────────────────────

func TestListTasks_NonMemberCannotList(t *testing.T) {
	svc, team, _, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})

	_, err := svc.ListTasks(context.Background(), model.TaskFilter{TeamID: 1}, 999)

	if err == nil {
		t.Fatal("expected error for non-member listing tasks")
	}
	if err.Error() != "you are not a member of this team" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListTasks_MemberCanList(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember)
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Task 1"}

	tasks, err := svc.ListTasks(context.Background(), model.TaskFilter{TeamID: 1}, 200)

	if err != nil {
		t.Fatalf("member should be able to list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

// ─── TaskService: UpdateTask access rights ───────────────────────────────────

func TestUpdateTask_OwnerCanEditAnyTask(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100}) // 100 is owner
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Original", CreatedBy: 200, Version: 1}

	updated, err := svc.UpdateTask(context.Background(), 1, model.UpdateTaskInput{
		Title:   strPtr("Updated by owner"),
		Version: 1,
	}, 100) // owner

	if err != nil {
		t.Fatalf("owner should be able to edit any task: %v", err)
	}
	if updated.Title != "Updated by owner" {
		t.Errorf("expected title updated, got %q", updated.Title)
	}
}

func TestUpdateTask_AdminCanEditAnyTask(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 150, model.RoleAdmin) // admin
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Original", CreatedBy: 200, Version: 1}

	updated, err := svc.UpdateTask(context.Background(), 1, model.UpdateTaskInput{
		Title:   strPtr("Updated by admin"),
		Version: 1,
	}, 150) // admin

	if err != nil {
		t.Fatalf("admin should be able to edit any task: %v", err)
	}
	if updated.Title != "Updated by admin" {
		t.Errorf("expected title updated, got %q", updated.Title)
	}
}

func TestUpdateTask_CreatorCanEditOwnTask(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember)
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "My task", CreatedBy: 200, Version: 1}

	updated, err := svc.UpdateTask(context.Background(), 1, model.UpdateTaskInput{
		Title:       strPtr("Updated title"),
		Description: strPtr("Updated desc"),
		Version:     1,
	}, 200) // creator

	if err != nil {
		t.Fatalf("creator should be able to edit own task: %v", err)
	}
	if updated.Title != "Updated title" {
		t.Errorf("expected title updated, got %q", updated.Title)
	}
	if updated.Description != "Updated desc" {
		t.Errorf("expected description updated, got %q", updated.Description)
	}
}

func TestUpdateTask_AssigneeCanOnlyChangeStatus(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 300, model.RoleMember)
	task.tasks[1] = &model.Task{
		ID:         1,
		TeamID:     1,
		Title:      "Task",
		Status:     model.TaskStatusTodo,
		CreatedBy:  200,
		AssigneeID: int64Ptr(300),
		Version:    1,
	}

	// Assignee CAN change status
	_, err := svc.UpdateTask(context.Background(), 1, model.UpdateTaskInput{
		Status:  statusPtr(model.TaskStatusInProgress),
		Version: 1,
	}, 300)

	if err != nil {
		t.Fatalf("assignee should be able to change status: %v", err)
	}
}

func TestUpdateTask_AssigneeCannotChangeTitle(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 300, model.RoleMember)
	task.tasks[1] = &model.Task{
		ID:         1,
		TeamID:     1,
		Title:      "Task",
		Status:     model.TaskStatusTodo,
		CreatedBy:  200,
		AssigneeID: int64Ptr(300),
		Version:    1,
	}

	// Assignee CANNOT change title
	_, err := svc.UpdateTask(context.Background(), 1, model.UpdateTaskInput{
		Title:   strPtr("New title"),
		Version: 1,
	}, 300)

	if err == nil {
		t.Fatal("assignee should not be able to change title")
	}
	if err.Error() != "assignee can only change status" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateTask_AssigneeCannotChangeDescription(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 300, model.RoleMember)
	task.tasks[1] = &model.Task{
		ID:         1,
		TeamID:     1,
		Title:      "Task",
		CreatedBy:  200,
		AssigneeID: int64Ptr(300),
		Version:    1,
	}

	_, err := svc.UpdateTask(context.Background(), 1, model.UpdateTaskInput{
		Description: strPtr("New desc"),
		Version:     1,
	}, 300)

	if err == nil {
		t.Fatal("assignee should not be able to change description")
	}
	if err.Error() != "assignee can only change status" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateTask_AssigneeCannotReassign(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 300, model.RoleMember)
	task.tasks[1] = &model.Task{
		ID:         1,
		TeamID:     1,
		Title:      "Task",
		CreatedBy:  200,
		AssigneeID: int64Ptr(300),
		Version:    1,
	}

	_, err := svc.UpdateTask(context.Background(), 1, model.UpdateTaskInput{
		AssigneeID: int64Ptr(400),
		Version:    1,
	}, 300)

	if err == nil {
		t.Fatal("assignee should not be able to reassign task")
	}
	if err.Error() != "assignee can only change status" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateTask_MemberCannotEditOthersTask(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember) // creator
	team.addMember(1, 400, model.RoleMember) // unrelated member
	task.tasks[1] = &model.Task{
		ID:        1,
		TeamID:    1,
		Title:     "Task",
		CreatedBy: 200,
		Version:   1,
	}

	_, err := svc.UpdateTask(context.Background(), 1, model.UpdateTaskInput{
		Title:   strPtr("Hijacked"),
		Version: 1,
	}, 400) // unrelated member

	if err == nil {
		t.Fatal("member who is not creator or assignee should not be able to edit")
	}
	if err.Error() != "you do not have permission to edit this task" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateTask_NonMemberCannotEdit(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Task", CreatedBy: 200, Version: 1}

	_, err := svc.UpdateTask(context.Background(), 1, model.UpdateTaskInput{
		Title:   strPtr("Hijacked"),
		Version: 1,
	}, 999) // non-member

	if err == nil {
		t.Fatal("non-member should not be able to edit task")
	}
	if err.Error() != "you are not a member of this team" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateTask_NonExistentTask(t *testing.T) {
	svc, team, _, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})

	_, err := svc.UpdateTask(context.Background(), 999, model.UpdateTaskInput{
		Title:   strPtr("Update"),
		Version: 1,
	}, 100)

	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestUpdateTask_VersionMismatch(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Task", CreatedBy: 100, Version: 2}

	_, err := svc.UpdateTask(context.Background(), 1, model.UpdateTaskInput{
		Title:   strPtr("Update"),
		Version: 1, // stale version
	}, 100)

	if err == nil {
		t.Fatal("expected version mismatch error")
	}
	if !errors.Is(err, repository.ErrVersionMismatch) {
		t.Fatalf("expected ErrVersionMismatch, got: %v", err)
	}
}

// ─── TaskService: ListTaskHistory access rights ──────────────────────────────

func TestListTaskHistory_NonMemberCannotView(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Task", CreatedBy: 200}

	_, err := svc.ListTaskHistory(context.Background(), 1, 999)

	if err == nil {
		t.Fatal("non-member should not be able to view history")
	}
	if err.Error() != "you are not a member of this team" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListTaskHistory_MemberCanView(t *testing.T) {
	svc, team, task, _ := setupTaskTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember)
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Task", CreatedBy: 200}

	_, err := svc.ListTaskHistory(context.Background(), 1, 200)

	if err != nil {
		t.Fatalf("member should be able to view history: %v", err)
	}
}

// ─── TeamService: InviteUser access rights ───────────────────────────────────

func setupTeamTest(t *testing.T) (*TeamService, *mockTeamStore, *mockUserStore) {
	t.Helper()
	team := newMockTeamStore()
	user := newMockUserStore()
	svc := NewTeamService(team, user)
	return svc, team, user
}

func TestInviteUser_OwnerCanInvite(t *testing.T) {
	svc, team, user := setupTeamTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100}) // 100 is owner
	user.addUser(&model.User{ID: 200, Email: "new@test.com", Name: "New"})

	_, err := svc.InviteUser(context.Background(), 1, 100, model.InviteInput{
		Email: "new@test.com",
		Role:  model.RoleMember,
	})

	if err != nil {
		t.Fatalf("owner should be able to invite: %v", err)
	}
}

func TestInviteUser_AdminCanInvite(t *testing.T) {
	svc, team, user := setupTeamTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 150, model.RoleAdmin)
	user.addUser(&model.User{ID: 200, Email: "new@test.com", Name: "New"})

	_, err := svc.InviteUser(context.Background(), 1, 150, model.InviteInput{
		Email: "new@test.com",
		Role:  model.RoleMember,
	})

	if err != nil {
		t.Fatalf("admin should be able to invite: %v", err)
	}
}

func TestInviteUser_MemberCannotInvite(t *testing.T) {
	svc, team, user := setupTeamTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember)
	user.addUser(&model.User{ID: 300, Email: "new@test.com", Name: "New"})

	_, err := svc.InviteUser(context.Background(), 1, 200, model.InviteInput{
		Email: "new@test.com",
		Role:  model.RoleMember,
	})

	if err == nil {
		t.Fatal("member should not be able to invite")
	}
	if err.Error() != "only owner or admin can invite users" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInviteUser_NonMemberCannotInvite(t *testing.T) {
	svc, team, user := setupTeamTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	user.addUser(&model.User{ID: 300, Email: "new@test.com", Name: "New"})

	_, err := svc.InviteUser(context.Background(), 1, 999, model.InviteInput{
		Email: "new@test.com",
		Role:  model.RoleMember,
	})

	if err == nil {
		t.Fatal("non-member should not be able to invite")
	}
	if err.Error() != "you are not a member of this team" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInviteUser_CannotAssignOwnerRole(t *testing.T) {
	svc, team, user := setupTeamTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	user.addUser(&model.User{ID: 200, Email: "new@test.com", Name: "New"})

	_, err := svc.InviteUser(context.Background(), 1, 100, model.InviteInput{
		Email: "new@test.com",
		Role:  model.RoleOwner,
	})

	if err == nil {
		t.Fatal("should not be able to assign owner role via invitation")
	}
	if err.Error() != "cannot assign owner role via invitation" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInviteUser_AdminCannotModifyOwner(t *testing.T) {
	svc, team, user := setupTeamTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100}) // 100 is owner
	team.addMember(1, 150, model.RoleAdmin)
	// Re-invite owner as admin? The check is: admin can't modify the team's CreatedBy user.

	// Try to invite the owner (user 100) with a different role
	// First remove the existing owner membership so it doesn't hit "already a member" error
	// Actually, the check happens before the InviteUser call — it checks team.CreatedBy == user.ID
	// Let's use a different user email that maps to user 100
	// Actually, we need a user that exists with an email, and whose ID matches team.CreatedBy
	// The owner is user 100, and they're already a member. Let's test with admin trying to re-invite them
	// Actually the check happens before the actual invite, so it should error first.
	// Let me just make user 100 have an email and test.
	user.addUser(&model.User{ID: 100, Email: "owner@test.com", Name: "Owner"})

	_, err := svc.InviteUser(context.Background(), 1, 150, model.InviteInput{
		Email: "owner@test.com",
		Role:  model.RoleMember,
	})

	if err == nil {
		t.Fatal("admin should not be able to modify owner")
	}
	if err.Error() != "admin cannot modify owner" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInviteUser_TeamNotFound(t *testing.T) {
	svc, _, user := setupTeamTest(t)
	user.addUser(&model.User{ID: 200, Email: "new@test.com", Name: "New"})

	_, err := svc.InviteUser(context.Background(), 999, 100, model.InviteInput{
		Email: "new@test.com",
		Role:  model.RoleMember,
	})

	if err == nil {
		t.Fatal("expected error for non-existent team")
	}
}

func TestInviteUser_UserNotFound(t *testing.T) {
	svc, team, _ := setupTeamTest(t)
	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})

	_, err := svc.InviteUser(context.Background(), 1, 100, model.InviteInput{
		Email: "nonexistent@test.com",
		Role:  model.RoleMember,
	})

	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
	if err.Error() != "user not found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── CommentService: access rights ───────────────────────────────────────────

func setupCommentTest(t *testing.T) (*CommentService, *mockTeamStore, *mockTaskStore, *mockCommentStore) {
	t.Helper()
	team := newMockTeamStore()
	task := newMockTaskStore()
	comment := newMockCommentStore()
	svc := NewCommentService(comment, task, team)
	return svc, team, task, comment
}

func TestCreateComment_NonMemberCannotComment(t *testing.T) {
	svc, team, task, _ := setupCommentTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Task"}

	_, err := svc.CreateComment(context.Background(), 1, model.CreateCommentInput{
		Content: "Hello",
	}, 999) // non-member

	if err == nil {
		t.Fatal("non-member should not be able to comment")
	}
	if err.Error() != "you are not a member of this team" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateComment_MemberCanComment(t *testing.T) {
	svc, team, task, _ := setupCommentTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember)
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Task"}

	comment, err := svc.CreateComment(context.Background(), 1, model.CreateCommentInput{
		Content: "Hello",
	}, 200)

	if err != nil {
		t.Fatalf("member should be able to comment: %v", err)
	}
	if comment.Content != "Hello" {
		t.Errorf("expected content 'Hello', got %q", comment.Content)
	}
}

func TestListComments_NonMemberCannotView(t *testing.T) {
	svc, team, task, _ := setupCommentTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Task"}

	_, err := svc.ListComments(context.Background(), 1, 999)

	if err == nil {
		t.Fatal("non-member should not be able to view comments")
	}
	if err.Error() != "you are not a member of this team" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListComments_MemberCanView(t *testing.T) {
	svc, team, task, _ := setupCommentTest(t)

	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember)
	task.tasks[1] = &model.Task{ID: 1, TeamID: 1, Title: "Task"}

	comments, err := svc.ListComments(context.Background(), 1, 200)

	if err != nil {
		t.Fatalf("member should be able to view comments: %v", err)
	}
	if comments == nil {
		// nil is ok for empty list
	}
}

// ─── StatsService: access rights ─────────────────────────────────────────────

func TestGetTeamStats_NonMemberCannotView(t *testing.T) {
	team := newMockTeamStore()
	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})

	svc := NewStatsService(nil, team) // statsStore not needed for access check

	_, err := svc.GetTeamStats(context.Background(), 1, 999)

	if err == nil {
		t.Fatal("non-member should not be able to view stats")
	}
	if err.Error() != "you are not a member of this team" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTeamStats_MemberCannotView(t *testing.T) {
	team := newMockTeamStore()
	team.addTeam(&model.Team{ID: 1, Name: "Team A", CreatedBy: 100})
	team.addMember(1, 200, model.RoleMember)

	svc := NewStatsService(nil, team)

	_, err := svc.GetTeamStats(context.Background(), 1, 200)

	if err == nil {
		t.Fatal("member should not be able to view stats")
	}
	if err.Error() != "only owner or admin can view team statistics" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTeamStats_TeamNotFound(t *testing.T) {
	team := newMockTeamStore()

	svc := NewStatsService(nil, team)

	_, err := svc.GetTeamStats(context.Background(), 999, 100)

	if err == nil {
		t.Fatal("expected error for non-existent team")
	}
}

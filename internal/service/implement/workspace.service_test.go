package implement

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/job"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

// ------- MOCKS -------

type mockWorkspaceRepo struct {
	createFunc                   func(workspace *models.Workspace) error
	getByIDFunc                  func(id string) (*models.Workspace, error)
	updateFunc                   func(workspace *models.Workspace) error
	deleteFunc                   func(id string) error
	getWorkspaceByDomainFunc     func(domain string) (*models.Workspace, error)
	countWorkspaceByPlanFunc     func(plans []models.WorkspacePlan, role models.WorkspaceRole, userID string) ([]dto.WorkspacePlanCount, error)
	addMemberFunc                func(member *models.WorkspaceMember) error
	getMemberFunc                func(workspaceID, userID string) (*models.WorkspaceMember, error)
	listByUserIDWithSummaryFunc  func(userID string) ([]dto.WorkspaceSummary, int, error)
	getByIDWithDetailFunc        func(workspaceID string) (*dto.WorkspaceDetailResponse, error)
}

func (m *mockWorkspaceRepo) WithTx(tx *gorm.DB) repoInterface.WorkspaceRepository { return m }
func (m *mockWorkspaceRepo) Create(w *models.Workspace) error                     { return m.createFunc(w) }
func (m *mockWorkspaceRepo) GetByID(id string) (*models.Workspace, error)         { return m.getByIDFunc(id) }
func (m *mockWorkspaceRepo) Update(w *models.Workspace) error                     { return m.updateFunc(w) }
func (m *mockWorkspaceRepo) Delete(id string) error                               { return m.deleteFunc(id) }
func (m *mockWorkspaceRepo) GetWorkspaceByDomain(domain string) (*models.Workspace, error) {
	return m.getWorkspaceByDomainFunc(domain)
}
func (m *mockWorkspaceRepo) CountWorkspaceByPlan(plans []models.WorkspacePlan, role models.WorkspaceRole, userID string) ([]dto.WorkspacePlanCount, error) {
	return m.countWorkspaceByPlanFunc(plans, role, userID)
}
func (m *mockWorkspaceRepo) AddMember(member *models.WorkspaceMember) error { return m.addMemberFunc(member) }
func (m *mockWorkspaceRepo) GetMember(workspaceID, userID string) (*models.WorkspaceMember, error) {
	return m.getMemberFunc(workspaceID, userID)
}
func (m *mockWorkspaceRepo) ListByUserIDWithSummary(userID string) ([]dto.WorkspaceSummary, int, error) {
	return m.listByUserIDWithSummaryFunc(userID)
}
func (m *mockWorkspaceRepo) GetByIDWithDetail(workspaceID string) (*dto.WorkspaceDetailResponse, error) {
	return m.getByIDWithDetailFunc(workspaceID)
}

type mockRoleRepo struct {
	createFunc func(role *models.Role) (*models.Role, error)
}

func (m *mockRoleRepo) Create(role *models.Role) (*models.Role, error) { return m.createFunc(role) }
func (m *mockRoleRepo) GetByID(id string) (*models.Role, error)        { return nil, nil }
func (m *mockRoleRepo) ListByWorkspaceID(workspaceID string) ([]models.Role, error) { return nil, nil }
func (m *mockRoleRepo) Update(role *models.Role) error                              { return nil }
func (m *mockRoleRepo) Delete(id string) error                                      { return nil }
func (m *mockRoleRepo) CountByWorkspaceID(workspaceID string) (int64, error)        { return 0, nil }
func (m *mockRoleRepo) ListWithPagination(workspaceID string, search string, page int, limit int) ([]dto.RoleSummary, *dto.Pagination, error) {
	return nil, nil, nil
}
func (m *mockRoleRepo) GetByIDWithDetail(workspaceID string, roleID string) (*dto.RoleDetailResponse, error) {
	return nil, nil
}
func (m *mockRoleRepo) GetAffectedProjectsByRoleID(roleID string) ([]dto.AffectedProject, int, error) {
	return nil, 0, nil
}
func (m *mockRoleRepo) ValidateRoleIDsBelongToWorkspace(roleIDs []string, workspaceID string) ([]string, error) {
	return nil, nil
}

type mockRolePermissionRepo struct {
	bulkCreateFunc func(roleID string, permissionIDs []string) error
}

func (m *mockRolePermissionRepo) GetPermissionsByRoleID(roleID string) ([]models.RolePermission, error) {
	return nil, nil
}
func (m *mockRolePermissionRepo) BulkCreate(roleID string, permissionIDs []string) error {
	return m.bulkCreateFunc(roleID, permissionIDs)
}
func (m *mockRolePermissionRepo) BulkDelete(roleID string, permissionIDs []string) error { return nil }

type mockActivityLogRepo struct{}

func (m *mockActivityLogRepo) Create(log *models.ActivityLog) error { return nil }
func (m *mockActivityLogRepo) ListByWorkspace(workspaceID string, filters map[string][]string, limit int, cursor string) (*dto.ActivityLogListResponse, error) {
	return nil, nil
}
func (m *mockActivityLogRepo) ListByProject(projectID string, filters map[string][]string, limit int, cursor string) (*dto.ActivityLogListResponse, error) {
	return nil, nil
}
func (m *mockActivityLogRepo) ListByTask(taskID string, limit int, cursor string, direction string) (*dto.ActivityLogListResponse, error) {
	return nil, nil
}

type mockUserRepo struct {
	getByIDFunc func(id string) (*models.User, error)
}

func (m *mockUserRepo) Create(user *models.User) error                           { return nil }
func (m *mockUserRepo) GetByID(id string) (*models.User, error)                  { return m.getByIDFunc(id) }
func (m *mockUserRepo) GetByEmail(email string) (*models.User, error)            { return nil, nil }
func (m *mockUserRepo) GetByUsername(username string) (*models.User, error)      { return nil, nil }
func (m *mockUserRepo) Update(id string, user *models.User) (*models.User, error) { return nil, nil }
func (m *mockUserRepo) Delete(id string) error                                   { return nil }
func (m *mockUserRepo) AnonymizeDelete(id string, updates map[string]interface{}) error { return nil }

// ------- HELPERS ----

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}
	return db, mock
}

func newWorkspaceServiceWithMocks(
	t *testing.T,
	wsRepo *mockWorkspaceRepo,
	roleRepo *mockRoleRepo,
	rpRepo *mockRolePermissionRepo,
	userRepo *mockUserRepo,
) (*workspaceService, *gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := newMockDB(t)
	tm := database.NewTransactionManager(db)
	disp := job.NewDispatcher(db)

	svc := NewWorkspaceService(tm, wsRepo, roleRepo, rpRepo, &mockActivityLogRepo{}, userRepo, disp)
	return svc.(*workspaceService), db, mock
}

func assertAppError(t *testing.T, err error, expectedCode string) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error but got nil")
		return
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("expected *apperror.AppError, got %T: %v", err, err)
		return
	}
	if appErr.Code != expectedCode {
		t.Errorf("expected error code %q, got %q: %v", expectedCode, appErr.Code, appErr.Message)
	}
}

// ------- TESTS: CreateWorkspace -------

func TestWorkspaceService_CreateWorkspace(t *testing.T) {
	userID := "user-1"
	domain := "my-team"
	req := &dto.CreateWorkspaceRequest{Name: "My Team", Domain: &domain}

	t.Run("success", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			getWorkspaceByDomainFunc: func(d string) (*models.Workspace, error) { return nil, nil },
			countWorkspaceByPlanFunc: func(plans []models.WorkspacePlan, role models.WorkspaceRole, uid string) ([]dto.WorkspacePlanCount, error) {
				return []dto.WorkspacePlanCount{{Plan: models.WorkspacePlanFREE, Count: 0}}, nil
			},
			createFunc: func(w *models.Workspace) error { return nil },
			addMemberFunc: func(m *models.WorkspaceMember) error { return nil },
		}
		roleRepo := &mockRoleRepo{
			createFunc: func(role *models.Role) (*models.Role, error) { return role, nil },
		}
		rpRepo := &mockRolePermissionRepo{
			bulkCreateFunc: func(roleID string, permissionIDs []string) error { return nil },
		}
		userRepo := &mockUserRepo{
			getByIDFunc: func(id string) (*models.User, error) { return &models.User{FullName: "Alice"}, nil },
		}

		svc, _, mock := newWorkspaceServiceWithMocks(t, wsRepo, roleRepo, rpRepo, userRepo)

		mock.ExpectBegin()
		// logActivityInTx inside the transaction will generate an INSERT
		mock.ExpectQuery(`INSERT INTO "activity_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("log-1", time.Now()))
		mock.ExpectCommit()

		result, err := svc.CreateWorkspace(userID, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "My Team" {
			t.Errorf("expected name 'My Team', got %q", result.Name)
		}
		if result.OwnerID != userID {
			t.Errorf("expected owner %q, got %q", userID, result.OwnerID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("sqlmock expectations not met: %v", err)
		}
	})

	t.Run("fail - invalid workspace name", func(t *testing.T) {
		svc, _, _ := newWorkspaceServiceWithMocks(t, &mockWorkspaceRepo{}, &mockRoleRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{})
		_, err := svc.CreateWorkspace(userID, &dto.CreateWorkspaceRequest{Name: "A"})
		assertAppError(t, err, "VALIDATION_ERROR")
	})

	t.Run("fail - invalid domain", func(t *testing.T) {
		svc, _, _ := newWorkspaceServiceWithMocks(t, &mockWorkspaceRepo{}, &mockRoleRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{})
		_, err := svc.CreateWorkspace(userID, &dto.CreateWorkspaceRequest{Name: "My Team", Domain: strPtr("-invalid")})
		assertAppError(t, err, "VALIDATION_ERROR")
	})

	t.Run("fail - domain already taken", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			getWorkspaceByDomainFunc: func(d string) (*models.Workspace, error) {
				return &models.Workspace{ID: "other-ws"}, nil
			},
		}
		svc, _, _ := newWorkspaceServiceWithMocks(t, wsRepo, &mockRoleRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{})
		_, err := svc.CreateWorkspace(userID, req)
		assertAppError(t, err, "DOMAIN_ALREADY_TAKEN")
	})

	t.Run("fail - workspace limit reached", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			getWorkspaceByDomainFunc: func(d string) (*models.Workspace, error) { return nil, nil },
			countWorkspaceByPlanFunc: func(plans []models.WorkspacePlan, role models.WorkspaceRole, uid string) ([]dto.WorkspacePlanCount, error) {
				return []dto.WorkspacePlanCount{{Plan: models.WorkspacePlanFREE, Count: 3}}, nil
			},
		}
		svc, _, _ := newWorkspaceServiceWithMocks(t, wsRepo, &mockRoleRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{})
		_, err := svc.CreateWorkspace(userID, req)
		assertAppError(t, err, "WORKSPACE_LIMIT_REACHED")
	})

	t.Run("fail - transaction create workspace error", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			getWorkspaceByDomainFunc: func(d string) (*models.Workspace, error) { return nil, nil },
			countWorkspaceByPlanFunc: func(plans []models.WorkspacePlan, role models.WorkspaceRole, uid string) ([]dto.WorkspacePlanCount, error) {
				return []dto.WorkspacePlanCount{{Plan: models.WorkspacePlanFREE, Count: 0}}, nil
			},
			createFunc: func(w *models.Workspace) error { return errors.New("db error") },
			addMemberFunc: func(m *models.WorkspaceMember) error { return nil },
		}
		userRepo := &mockUserRepo{
			getByIDFunc: func(id string) (*models.User, error) { return &models.User{FullName: "Alice"}, nil },
		}
		svc, _, mock := newWorkspaceServiceWithMocks(t, wsRepo, &mockRoleRepo{}, &mockRolePermissionRepo{}, userRepo)

		mock.ExpectBegin()

		result, err := svc.CreateWorkspace(userID, req)
		assertAppError(t, err, "INTERNAL_ERROR")
		if result != nil {
			t.Error("expected nil result on error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("sqlmock expectations not met: %v", err)
		}
	})

	t.Run("fail - seed default roles error", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			getWorkspaceByDomainFunc: func(d string) (*models.Workspace, error) { return nil, nil },
			countWorkspaceByPlanFunc: func(plans []models.WorkspacePlan, role models.WorkspaceRole, uid string) ([]dto.WorkspacePlanCount, error) {
				return []dto.WorkspacePlanCount{{Plan: models.WorkspacePlanFREE, Count: 0}}, nil
			},
			createFunc: func(w *models.Workspace) error { return nil },
			addMemberFunc: func(m *models.WorkspaceMember) error { return nil },
		}
		roleRepo := &mockRoleRepo{
			createFunc: func(role *models.Role) (*models.Role, error) { return nil, errors.New("db error") },
		}
		userRepo := &mockUserRepo{
			getByIDFunc: func(id string) (*models.User, error) { return &models.User{FullName: "Alice"}, nil },
		}
		svc, _, mock := newWorkspaceServiceWithMocks(t, wsRepo, roleRepo, &mockRolePermissionRepo{}, userRepo)

		mock.ExpectBegin()
		// logActivityInTx inside the transaction will generate an INSERT
		mock.ExpectQuery(`INSERT INTO "activity_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("log-1", time.Now()))
		mock.ExpectCommit()

		_, err := svc.CreateWorkspace(userID, req)
		assertAppError(t, err, "INTERNAL_ERROR")
	})
}

// ------- TESTS: GetWorkspaceById -------

func TestWorkspaceService_GetWorkspaceById(t *testing.T) {
	wsID := "ws-1"
	userID := "user-1"

	t.Run("success", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			getByIDWithDetailFunc: func(id string) (*dto.WorkspaceDetailResponse, error) {
				return &dto.WorkspaceDetailResponse{ID: id, Name: "Team", Plan: "FREE"}, nil
			},
			getMemberFunc: func(wid, uid string) (*models.WorkspaceMember, error) {
				return &models.WorkspaceMember{Role: models.WorkspaceRoleMEMBER}, nil
			},
		}
		svc, _, _ := newWorkspaceServiceWithMocks(t, wsRepo, &mockRoleRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{})

		result, err := svc.GetWorkspaceById(wsID, userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != wsID {
			t.Errorf("expected id %q, got %q", wsID, result.ID)
		}
	})

	t.Run("fail - workspace not found", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			getByIDWithDetailFunc: func(id string) (*dto.WorkspaceDetailResponse, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}
		svc, _, _ := newWorkspaceServiceWithMocks(t, wsRepo, &mockRoleRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{})

		_, err := svc.GetWorkspaceById(wsID, userID)
		assertAppError(t, err, "WORKSPACE_NOT_FOUND")
	})

	t.Run("fail - internal error on get workspace", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			getByIDWithDetailFunc: func(id string) (*dto.WorkspaceDetailResponse, error) {
				return nil, errors.New("db error")
			},
		}
		svc, _, _ := newWorkspaceServiceWithMocks(t, wsRepo, &mockRoleRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{})

		_, err := svc.GetWorkspaceById(wsID, userID)
		assertAppError(t, err, "INTERNAL_ERROR")
	})

	t.Run("fail - not a member", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			getByIDWithDetailFunc: func(id string) (*dto.WorkspaceDetailResponse, error) {
				return &dto.WorkspaceDetailResponse{ID: id}, nil
			},
			getMemberFunc: func(wid, uid string) (*models.WorkspaceMember, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}
		svc, _, _ := newWorkspaceServiceWithMocks(t, wsRepo, &mockRoleRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{})

		_, err := svc.GetWorkspaceById(wsID, userID)
		assertAppError(t, err, "NOT_A_MEMBER")
	})
}

// ------- TESTS: GetWorkspacesByUserId -------

func TestWorkspaceService_GetWorkspacesByUserId(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			listByUserIDWithSummaryFunc: func(uid string) ([]dto.WorkspaceSummary, int, error) {
				return []dto.WorkspaceSummary{{ID: "ws-1", Name: "Team"}}, 1, nil
			},
		}
		svc, _, _ := newWorkspaceServiceWithMocks(t, wsRepo, &mockRoleRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{})

		result, err := svc.GetWorkspacesByUserId("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected total 1, got %d", result.Total)
		}
	})

	t.Run("fail - repo error", func(t *testing.T) {
		wsRepo := &mockWorkspaceRepo{
			listByUserIDWithSummaryFunc: func(uid string) ([]dto.WorkspaceSummary, int, error) {
				return nil, 0, errors.New("db error")
			},
		}
		svc, _, _ := newWorkspaceServiceWithMocks(t, wsRepo, &mockRoleRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{})

		_, err := svc.GetWorkspacesByUserId("user-1")
		assertAppError(t, err, "INTERNAL_ERROR")
	})
}

func strPtr(s string) *string { return &s }

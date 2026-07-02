package implement

import (
	"errors"
	"net/http"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type projectMemberService struct {
	memberRepo         repoInterface.ProjectMemberRepository
	wsMemberRepo       repoInterface.WorkspaceMemberRepository
	workspaceRepo      repoInterface.WorkspaceRepository
	projectRepo        repoInterface.ProjectRepository
	roleRepo           repoInterface.RoleRepository
}

func NewProjectMemberService(
	memberRepo repoInterface.ProjectMemberRepository,
	wsMemberRepo repoInterface.WorkspaceMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	projectRepo repoInterface.ProjectRepository,
	roleRepo repoInterface.RoleRepository,
) _interface.ProjectMemberService {
	return &projectMemberService{
		memberRepo:    memberRepo,
		wsMemberRepo:  wsMemberRepo,
		workspaceRepo: workspaceRepo,
		projectRepo:   projectRepo,
		roleRepo:      roleRepo,
	}
}

func (s *projectMemberService) getProjectOrFail(workspaceID, projectID string) (*models.Project, error) {
	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.WorkspaceID != workspaceID {
		return nil, apperror.ErrProjectNotFound
	}
	if project.DeletedAt.Valid {
		return nil, apperror.ErrProjectNotFound
	}
	return project, nil
}

func (s *projectMemberService) isWorkspaceOwner(workspaceID, userID string) (bool, error) {
	ws, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, apperror.ErrWorkspaceNotFound
		}
		return false, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get workspace")
	}
	return ws.OwnerID == userID, nil
}

func (s *projectMemberService) ListMembers(workspaceID string, userID string, projectID string, page int, limit int, search string, roleID string) ([]dto.ProjectMemberInfo, *dto.Pagination, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, nil, err
	}
	return s.memberRepo.ListWithPagination(projectID, search, roleID, page, limit)
}

func (s *projectMemberService) GetAvailableWorkspaceMembers(workspaceID string, userID string, projectID string, search string, page int, limit int) ([]dto.AvailableWorkspaceMember, *dto.Pagination, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, nil, err
	}
	return s.memberRepo.ListAvailableWorkspaceMembers(workspaceID, projectID, search, page, limit)
}

func (s *projectMemberService) AddMembersToProject(workspaceID string, userID string, projectID string, req *dto.AddMembersRequest) (*dto.AddMembersResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}
	if len(req.Members) == 0 {
		return nil, apperror.ErrMembersRequired
	}

	var userIDs []string
	var roleIDs []string
	roleIDSet := make(map[string]struct{})
	for _, m := range req.Members {
		userIDs = append(userIDs, m.UserID)
		roleIDs = append(roleIDs, m.RoleID)
		roleIDSet[m.RoleID] = struct{}{}
	}

	for _, uid := range userIDs {
		_, err := s.wsMemberRepo.GetByID(workspaceID, uid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.NewAppError(http.StatusBadRequest, "USER_NOT_IN_WORKSPACE", "User(s) not found in workspace")
			}
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify workspace membership")
		}
	}

	wsRoles, err := 	s.roleRepo.ListByWorkspaceID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get roles")
	}
	validRoleIDs := make(map[string]struct{}, len(wsRoles))
	for _, r := range wsRoles {
		validRoleIDs[r.ID] = struct{}{}
	}
	for _, rid := range roleIDs {
		if _, ok := validRoleIDs[rid]; !ok {
			return nil, apperror.NewAppError(http.StatusBadRequest, "INVALID_ROLE_ID", "One or more role IDs do not belong to this workspace")
		}
	}

	existingMembers, err := 	s.memberRepo.ListByProjectID(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get existing members")
	}
	existingSet := make(map[string]struct{}, len(existingMembers))
	for _, m := range existingMembers {
		existingSet[m.UserID] = struct{}{}
	}

	var toAdd []dto.MemberRolePair
	var skipped []string
	for _, m := range req.Members {
		if _, ok := existingSet[m.UserID]; ok {
			skipped = append(skipped, m.UserID)
		} else {
			toAdd = append(toAdd, m)
		}
	}

	if len(toAdd) == 0 {
		return &dto.AddMembersResponse{
			Added:                []dto.AddedMemberInfo{},
			SkippedAlreadyMember: skipped,
			TotalAdded:           0,
		}, nil
	}

	createdMembers, err := s.memberRepo.BulkAddMember(projectID, toAdd)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add members")
	}

	roleMap := make(map[string]string)
	for _, r := range wsRoles {
		roleMap[r.ID] = r.Name
	}

	var addedInfos []dto.AddedMemberInfo
	for _, cm := range createdMembers {
		roleName := ""
		if cm.RoleID != nil {
			roleName = roleMap[*cm.RoleID]
		}
		addedInfos = append(addedInfos, dto.AddedMemberInfo{
			UserID:   cm.UserID,
			FullName: cm.User.FullName,
			ProjectRole: &dto.RoleRef{
				ID:   *cm.RoleID,
				Name: roleName,
			},
			JoinedAt: cm.JoinedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return &dto.AddMembersResponse{
		Added:                addedInfos,
		SkippedAlreadyMember: skipped,
		TotalAdded:           len(addedInfos),
	}, nil
}

func (s *projectMemberService) UpdateMemberRole(workspaceID string, userID string, projectID string, targetUserID string, req *dto.UpdateProjectMemberRoleRequest) (*dto.UpdateProjectMemberRoleResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if targetUserID == userID {
		return nil, apperror.ErrCannotChangeOwnRole
	}

	member, err := 	s.memberRepo.GetByIDWithRelationRole(projectID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrMemberNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get member")
	}

	isOwner, err := s.isWorkspaceOwner(workspaceID, targetUserID)
	if err != nil {
		return nil, err
	}
	if isOwner {
		return nil, apperror.ErrCannotChangeWorkspaceOwnerRole
	}

	wsRoles, err := 	s.roleRepo.ListByWorkspaceID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get roles")
	}
	validRoleIDs := make(map[string]struct{}, len(wsRoles))
	for _, r := range wsRoles {
		validRoleIDs[r.ID] = struct{}{}
	}
	if _, ok := validRoleIDs[req.RoleID]; !ok {
		return nil, apperror.ErrInvalidRoleID
	}

	if member.RoleID != nil && *member.RoleID == req.RoleID {
		return nil, apperror.ErrSameRole
	}

	prevRoleRef := &dto.RoleRef{}
	if member.RoleID != nil && member.Role != nil {
		prevRoleRef = &dto.RoleRef{
			ID:   *member.RoleID,
			Name: member.Role.Name,
		}
	}

	if err := s.memberRepo.UpdateRole(projectID, targetUserID, req.RoleID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update role")
	}

	updatedMember, err := 	s.memberRepo.GetByIDWithRelationRole(projectID, targetUserID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get updated member")
	}

	currentRoleRef := &dto.RoleRef{}
	if updatedMember.RoleID != nil && updatedMember.Role != nil {
		currentRoleRef = &dto.RoleRef{
			ID:   *updatedMember.RoleID,
			Name: updatedMember.Role.Name,
		}
	}

	return &dto.UpdateProjectMemberRoleResponse{
		UserID:       member.UserID,
		FullName:     member.User.FullName,
		PreviousRole: prevRoleRef,
		CurrentRole:  currentRoleRef,
		UpdatedBy:    userID,
		UpdatedAt:    updatedMember.JoinedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *projectMemberService) RemoveMemberFromProject(workspaceID string, userID string, projectID string, targetUserID string) (*dto.RemoveProjectMemberResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if targetUserID == userID {
		return nil, apperror.ErrCannotRemoveSelf
	}

	member, err := s.memberRepo.GetByID(projectID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrMemberNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get member")
	}

	isOwner, err := s.isWorkspaceOwner(workspaceID, targetUserID)
	if err != nil {
		return nil, err
	}
	if isOwner {
		return nil, apperror.ErrCannotRemoveWorkspaceOwner
	}

	if err := s.memberRepo.Delete(projectID, targetUserID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove member")
	}

	return &dto.RemoveProjectMemberResponse{
		Message:       "Member has been removed from the project.",
		RemovedUserID: member.UserID,
	}, nil
}

func (s *projectMemberService) LeaveProject(workspaceID string, userID string, projectID string, req *dto.LeaveProjectRequest) (*dto.LeaveProjectResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if !req.Confirmation {
		return nil, apperror.ErrConfirmationRequired
	}

	_, err = s.memberRepo.GetByID(projectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotAProjectMember
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get member")
	}

	isOwner, err := s.isWorkspaceOwner(workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if isOwner {
		return nil, apperror.ErrWorkspaceOwnerCannotLeave
	}

	if err := s.memberRepo.Delete(projectID, userID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to leave project")
	}

	return &dto.LeaveProjectResponse{
		Message:       "You have left the project successfully.",
		LeftProjectID: projectID,
	}, nil
}

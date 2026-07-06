package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/activitylog"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/markdown"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/notif"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

var mentionExtractRe = regexp.MustCompile(`@([a-zA-Z0-9_]{3,30})`)

type commentService struct {
	commentRepo       repoInterface.CommentRepository
	taskRepo          repoInterface.TaskRepository
	projectRepo       repoInterface.ProjectRepository
	projectMemberRepo repoInterface.ProjectMemberRepository
	taskAssigneeRepo  repoInterface.TaskAssigneeRepository
	workspaceRepo     repoInterface.WorkspaceRepository
	notifRepo         repoInterface.NotificationRepository
	activityLogRepo   repoInterface.ActivityLogRepository
	userRepo          repoInterface.UserRepository
	dispatcher        *notif.Dispatcher
}

func NewCommentService(
	commentRepo repoInterface.CommentRepository,
	taskRepo repoInterface.TaskRepository,
	projectRepo repoInterface.ProjectRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	taskAssigneeRepo repoInterface.TaskAssigneeRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	notifRepo repoInterface.NotificationRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	userRepo repoInterface.UserRepository,
	dispatcher *notif.Dispatcher,
) _interface.CommentService {
	return &commentService{
		commentRepo:       commentRepo,
		taskRepo:          taskRepo,
		projectRepo:       projectRepo,
		projectMemberRepo: projectMemberRepo,
		taskAssigneeRepo:  taskAssigneeRepo,
		workspaceRepo:     workspaceRepo,
		notifRepo:         notifRepo,
		activityLogRepo:   activityLogRepo,
		userRepo:          userRepo,
		dispatcher:        dispatcher,
	}
}

func (s *commentService) getProjectOrFail(workspaceID, projectID string) (*models.Project, error) {
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

func (s *commentService) getTaskOrFail(projectID, taskID string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.ProjectID != projectID || task.DeletedAt.Valid {
		return nil, apperror.ErrTaskNotFound
	}
	return task, nil
}

func (s *commentService) canDeleteComment(workspaceID, projectID, userID string, comment *models.Comment) error {
	ws, err := s.workspaceRepo.GetByID(workspaceID)
	if err == nil && ws.OwnerID == userID {
		return nil
	}

	isOwner := comment.UserID != nil && *comment.UserID == userID
	if isOwner {
		hasPerm, err := s.projectMemberRepo.HasPermission(projectID, userID, "comment:delete_own")
		if err == nil && hasPerm {
			return nil
		}
	}

	hasAny, err := s.projectMemberRepo.HasPermission(projectID, userID, "comment:delete_any")
	if err == nil && hasAny {
		return nil
	}

	return apperror.ErrForbidden
}

func (s *commentService) extractAndResolveMentions(workspaceID, projectID string, content string) ([]dto.MentionUser, []string) {
	matches := mentionExtractRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var usernames []string
	for _, m := range matches {
		u := strings.ToLower(m[1])
		if !seen[u] {
			seen[u] = true
			usernames = append(usernames, u)
		}
	}

	if len(usernames) == 0 {
		return nil, nil
	}

	resolved, _ := s.commentRepo.ResolveUsernames(projectID, workspaceID, usernames)

	var mentions []dto.MentionUser
	var userIDs []string
	for _, username := range usernames {
		if m, ok := resolved[username]; ok {
			mentions = append(mentions, m)
			userIDs = append(userIDs, m.UserID)
		}
	}
	return mentions, userIDs
}

func (s *commentService) getInterestedUsers(taskID string, excludeUserID string) ([]string, error) {
	seen := make(map[string]bool)
	if excludeUserID != "" {
		seen[excludeUserID] = true
	}
	var result []string

	task, err := s.taskRepo.GetByID(taskID)
	if err == nil && task.CreatorID != nil && !seen[*task.CreatorID] {
		seen[*task.CreatorID] = true
		result = append(result, *task.CreatorID)
	}

	assignees, err := s.taskAssigneeRepo.ListByTaskID(taskID)
	if err == nil {
		for _, a := range assignees {
			if !seen[a.UserID] {
				seen[a.UserID] = true
				result = append(result, a.UserID)
			}
		}
	}

	previous, err := s.commentRepo.ListPreviousCommenters(taskID, excludeUserID)
	if err == nil {
		for _, uid := range previous {
			if !seen[uid] {
				seen[uid] = true
				result = append(result, uid)
			}
		}
	}

	return result, nil
}

func (s *commentService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
}

func (s *commentService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
	wsID := workspaceID
	uID := userID
	var metaStr *string
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		str := string(b)
		metaStr = &str
	}
	var snapStr *string
	if entitySnapshot != nil {
		b, _ := json.Marshal(entitySnapshot)
		str := string(b)
		snapStr = &str
	}
	var descPtr *string
	if description != "" {
		descPtr = &description
	}
	_ = s.activityLogRepo.Create(&models.ActivityLog{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypeCOMMENT,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return content
}

func (s *commentService) ListComments(workspaceID string, userID string, projectID string, taskID string, limit int, cursor string, direction string) (*dto.CommentListResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	_, err = s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 30
	}
	if direction != "desc" {
		direction = "asc"
	}

	result, err := s.commentRepo.ListByTaskIDWithCursor(taskID, limit, cursor, direction)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list comments")
	}
	return result, nil
}

func (s *commentService) CreateComment(workspaceID string, userID string, projectID string, taskID string, req *dto.CreateCommentRequest) (*dto.CommentCreateResponse, error) {
	content := markdown.SanitizeInput(req.Content)
	if content == "" {
		return nil, apperror.ErrContentRequired
	}
	if len([]rune(content)) > 10000 {
		return nil, apperror.ErrContentTooLong
	}

	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	task, err := s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	comment := models.Comment{
		TaskID:  taskID,
		UserID:  &userID,
		Content: content,
	}
	if err := s.commentRepo.Create(&comment); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create comment")
	}

	mentions, mentionedIDs := s.extractAndResolveMentions(workspaceID, projectID, content)
	for _, mid := range mentionedIDs {
		_ = s.commentRepo.CreateMention(&models.CommentMention{
			CommentID: comment.ID,
			UserID:    mid,
		})
	}

	mm := make([]markdown.MentionUser, len(mentions))
	for i, m := range mentions {
		mm[i] = markdown.MentionUser{UserID: m.UserID, Username: m.Username}
	}
	html := markdown.RenderToHTML(content, mm)

	author := dto.CommentAuthor{
		UserID:   userID,
		FullName: "",
		Username: "",
	}

	taskRef := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)
	actorName := s.getUserName(userID)

	var mentionedNotified []string
	for _, mid := range mentionedIDs {
		if mid != userID {
			mentionedNotified = append(mentionedNotified, mid)
		}
	}

	if len(mentionedNotified) > 0 {
		s.dispatcher.DispatchMENTIONED(&notif.MENTIONEDInput{
			ActorID:      userID,
			ActorName:    actorName,
			RecipientIDs: mentionedNotified,
			TaskRef:      taskRef,
			TaskTitle:    task.Title,
			CommentID:    comment.ID,
			CommentBody:  content,
			WorkspaceID:  workspaceID,
			ProjectID:    projectID,
			TaskID:       taskID,
		})
	}

	interested, _ := s.getInterestedUsers(taskID, userID)
	var commentedNotified []string
	for _, uid := range interested {
		found := false
		for _, mid := range mentionedNotified {
			if uid == mid {
				found = true
				break
			}
		}
		if !found {
			commentedNotified = append(commentedNotified, uid)
		}
	}

	if len(commentedNotified) > 0 {
		s.dispatcher.DispatchCOMMENTED(&notif.COMMENTEDInput{
			ActorID:      userID,
			ActorName:    actorName,
			RecipientIDs: commentedNotified,
			TaskRef:      taskRef,
			TaskTitle:    task.Title,
			CommentID:    comment.ID,
			CommentBody:  content,
			WorkspaceID:  workspaceID,
			ProjectID:    projectID,
			TaskID:       taskID,
		})
	}

	meta := activitylog.CommentCreated(comment.ID, truncateContent(content, 50), len(mentionedIDs))
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildCommentSnapshot(taskRef, task.Title)
	s.logActivity(workspaceID, projectID, userID, comment.ID, models.ActivityActionCREATE, meta, desc, snap)

	return &dto.CommentCreateResponse{
		ID:          comment.ID,
		Content:     content,
		ContentHTML: html,
		IsDeleted:   false,
		Author:      author,
		Mentions:    mentions,
		IsEdited:    false,
		CreatedAt:   comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   comment.UpdatedAt.Format(time.RFC3339),
		NotificationsSent: &dto.NotificationsSent{
			Mentioned: mentionedNotified,
			Commented: commentedNotified,
		},
	}, nil
}

func (s *commentService) UpdateComment(workspaceID string, userID string, projectID string, taskID string, commentID string, req *dto.UpdateCommentRequest) (*dto.CommentUpdateResponse, error) {
	content := markdown.SanitizeInput(req.Content)
	if content == "" {
		return nil, apperror.ErrContentRequired
	}
	if len([]rune(content)) > 10000 {
		return nil, apperror.ErrContentTooLong
	}

	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	task, err := s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrCommentNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get comment")
	}
	if comment.TaskID != taskID || comment.DeletedAt.Valid {
		return nil, apperror.ErrCommentNotFound
	}
	if comment.UserID == nil || *comment.UserID != userID {
		return nil, apperror.ErrForbidden
	}

	oldMentions, _ := s.commentRepo.GetMentionsByCommentID(commentID)
	oldMentionedIDs := make(map[string]bool)
	for _, m := range oldMentions {
		oldMentionedIDs[m.UserID] = true
	}

	_ = s.commentRepo.DeleteMentionsByCommentID(commentID)

	newMentions, newMentionedIDs := s.extractAndResolveMentions(workspaceID, projectID, content)
	for _, mid := range newMentionedIDs {
		_ = s.commentRepo.CreateMention(&models.CommentMention{
			CommentID: commentID,
			UserID:    mid,
		})
	}

	var newMentionsNotified []string
	for _, mid := range newMentionedIDs {
		if !oldMentionedIDs[mid] && mid != userID {
			newMentionsNotified = append(newMentionsNotified, mid)
		}
	}

	comment.Content = content
	comment.UpdatedAt = time.Now()
	if err := s.commentRepo.Update(comment); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update comment")
	}

	mm := make([]markdown.MentionUser, len(newMentions))
	for i, m := range newMentions {
		mm[i] = markdown.MentionUser{UserID: m.UserID, Username: m.Username}
	}
	html := markdown.RenderToHTML(content, mm)

	author := dto.CommentAuthor{
		UserID:   userID,
		FullName: "",
		Username: "",
	}

	if len(newMentionsNotified) > 0 {
		taskRef := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)
		actorName := s.getUserName(userID)
		s.dispatcher.DispatchMENTIONED(&notif.MENTIONEDInput{
			ActorID:      userID,
			ActorName:    actorName,
			RecipientIDs: newMentionsNotified,
			TaskRef:      taskRef,
			TaskTitle:    task.Title,
			CommentID:    commentID,
			CommentBody:  content,
			WorkspaceID:  workspaceID,
			ProjectID:    projectID,
			TaskID:       taskID,
		})
	}

	return &dto.CommentUpdateResponse{
		ID:                  commentID,
		Content:             content,
		ContentHTML:         html,
		IsDeleted:           false,
		Author:              author,
		Mentions:            newMentions,
		IsEdited:            true,
		CreatedAt:           comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           comment.UpdatedAt.Format(time.RFC3339),
		NewMentionsNotified: newMentionsNotified,
	}, nil
}

func (s *commentService) DeleteComment(workspaceID string, userID string, projectID string, taskID string, commentID string) (*dto.CommentDeleteResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	_, err = s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrCommentNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get comment")
	}
	if comment.TaskID != taskID || comment.DeletedAt.Valid {
		return nil, apperror.ErrCommentNotFound
	}

	if err := s.canDeleteComment(workspaceID, projectID, userID, comment); err != nil {
		return nil, err
	}

	if err := s.commentRepo.Delete(commentID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete comment")
	}

	return &dto.CommentDeleteResponse{
		ID:          commentID,
		TaskID:      taskID,
		IsDeleted:   true,
		Content:     nil,
		ContentHTML: nil,
		Author:      nil,
		Mentions:    []dto.MentionUser{},
		IsEdited:    false,
		CreatedAt:   comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   comment.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *commentService) GetMentionableUsers(workspaceID string, userID string, projectID string, taskID string, search string, limit int) (*dto.MentionableUsersResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 10
	}

	result, err := s.commentRepo.ListMentionableUsers(projectID, search, limit)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get mentionable users")
	}
	return result, nil
}

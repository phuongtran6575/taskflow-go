package implement

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

var (
	mentionExtractRe = regexp.MustCompile(`@(\w+)`)
	mentionReplaceRe = regexp.MustCompile(`@(\w+)`)
	urlReplaceRe     = regexp.MustCompile(`https?://[^\s<>"]+|www\.[^\s<>"]+`)
	httpPrefixRe     = regexp.MustCompile(`^https?://`)
)

func renderCommentHTML(content string, mentions []dto.MentionUser) string {
	mentionMap := make(map[string]string)
	for _, m := range mentions {
		mentionMap[m.Username] = m.UserID
	}
	result := content
	result = mentionReplaceRe.ReplaceAllStringFunc(result, func(match string) string {
		username := match[1:]
		if uid, ok := mentionMap[username]; ok {
			return fmt.Sprintf(`<span class="mention" data-user-id="%s">%s</span>`, uid, match)
		}
		return match
	})
	result = urlReplaceRe.ReplaceAllStringFunc(result, func(match string) string {
		href := match
		if !httpPrefixRe.MatchString(match) {
			href = "https://" + match
		}
		return fmt.Sprintf(`<a href="%s">%s</a>`, href, match)
	})
	return "<p>" + result + "</p>"
}

type commentService struct {
	commentRepo       repoInterface.CommentRepository
	taskRepo          repoInterface.TaskRepository
	projectRepo       repoInterface.ProjectRepository
	projectMemberRepo repoInterface.ProjectMemberRepository
}

func NewCommentService(
	commentRepo repoInterface.CommentRepository,
	taskRepo repoInterface.TaskRepository,
	projectRepo repoInterface.ProjectRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
) _interface.CommentService {
	return &commentService{
		commentRepo:       commentRepo,
		taskRepo:          taskRepo,
		projectRepo:       projectRepo,
		projectMemberRepo: projectMemberRepo,
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

func (s *commentService) extractAndResolveMentions(projectID string, content string) ([]dto.MentionUser, []string) {
	matches := mentionExtractRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var usernames []string
	for _, m := range matches {
		u := m[1]
		if !seen[u] {
			seen[u] = true
			usernames = append(usernames, u)
		}
	}

	if len(usernames) == 0 {
		return nil, nil
	}

	resolved, _ := s.commentRepo.ResolveUsernames(projectID, usernames)
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
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, apperror.ErrContentRequired
	}
	if len(content) > 10000 {
		return nil, apperror.ErrContentTooLong
	}

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

	comment := models.Comment{
		TaskID:  taskID,
		UserID:  &userID,
		Content: content,
	}
	if err := s.commentRepo.Create(&comment); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create comment")
	}

	mentions, mentionedIDs := s.extractAndResolveMentions(projectID, content)
	for _, mid := range mentionedIDs {
		_ = s.commentRepo.CreateMention(&models.CommentMention{
			CommentID: comment.ID,
			UserID:    mid,
		})
	}

	html := renderCommentHTML(content, mentions)

	author := dto.CommentAuthor{
		UserID:   userID,
		FullName: "",
		Username: "",
	}

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
			Mentioned: mentionedIDs,
			Commented: nil,
		},
	}, nil
}

func (s *commentService) UpdateComment(workspaceID string, userID string, projectID string, taskID string, commentID string, req *dto.UpdateCommentRequest) (*dto.CommentUpdateResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, apperror.ErrContentRequired
	}
	if len(content) > 10000 {
		return nil, apperror.ErrContentTooLong
	}

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
	if comment.UserID == nil || *comment.UserID != userID {
		return nil, apperror.ErrForbidden
	}

	oldMentions, _ := s.commentRepo.GetMentionsByCommentID(commentID)
	oldMentionedIDs := make(map[string]bool)
	for _, m := range oldMentions {
		oldMentionedIDs[m.UserID] = true
	}

	_ = s.commentRepo.DeleteMentionsByCommentID(commentID)

	newMentions, newMentionedIDs := s.extractAndResolveMentions(projectID, content)
	for _, mid := range newMentionedIDs {
		_ = s.commentRepo.CreateMention(&models.CommentMention{
			CommentID: commentID,
			UserID:    mid,
		})
	}

	var newMentionsNotified []string
	for _, mid := range newMentionedIDs {
		if !oldMentionedIDs[mid] {
			newMentionsNotified = append(newMentionsNotified, mid)
		}
	}

	comment.Content = content
	comment.UpdatedAt = time.Now()
	if err := s.commentRepo.Update(comment); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update comment")
	}

	html := renderCommentHTML(content, newMentions)

	author := dto.CommentAuthor{
		UserID:   userID,
		FullName: "",
		Username: "",
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

	isOwner := comment.UserID != nil && *comment.UserID == userID
	if !isOwner {
		return nil, apperror.ErrForbidden
	}

	deletedByOwner := !isOwner

	if err := s.commentRepo.Delete(commentID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete comment")
	}

	return &dto.CommentDeleteResponse{
		Message:          "Comment has been deleted.",
		DeletedCommentID: commentID,
		DeletedByOwner:   deletedByOwner,
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

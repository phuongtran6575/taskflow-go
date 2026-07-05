package notif

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	markdownRe = regexp.MustCompile(`[*_~` + "`" + `]|\[([^\]]*)\]\([^)]*\)`)
	htmlRe     = regexp.MustCompile(`<[^>]*>`)
	urlRe      = regexp.MustCompile(`https?://\S+`)
)

type TaskRefData struct {
	TaskRef   string
	TaskTitle string
	ActorName string
	ProjectID string
}

type CommentData struct {
	TaskRef    string
	TaskTitle  string
	ActorName  string
	Preview    string
	ProjectID  string
}

type StatusChangeData struct {
	TaskRef    string
	TaskTitle  string
	OldColumn  string
	NewColumn  string
	ActorName  string
}

type AddedToProjectData struct {
	ProjectName string
	RoleName    string
	ActorName   string
}

type AddedToWorkspaceData struct {
	WorkspaceName string
	Role          string
	UserName      string
}

type TaskDueSoonData struct {
	TaskRef          string
	TaskTitle        string
	DueDateFormatted string
}

type AnnouncementData struct {
	AnnouncementTitle   string
	AnnouncementContent string
}

func truncate(s string, maxLen int) string {
	cleaned := stripFormatting(s)
	runes := []rune(cleaned)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return cleaned
}

func stripFormatting(s string) string {
	s = markdownRe.ReplaceAllString(s, "$1")
	s = htmlRe.ReplaceAllString(s, "")
	s = urlRe.ReplaceAllString(s, "[link]")
	s = strings.TrimSpace(s)
	return s
}

func truncateTaskTitle(title string) string {
	runes := []rune(title)
	if len(runes) > 80 {
		return string(runes[:80]) + "..."
	}
	return title
}

func TrimContent(s string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 200
	}
	return truncate(s, maxLen)
}

func defaultDueText(dueDate *time.Time) string {
	if dueDate == nil {
		return "Không có deadline"
	}
	return dueDate.Format("2006-01-02 15:04")
}

func formatDueDate(dueDate time.Time) string {
	return dueDate.Format("2006-01-02 15:04 MST")
}

func BuildASSIGNEDTitle(actorName string, taskRef string, taskTitle string) string {
	return fmt.Sprintf("%s đã giao task %s: %s", actorName, taskRef, truncateTaskTitle(taskTitle))
}

func BuildASSIGNEDContent(projectName string, dueDate *time.Time) string {
	return fmt.Sprintf("Trong project %s. Hạn: %s.", projectName, defaultDueText(dueDate))
}

func BuildMENTIONEDTitle(actorName string, taskRef string, taskTitle string) string {
	return fmt.Sprintf("%s đã nhắc đến bạn trong %s: %s", actorName, taskRef, truncateTaskTitle(taskTitle))
}

func BuildMENTIONEDContent(commentPreview string) string {
	return TrimContent(commentPreview, 200)
}

func BuildCOMMENTEDTitle(actorName string, taskRef string, taskTitle string) string {
	return fmt.Sprintf("%s đã comment trong %s: %s", actorName, taskRef, truncateTaskTitle(taskTitle))
}

func BuildCOMMENTEDGroupedTitle(firstActor string, n int, taskRef string, taskTitle string) string {
	return fmt.Sprintf("%s và %d người khác đã comment trong %s: %s",
		firstActor, n, taskRef, truncateTaskTitle(taskTitle))
}

func BuildCOMMENTEDContent(commentPreview string) string {
	return TrimContent(commentPreview, 200)
}

func BuildCOMMENTEDGroupedContent(latestPreview string) string {
	return TrimContent(latestPreview, 200)
}

func BuildSTATUSCHANGEDTitle(taskRef string, taskTitle string, newColumn string) string {
	return fmt.Sprintf("%s: %s đã chuyển sang %s", taskRef, truncateTaskTitle(taskTitle), newColumn)
}

func BuildSTATUSCHANGEDContent(oldColumn string, newColumn string, actorName string) string {
	return fmt.Sprintf("Từ '%s' sang '%s' bởi %s.", oldColumn, newColumn, actorName)
}

func BuildADDEDTOPROJECTTitle(projectName string) string {
	return fmt.Sprintf("Bạn đã được thêm vào project %s", projectName)
}

func BuildADDEDTOPROJECTContent(roleName string, actorName string) string {
	return fmt.Sprintf("Vai trò: %s. Được thêm bởi %s.", roleName, actorName)
}

func BuildADDEDTOWORKSPACEForUserTitle(workspaceName string) string {
	return fmt.Sprintf("Chào mừng bạn đến với %s!", workspaceName)
}

func BuildADDEDTOWORKSPACEForUserContent(role string) string {
	return fmt.Sprintf("Bạn đã tham gia với vai trò %s.", role)
}

func BuildADDEDTOWORKSPACEForAdminTitle(userName string, workspaceName string) string {
	return fmt.Sprintf("%s đã tham gia %s", userName, workspaceName)
}

func BuildADDEDTOWORKSPACEForAdminContent(role string) string {
	return fmt.Sprintf("Vai trò: %s. Tham gia qua invite link.", role)
}

func BuildTASKDUESOONTitle(taskRef string) string {
	return fmt.Sprintf("Task %s sắp đến hạn", taskRef)
}

func BuildTASKDUESOONContent(taskTitle string, dueDate time.Time) string {
	return fmt.Sprintf("'%s' sẽ đến hạn trong 24 giờ nữa (%s).",
		truncateTaskTitle(taskTitle), formatDueDate(dueDate))
}

func BuildANNOUNCEMENTTitle(announcementTitle string) string {
	return fmt.Sprintf("[Thông báo] %s", announcementTitle)
}

func BuildANNOUNCEMENTContent(content string) string {
	if utf8.RuneCountInString(content) > 500 {
		runes := []rune(content)
		return string(runes[:500]) + "..."
	}
	return content
}

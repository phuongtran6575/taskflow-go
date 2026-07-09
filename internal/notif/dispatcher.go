package notif

import (
	"fmt"
	"strings"
	"time"

	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
)

type Dispatcher struct {
	notifRepo repoInterface.NotificationRepository
}

func NewDispatcher(notifRepo repoInterface.NotificationRepository) *Dispatcher {
	return &Dispatcher{notifRepo: notifRepo}
}

type ASSIGNEDInput struct {
	ActorID     string
	ActorName   string
	RecipientID string
	TaskRef     string
	TaskTitle   string
	ProjectName string
	DueDate     *time.Time
	WorkspaceID string
	ProjectID   string
	TaskID      string
}

func (d *Dispatcher) DispatchASSIGNED(in *ASSIGNEDInput) {
	if in.ActorID == in.RecipientID {
		return
	}
	title := BuildASSIGNEDTitle(in.ActorName, in.TaskRef, in.TaskTitle)
	content := BuildASSIGNEDContent(in.ProjectName, in.DueDate)
	refURL := BuildASSIGNEDURL(in.WorkspaceID, in.ProjectID, in.TaskID)
	d.createNotification(models.NotificationTypeASSIGNED, &in.ActorID, title, content, refURL, []string{in.RecipientID})
}

type MENTIONEDInput struct {
	ActorID      string
	ActorName    string
	RecipientIDs []string
	TaskRef      string
	TaskTitle    string
	CommentID    string
	CommentBody  string
	WorkspaceID  string
	ProjectID    string
	TaskID       string
}

func (d *Dispatcher) DispatchMENTIONED(in *MENTIONEDInput) {
	var recipients []string
	for _, rid := range in.RecipientIDs {
		if rid != in.ActorID {
			recipients = append(recipients, rid)
		}
	}
	if len(recipients) == 0 {
		return
	}
	title := BuildMENTIONEDTitle(in.ActorName, in.TaskRef, in.TaskTitle)
	content := BuildMENTIONEDContent(in.CommentBody)
	refURL := BuildMENTIONEDURL(in.WorkspaceID, in.ProjectID, in.TaskID, in.CommentID)
	d.createNotification(models.NotificationTypeMENTIONED, &in.ActorID, title, content, refURL, recipients)
}

type COMMENTEDInput struct {
	ActorID      string
	ActorName    string
	RecipientIDs []string
	TaskRef      string
	TaskTitle    string
	CommentID    string
	CommentBody  string
	WorkspaceID  string
	ProjectID    string
	TaskID       string
}

func (d *Dispatcher) DispatchCOMMENTED(in *COMMENTEDInput) {
	if len(in.RecipientIDs) == 0 {
		return
	}

	title := BuildCOMMENTEDTitle(in.ActorName, in.TaskRef, in.TaskTitle)
	content := BuildCOMMENTEDContent(in.CommentBody)
	refURL := BuildCOMMENTEDURL(in.WorkspaceID, in.ProjectID, in.TaskID, in.CommentID)
	since := time.Now().Add(-5 * time.Minute)

	var finalRecipients []string
	for _, rid := range in.RecipientIDs {
		if rid == in.ActorID {
			continue
		}

		ok, _ := d.notifRepo.IsUserProjectMember(in.ProjectID, rid)
		if !ok {
			continue
		}

		existing, err := d.notifRepo.FindUnreadCOMMENTEDByTask(in.TaskID, rid, since)
		if err == nil && existing != nil && existing.ID != "" {
			oldTitle := existing.Title
			var oldContentStr string
			if existing.Content != nil {
				oldContentStr = *existing.Content
			}

			newTitle, newContent := mergeGroupedCOMMENTED(oldTitle, oldContentStr, in.ActorName)
			_ = d.notifRepo.UpdateNotification(existing.ID, newTitle, newContent)
			continue
		}

		finalRecipients = append(finalRecipients, rid)
	}

	if len(finalRecipients) > 0 {
		d.createNotification(models.NotificationTypeCOMMENTED, &in.ActorID, title, content, refURL, finalRecipients)
	}
}

func mergeGroupedCOMMENTED(oldTitle, oldContent, newActor string) (string, string) {
	taskRef := extractTaskRef(oldTitle)
	firstActor := extractFirstActor(oldTitle)

	if firstActor == "" {
		firstActor = extractFirstActorFromUngrouped(oldTitle)
	}

	if firstActor != "" {
		return BuildCOMMENTEDGroupedTitle(firstActor, 2, taskRef, ""), oldContent
	}

	if taskRef != "" {
		return BuildCOMMENTEDGroupedTitle(newActor, 2, taskRef, ""), oldContent
	}

	return oldTitle, oldContent
}

func extractFirstActorFromUngrouped(title string) string {
	idx := strings.Index(title, " đã comment trong ")
	if idx >= 0 {
		return title[:idx]
	}
	return ""
}

func extractTaskRef(title string) string {
	idx := strings.Index(title, " đã comment trong ")
	if idx < 0 {
		return ""
	}
	rest := title[idx+len(" đã comment trong "):]
	idx2 := strings.LastIndex(rest, ":")
	if idx2 < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:idx2])
}

func extractFirstActor(title string) string {
	idx := strings.Index(title, " và ")
	if idx < 0 {
		return ""
	}
	rest := title[idx+len(" và "):]
	if strings.Contains(rest, " người khác") {
		return title[:idx]
	}
	return ""
}

type STATUSCHANGEDInput struct {
	ActorID     string
	ActorName   string
	RecipientIDs []string
	TaskRef     string
	TaskTitle   string
	OldColumn   string
	NewColumn   string
	WorkspaceID string
	ProjectID   string
	TaskID      string
}

func (d *Dispatcher) DispatchSTATUSCHANGED(in *STATUSCHANGEDInput) {
	var recipients []string
	for _, rid := range in.RecipientIDs {
		if rid != in.ActorID {
			recipients = append(recipients, rid)
		}
	}
	if len(recipients) == 0 {
		return
	}
	title := BuildSTATUSCHANGEDTitle(in.TaskRef, in.TaskTitle, in.NewColumn)
	content := BuildSTATUSCHANGEDContent(in.OldColumn, in.NewColumn, in.ActorName)
	refURL := BuildSTATUSCHANGEDURL(in.WorkspaceID, in.ProjectID, in.TaskID)
	d.createNotification(models.NotificationTypeSTATUSCHANGED, &in.ActorID, title, content, refURL, recipients)
}

type ADDEDTOPROJECTInput struct {
	ActorID     string
	ActorName   string
	RecipientID string
	ProjectName string
	RoleName    string
	WorkspaceID string
	ProjectID   string
}

func (d *Dispatcher) DispatchADDEDTOPROJECT(in *ADDEDTOPROJECTInput) {
	title := BuildADDEDTOPROJECTTitle(in.ProjectName)
	content := BuildADDEDTOPROJECTContent(in.RoleName, in.ActorName)
	refURL := BuildADDEDTOPROJECTURL(in.WorkspaceID, in.ProjectID)
	d.createNotification(models.NotificationTypeADDEDTOPROJECT, &in.ActorID, title, content, refURL, []string{in.RecipientID})
}

type ADDEDTOWORKSPACEForUserInput struct {
	RecipientID   string
	WorkspaceName string
	Role          string
	WorkspaceID   string
}

func (d *Dispatcher) DispatchADDEDTOWORKSPACEForUser(in *ADDEDTOWORKSPACEForUserInput) {
	title := BuildADDEDTOWORKSPACEForUserTitle(in.WorkspaceName)
	content := BuildADDEDTOWORKSPACEForUserContent(in.Role)
	refURL := BuildADDEDTOWORKSPACEURL(in.WorkspaceID)
	d.createNotification(models.NotificationTypeADDEDTOWORKSPACE, nil, title, content, refURL, []string{in.RecipientID})
}

type ADDEDTOWORKSPACEForAdminInput struct {
	AdminIDs       []string
	UserName       string
	WorkspaceName  string
	Role           string
	WorkspaceID    string
	CreatedByName  string // BR-INV-07: Người tạo invite link
}

func (d *Dispatcher) DispatchADDEDTOWORKSPACEForAdmin(in *ADDEDTOWORKSPACEForAdminInput) {
	if len(in.AdminIDs) == 0 {
		return
	}
	title := BuildADDEDTOWORKSPACEForAdminTitle(in.UserName, in.WorkspaceName)
	content := BuildADDEDTOWORKSPACEForAdminContent(in.Role, in.CreatedByName)
	refURL := BuildADDEDTOWORKSPACEURL(in.WorkspaceID)
	d.createNotification(models.NotificationTypeADDEDTOWORKSPACE, nil, title, content, refURL, in.AdminIDs)
}

type TASKDUESOONInput struct {
	RecipientIDs []string
	TaskRef      string
	TaskTitle    string
	DueDate      time.Time
	WorkspaceID  string
	ProjectID    string
	TaskID       string
}

func (d *Dispatcher) DispatchTASKDUESOON(in *TASKDUESOONInput) {
	if len(in.RecipientIDs) == 0 {
		return
	}
	title := BuildTASKDUESOONTitle(in.TaskRef)
	content := BuildTASKDUESOONContent(in.TaskTitle, in.DueDate)
	refURL := BuildTASKDUESOONURL(in.WorkspaceID, in.ProjectID, in.TaskID)
	d.createNotification(models.NotificationTypeTASKDUESOON, nil, title, content, refURL, in.RecipientIDs)
}

func (d *Dispatcher) createNotification(notifType models.NotificationType, actorID *string, title, content, refURL string, recipients []string) {
	if len(recipients) == 0 {
		return
	}
	now := time.Now()
	notif := &models.Notification{
		ActorID:      actorID,
		Type:         notifType,
		Title:        title,
		Content:      &content,
		ReferenceURL: &refURL,
		CreatedAt:    now,
	}
	_ = d.notifRepo.Create(notif, recipients)
}

func dedupRecipients(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func (d *Dispatcher) FormatTaskRef(projectKey string, taskNumber int) string {
	return fmt.Sprintf("%s-%d", projectKey, taskNumber)
}

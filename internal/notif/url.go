package notif

import "fmt"

func BuildASSIGNEDURL(workspaceID, projectID, taskID string) string {
	return fmt.Sprintf("/workspaces/%s/projects/%s/tasks/%s", workspaceID, projectID, taskID)
}

func BuildMENTIONEDURL(workspaceID, projectID, taskID, commentID string) string {
	return fmt.Sprintf("/workspaces/%s/projects/%s/tasks/%s#comment-%s",
		workspaceID, projectID, taskID, commentID)
}

func BuildCOMMENTEDURL(workspaceID, projectID, taskID, commentID string) string {
	return fmt.Sprintf("/workspaces/%s/projects/%s/tasks/%s#comment-%s",
		workspaceID, projectID, taskID, commentID)
}

func BuildSTATUSCHANGEDURL(workspaceID, projectID, taskID string) string {
	return fmt.Sprintf("/workspaces/%s/projects/%s/tasks/%s", workspaceID, projectID, taskID)
}

func BuildADDEDTOPROJECTURL(workspaceID, projectID string) string {
	return fmt.Sprintf("/workspaces/%s/projects/%s", workspaceID, projectID)
}

func BuildADDEDTOWORKSPACEURL(workspaceID string) string {
	return fmt.Sprintf("/workspaces/%s", workspaceID)
}

func BuildTASKDUESOONURL(workspaceID, projectID, taskID string) string {
	return fmt.Sprintf("/workspaces/%s/projects/%s/tasks/%s", workspaceID, projectID, taskID)
}

func BuildANNOUNCEMENTURL(workspaceID, notificationID string) string {
	return fmt.Sprintf("/workspaces/%s/notifications/%s", workspaceID, notificationID)
}

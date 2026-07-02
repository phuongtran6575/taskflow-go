package _interface

import "TaskFlow-Go/internal/dto"

type SessionService interface {
	GetSessionsByUserId(userID string) (*dto.SessionListResponse, error)
	RevokeSession(userID string, sessionID string) error
	RevokeAllOtherSessions(userID string) (*dto.RevokeAllSessionsResponse, error)
}

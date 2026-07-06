package implement

import (
	"strings"

	"TaskFlow-Go/internal/dto"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type sessionService struct {
	sessionRepo repoInterface.SessionRepository
}

func NewSessionService(sessionRepo repoInterface.SessionRepository) _interface.SessionService {
	return &sessionService{sessionRepo: sessionRepo}
}

func maskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return parts[0] + "." + parts[1] + ".x.x"
}

func (s *sessionService) GetSessionsByUserId(userID string) (*dto.SessionListResponse, error) {
	sessions, err := s.sessionRepo.GetByUserID(userID)
	if err != nil {
		return nil, apperror.ErrSessionNotFound
	}

	var currentSessionID string
	var sessionInfos []dto.SessionInfo

	for _, sess := range sessions {
		if !sess.IsRevoked {
			if currentSessionID == "" {
				currentSessionID = sess.ID
			}
			info := dto.SessionInfo{
				ID:        sess.ID,
				CreatedAt: sess.CreatedAt,
				ExpiresAt: sess.ExpiresAt,
				UserAgent: sess.UserAgent,
				IPAddress: maskIP(sess.IPAddress),
				IsCurrent: false,
			}
			sessionInfos = append(sessionInfos, info)
		}
	}

	for i := range sessionInfos {
		if sessionInfos[i].ID == currentSessionID {
			sessionInfos[i].IsCurrent = true
			break
		}
	}

	return &dto.SessionListResponse{
		CurrentSessionID: currentSessionID,
		Sessions:         sessionInfos,
	}, nil
}

func (s *sessionService) RevokeSession(userID string, sessionID string) error {
	session, err := s.sessionRepo.GetCurrentSession(userID, sessionID)
	if err != nil {
		return apperror.ErrSessionNotFound
	}
	if session.ID == sessionID && !session.IsRevoked {
		sessions, _ := s.sessionRepo.GetByUserID(userID)
		if len(sessions) > 0 && sessions[0].ID == sessionID {
			return apperror.ErrCannotRevokeCurrentSession
		}
	}
	return s.sessionRepo.RevokeByID(sessionID)
}

func (s *sessionService) RevokeAllOtherSessions(userID string) (*dto.RevokeAllSessionsResponse, error) {
	sessions, err := s.sessionRepo.GetByUserID(userID)
	if err != nil {
		return nil, apperror.ErrSessionNotFound
	}

	var currentSessionID string
	for _, sess := range sessions {
		if !sess.IsRevoked {
			currentSessionID = sess.ID
			break
		}
	}

	count, err := s.sessionRepo.RevokeAllExcept(userID, currentSessionID)
	if err != nil {
		return nil, apperror.ErrUploadFailed
	}

	return &dto.RevokeAllSessionsResponse{
		Message:      "All other sessions have been logged out.",
		RevokedCount: int(count),
	}, nil
}

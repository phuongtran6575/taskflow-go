package implement

import (
	"TaskFlow-Go/internal/dto"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"errors"
)

type sessionService struct {
	userRepo repoInterface.UserRepository
}

func NewSessionService(userRepo repoInterface.UserRepository) _interface.SessionService {
	return &sessionService{userRepo: userRepo}
}

func (s *sessionService) GetSessionsByUserId(userID string) (*dto.SessionListResponse, error) {
	_ = s.userRepo
	return nil, errors.New("not implemented")
}

func (s *sessionService) RevokeSession(userID string, sessionID string) error {
	return errors.New("not implemented")
}

func (s *sessionService) RevokeAllOtherSessions(userID string) (*dto.RevokeAllSessionsResponse, error) {
	return nil, errors.New("not implemented")
}

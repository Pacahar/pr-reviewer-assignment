package storage

import (
	"context"

	"github.com/pacahar/pr-reviewer-assignment/internal/models"
)

type Storage struct {
	UserStorage        UserStorage
	TeamStorage        TeamStorage
	PullRequestStorage PullRequestStorage
}

type UserStorage interface {
	CreateUser(ctx context.Context, userID, username, teamName string) error
	GetUserByID(ctx context.Context, userID string) (models.User, error)
	SetUserActiveStatus(ctx context.Context, userID string, isActive bool) error
	SetUserTeam(ctx context.Context, userID, teamName string) error
	GetActiveUsersByTeam(ctx context.Context, teamName string) ([]models.User, error)
}

type TeamStorage interface {
	CreateTeam(ctx context.Context, teamName string) error
	GetTeamByName(ctx context.Context, teamName string) (models.Team, error)
	GetUsersByTeam(ctx context.Context, teamName string) ([]models.User, error)
}

type PullRequestStorage interface {
	CreatePullRequest(ctx context.Context, prID, prName, authorID string) error
	GetPullRequestByID(ctx context.Context, prID string) (models.PullRequest, error)
	SetPullRequestStatus(ctx context.Context, prID, status string) error
	AddReviewer(ctx context.Context, prID, userID string) error
	RemoveReviewer(ctx context.Context, prID, userID string) error
	GetReviewersByPR(ctx context.Context, prID string) ([]string, error)
	GetPullRequestsByReviewer(ctx context.Context, reviewerID string) ([]models.PullRequestShort, error)
}

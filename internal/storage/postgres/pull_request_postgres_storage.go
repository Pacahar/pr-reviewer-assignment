package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/pacahar/pr-reviewer-assignment/internal/models"
	storageErrors "github.com/pacahar/pr-reviewer-assignment/internal/storage/errors"
)

type PullRequestPostgresStorage struct {
	db *sql.DB
}

func (prs *PullRequestPostgresStorage) CreatePullRequest(ctx context.Context, prID, prName, authorID string) error {
	_, err := prs.db.ExecContext(ctx, `
		INSERT INTO pull_requests
		(pull_request_id, pull_request_name, author_id, status) 
		values ($1, $2, $3, $4);`,
		prID,
		prName,
		authorID,
		"OPEN",
	)
	return err
}

func (prs *PullRequestPostgresStorage) GetPullRequestByID(ctx context.Context, prID string) (models.PullRequest, error) {
	var pr models.PullRequest
	var createdAt, mergedAt sql.NullTime

	err := prs.db.QueryRowContext(ctx, `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1;`,
		prID,
	).Scan(
		&pr.PullRequestID,
		&pr.PullRequestName,
		&pr.AuthorID,
		&pr.Status,
		&createdAt,
		&mergedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PullRequest{}, storageErrors.ErrPRNotFound
		}
		return models.PullRequest{}, err
	}

	if createdAt.Valid {
		t := createdAt.Time
		pr.CreatedAt = &t
	}

	if mergedAt.Valid {
		t := mergedAt.Time
		pr.MergedAt = &t
	}

	reviewers, err := prs.GetReviewersByPR(ctx, prID)
	if err != nil {
		return models.PullRequest{}, err
	}

	pr.AssignedReviewers = reviewers

	return pr, nil
}

func (prs *PullRequestPostgresStorage) SetPullRequestStatus(ctx context.Context, prID, status string, now time.Time) error {
	// prs.GetPullRequestByID(ctx, prID)

	var mergedAt *time.Time
	if status == "MERGED" {
		mergedAt = &now
	}

	_, err := prs.db.ExecContext(ctx, `
		UPDATE pull_requests
		SET status = $1,
		    merged_at = $2
		WHERE pull_request_id = $3;`,
		status,
		mergedAt,
		prID,
	)
	return err
}

func (prs *PullRequestPostgresStorage) AddReviewer(ctx context.Context, prID, userID string) error {
	_, err := prs.db.ExecContext(ctx, `
		INSERT INTO pr_reviewers (pull_request_id, reviewer_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING;`,
		prID,
		userID,
	)
	return err
}

func (prs *PullRequestPostgresStorage) RemoveReviewer(ctx context.Context, prID, userID string) error {
	_, err := prs.db.ExecContext(ctx, `
		DELETE FROM pr_reviewers
		WHERE pull_request_id = $1 AND reviewer_id = $2;`,
		prID,
		userID,
	)
	return err
}

func (prs *PullRequestPostgresStorage) GetReviewersByPR(ctx context.Context, prID string) ([]string, error) {
	rows, err := prs.db.QueryContext(ctx, `
		SELECT reviewer_id
		FROM pr_reviewers
		WHERE pull_request_id = $1;`,
		prID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string

	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		reviewers = append(reviewers, id)
	}

	return reviewers, nil
}

func (prs *PullRequestPostgresStorage) GetPullRequestsByReviewer(ctx context.Context, reviewerID string) ([]models.PullRequestShort, error) {
	rows, err := prs.db.QueryContext(ctx, `
		SELECT pr.pull_request_id,
		       pr.pull_request_name,
		       pr.author_id,
		       pr.status
		FROM pull_requests pr
		JOIN pr_reviewers r
		    ON pr.pull_request_id = r.pull_request_id
		WHERE r.reviewer_id = $1;`,
		reviewerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.PullRequestShort

	for rows.Next() {
		var pr models.PullRequestShort

		err := rows.Scan(
			&pr.PullRequestID,
			&pr.PullRequestName,
			&pr.AuthorID,
			&pr.Status,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, pr)
	}

	return result, nil
}

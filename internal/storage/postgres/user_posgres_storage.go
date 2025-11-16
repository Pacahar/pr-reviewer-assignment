package postgres

import (
	"context"
	"database/sql"

	"github.com/pacahar/pr-reviewer-assignment/internal/models"
)

type UserPostgresStorage struct {
	db *sql.DB
}

func (us *UserPostgresStorage) CreateUser(ctx context.Context, userID, username, teamName string) error {
	_, err := us.db.ExecContext(ctx, `
		INSERT INTO users (user_id, username, team_name)
		VALUES ($1, $2, $3);`,
		userID,
		username,
		teamName,
	)
	return err
}

func (us *UserPostgresStorage) GetUserByID(ctx context.Context, userID string) (models.User, error) {
	var user models.User

	err := us.db.QueryRowContext(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id = $1;`,
		userID,
	).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)

	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (us *UserPostgresStorage) SetUserActiveStatus(ctx context.Context, userID string, isActive bool) error {
	_, err := us.db.ExecContext(ctx, `
		UPDATE users
		SET is_active = $1
		WHERE user_id = $2;`,
		isActive,
		userID,
	)
	return err
}

func (us *UserPostgresStorage) SetUserTeam(ctx context.Context, userID, teamName string) error {
	_, err := us.db.ExecContext(ctx, `
		UPDATE users
		SET team_name = $1
		WHERE user_id = $2;
	`,
		teamName,
		userID,
	)
	return err
}

func (us *UserPostgresStorage) GetActiveUsersByTeam(ctx context.Context, teamName string) ([]models.User, error) {
	rows, err := us.db.QueryContext(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name = $1 AND is_active = TRUE;`,
		teamName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User

	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

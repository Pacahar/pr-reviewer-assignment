package postgres

import (
	"context"
	"database/sql"

	"github.com/pacahar/pr-reviewer-assignment/internal/models"
)

type TeamPostgresStorage struct {
	db *sql.DB
}

func (ts *TeamPostgresStorage) CreateTeam(ctx context.Context, teamName string) error {
	_, err := ts.db.ExecContext(ctx, `
		INSERT INTO teams (team_name)
		VALUES ($1)
		ON CONFLICT (team_name) DO NOTHING;`,
		teamName,
	)

	return err
}

func (ts *TeamPostgresStorage) GetTeamByName(ctx context.Context, teamName string) (models.Team, error) {
	var team models.Team

	err := ts.db.QueryRowContext(ctx,
		`SELECT team_name FROM teams WHERE team_name = $1;`,
		teamName,
	).Scan(&team.TeamName)

	if err != nil {
		return models.Team{}, err
	}

	rows, err := ts.db.QueryContext(ctx, `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_name = $1;`,
		teamName,
	)
	if err != nil {
		return models.Team{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var u models.TeamMember
		if err := rows.Scan(&u.UserID, &u.Username, &u.IsActive); err != nil {
			return models.Team{}, err
		}
		team.Members = append(team.Members, u)
	}

	return team, nil
}

func (ts *TeamPostgresStorage) GetUsersByTeam(ctx context.Context, teamName string) ([]models.User, error) {

	rows, err := ts.db.QueryContext(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name = $1;`,
		teamName,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.User

	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.UserID, &u.Username, &u.TeamName, &u.IsActive)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}

	return result, nil
}

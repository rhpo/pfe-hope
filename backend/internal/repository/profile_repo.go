package repository

import (
	"database/sql"

	"pfe-backend/internal/entity"
	"pfe-backend/internal/shared/convert"
)

type ProfileRepository struct {
	db *sql.DB
}

func NewProfileRepository(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) FindByEmail(email string) (*entity.Profile, error) {
	query := `SELECT id, role, full_name, email, avatar_url, is_active, created_at, updated_at
		FROM profiles WHERE email = ?`
	row := r.db.QueryRow(query, email)

	var avatarURL sql.NullString
	profile := &entity.Profile{}
	err := row.Scan(
		&profile.ID,
		&profile.Role,
		&profile.FullName,
		&profile.Email,
		&avatarURL,
		&profile.IsActive,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	profile.AvatarURL = convert.StringPtr(avatarURL)
	return profile, nil
}

func (r *ProfileRepository) FindByID(id int64) (*entity.Profile, error) {
	query := `SELECT id, role, full_name, email, avatar_url, is_active, created_at, updated_at
		FROM profiles WHERE id = ?`
	row := r.db.QueryRow(query, id)

	var avatarURL sql.NullString
	profile := &entity.Profile{}
	err := row.Scan(
		&profile.ID,
		&profile.Role,
		&profile.FullName,
		&profile.Email,
		&avatarURL,
		&profile.IsActive,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	profile.AvatarURL = convert.StringPtr(avatarURL)
	return profile, nil
}

func (r *ProfileRepository) FindAll() ([]*entity.Profile, error) {
	query := `SELECT id, role, full_name, email, avatar_url, is_active, created_at, updated_at
		FROM profiles ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*entity.Profile
	for rows.Next() {
		var avatarURL sql.NullString
		profile := &entity.Profile{}
		err := rows.Scan(
			&profile.ID,
			&profile.Role,
			&profile.FullName,
			&profile.Email,
			&avatarURL,
			&profile.IsActive,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		profile.AvatarURL = convert.StringPtr(avatarURL)
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func (r *ProfileRepository) Insert(p *entity.Profile) error {
	query := `INSERT INTO profiles (role, full_name, email, avatar_url, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	result, err := r.db.Exec(query, p.Role, p.FullName, p.Email, convert.NullString(p.AvatarURL), p.IsActive)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return nil
}

func (r *ProfileRepository) Update(p *entity.Profile) error {
	query := `UPDATE profiles SET role = ?, full_name = ?, email = ?, avatar_url = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, p.Role, p.FullName, p.Email, convert.NullString(p.AvatarURL), p.IsActive, p.ID)
	return err
}

func (r *ProfileRepository) UpdateAvatarURL(id int64, url string) error {
	_, err := r.db.Exec(`UPDATE profiles SET avatar_url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, url, id)
	return err
}

func (r *ProfileRepository) Delete(id int64) error {
	query := `DELETE FROM profiles WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

package repository

import (
	"database/sql"
	"pfe-backend/internal/entity"
)

type PfeSubjectRepository struct {
	db *sql.DB
}

func NewPfeSubjectRepository(db *sql.DB) *PfeSubjectRepository {
	return &PfeSubjectRepository{db: db}
}

func (r *PfeSubjectRepository) FindByID(id int64) (*entity.PfeSubject, error) {
	query := `SELECT id, title, description, group_type, proposer_id, proposer_role, company_id, academic_year_id,
		validator1_id, validator2_id, validator1_decision, validator2_decision,
		validator1_comment, validator2_comment, status, co_supervisor_id, pre_assigned_student_ids, created_at, updated_at
		FROM pfe_subjects WHERE id = ?`
	row := r.db.QueryRow(query, id)
	s := &entity.PfeSubject{}
	err := row.Scan(
		&s.ID, &s.Title, &s.Description, &s.GroupType, &s.ProposerID, &s.ProposerRole, &s.CompanyID, &s.AcademicYearID,
		&s.Validator1ID, &s.Validator2ID, &s.Validator1Decision, &s.Validator2Decision,
		&s.Validator1Comment, &s.Validator2Comment, &s.Status, &s.CoSupervisorID, &s.PreAssignedStudentIDs, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

func (r *PfeSubjectRepository) FindAll() ([]*entity.PfeSubject, error) {
	query := `SELECT id, title, description, group_type, proposer_id, proposer_role, company_id, academic_year_id,
		validator1_id, validator2_id, validator1_decision, validator2_decision,
		validator1_comment, validator2_comment, status, co_supervisor_id, pre_assigned_student_ids, created_at, updated_at
		FROM pfe_subjects ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSubjects(rows)
}

func (r *PfeSubjectRepository) FindByProposer(proposerID int64) ([]*entity.PfeSubject, error) {
	query := `SELECT id, title, description, group_type, proposer_id, proposer_role, company_id, academic_year_id,
		validator1_id, validator2_id, validator1_decision, validator2_decision,
		validator1_comment, validator2_comment, status, co_supervisor_id, pre_assigned_student_ids, created_at, updated_at
		FROM pfe_subjects WHERE proposer_id = ? ORDER BY created_at DESC`
	rows, err := r.db.Query(query, proposerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSubjects(rows)
}

func (r *PfeSubjectRepository) FindByStatus(status string) ([]*entity.PfeSubject, error) {
	query := `SELECT id, title, description, group_type, proposer_id, proposer_role, company_id, academic_year_id,
		validator1_id, validator2_id, validator1_decision, validator2_decision,
		validator1_comment, validator2_comment, status, co_supervisor_id, pre_assigned_student_ids, created_at, updated_at
		FROM pfe_subjects WHERE status = ? ORDER BY created_at DESC`
	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSubjects(rows)
}

func (r *PfeSubjectRepository) FindPendingValidation(validatorID int64) ([]*entity.PfeSubject, error) {

	query := `SELECT id, title, description, group_type, proposer_id, proposer_role, company_id, academic_year_id,
		validator1_id, validator2_id, validator1_decision, validator2_decision,
		validator1_comment, validator2_comment, status, co_supervisor_id, pre_assigned_student_ids, created_at, updated_at
		FROM pfe_subjects
		WHERE status = 'en_attente'
		  AND (
		    (validator1_id = ? AND validator1_decision IS NULL) OR
		    (validator2_id = ? AND validator2_decision IS NULL)
		  )
		ORDER BY created_at DESC`
	rows, err := r.db.Query(query, validatorID, validatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSubjects(rows)
}

func (r *PfeSubjectRepository) FindByAcademicYear(academicYearID int64) ([]*entity.PfeSubject, error) {
	query := `SELECT id, title, description, group_type, proposer_id, proposer_role, company_id, academic_year_id,
		validator1_id, validator2_id, validator1_decision, validator2_decision,
		validator1_comment, validator2_comment, status, co_supervisor_id, pre_assigned_student_ids, created_at, updated_at
		FROM pfe_subjects WHERE academic_year_id = ? ORDER BY created_at DESC`
	rows, err := r.db.Query(query, academicYearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSubjects(rows)
}

func (r *PfeSubjectRepository) FindByCompany(companyID int64) ([]*entity.PfeSubject, error) {
	query := `SELECT id, title, description, group_type, proposer_id, proposer_role, company_id, academic_year_id,
		validator1_id, validator2_id, validator1_decision, validator2_decision,
		validator1_comment, validator2_comment, status, co_supervisor_id, pre_assigned_student_ids, created_at, updated_at
		FROM pfe_subjects WHERE company_id = ? ORDER BY created_at DESC`
	rows, err := r.db.Query(query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSubjects(rows)
}

func (r *PfeSubjectRepository) FindAvailable(academicYearID int64) ([]*entity.PfeSubject, error) {
	query := `SELECT id, title, description, group_type, proposer_id, proposer_role, company_id, academic_year_id,
		validator1_id, validator2_id, validator1_decision, validator2_decision,
		validator1_comment, validator2_comment, status, co_supervisor_id, pre_assigned_student_ids, created_at, updated_at
		FROM pfe_subjects WHERE status = 'valide' AND academic_year_id = ? ORDER BY created_at DESC`
	rows, err := r.db.Query(query, academicYearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSubjects(rows)
}

func (r *PfeSubjectRepository) Insert(s *entity.PfeSubject) error {
	query := `INSERT INTO pfe_subjects (title, description, group_type, proposer_id, proposer_role, company_id, academic_year_id,
		validator1_id, validator2_id, validator1_decision, validator2_decision,
		validator1_comment, validator2_comment, status, co_supervisor_id, pre_assigned_student_ids)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.Exec(query, s.Title, s.Description, s.GroupType, s.ProposerID, s.ProposerRole, s.CompanyID,
		s.AcademicYearID, s.Validator1ID, s.Validator2ID, s.Validator1Decision, s.Validator2Decision,
		s.Validator1Comment, s.Validator2Comment, s.Status, s.CoSupervisorID, s.PreAssignedStudentIDs)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	s.ID = id
	return nil
}

func (r *PfeSubjectRepository) Update(s *entity.PfeSubject) error {
	query := `UPDATE pfe_subjects SET title = ?, description = ?, group_type = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, s.Title, s.Description, s.GroupType, s.ID)
	return err
}

func (r *PfeSubjectRepository) Resubmit(id int64, title, description, groupType string) error {
	query := `UPDATE pfe_subjects SET
		title = ?, description = ?, group_type = ?,
		status = 'en_attente',
		validator1_id = NULL, validator2_id = NULL,
		validator1_decision = NULL, validator2_decision = NULL,
		validator1_comment = NULL, validator2_comment = NULL,
		updated_at = CURRENT_TIMESTAMP
	WHERE id = ?`
	_, err := r.db.Exec(query, title, description, groupType, id)
	return err
}

func (r *PfeSubjectRepository) UpdateStatus(id int64, status string) error {
	_, err := r.db.Exec(`UPDATE pfe_subjects SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	return err
}

func (r *PfeSubjectRepository) UpdateValidation(id int64, validatorField, decision, comment string) error {
	var query string
	if validatorField == "validator1" {
		query = `UPDATE pfe_subjects SET validator1_decision = ?, validator1_comment = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	} else {
		query = `UPDATE pfe_subjects SET validator2_decision = ?, validator2_comment = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	}
	_, err := r.db.Exec(query, decision, comment, id)
	return err
}

func (r *PfeSubjectRepository) AssignValidators(id, validator1ID, validator2ID int64) error {
	_, err := r.db.Exec(`UPDATE pfe_subjects
		SET validator1_id = ?, validator2_id = ?,
		    validator1_decision = NULL, validator2_decision = NULL,
		    validator1_comment = NULL, validator2_comment = NULL,
		    status = 'en_attente',
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		validator1ID, validator2ID, id)
	return err
}

func (r *PfeSubjectRepository) AssignCoSupervisor(id, coSupervisorID int64) error {
	_, err := r.db.Exec(`UPDATE pfe_subjects SET co_supervisor_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		coSupervisorID, id)
	return err
}

func (r *PfeSubjectRepository) AddDomain(subjectID, domainID int64) error {
	_, err := r.db.Exec(`INSERT OR IGNORE INTO subject_domains (subject_id, domain_id) VALUES (?, ?)`, subjectID, domainID)
	return err
}

func (r *PfeSubjectRepository) RemoveDomain(subjectID, domainID int64) error {
	_, err := r.db.Exec(`DELETE FROM subject_domains WHERE subject_id = ? AND domain_id = ?`, subjectID, domainID)
	return err
}

func (r *PfeSubjectRepository) GetDomains(subjectID int64) ([]*entity.Domain, error) {
	query := `SELECT d.id, d.name, d.created_at, d.updated_at
		FROM domains d INNER JOIN subject_domains sd ON sd.domain_id = d.id WHERE sd.subject_id = ?`
	rows, err := r.db.Query(query, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var domains []*entity.Domain
	for rows.Next() {
		d := &entity.Domain{}
		if err := rows.Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

func (r *PfeSubjectRepository) scanSubjects(rows *sql.Rows) ([]*entity.PfeSubject, error) {
	var subjects []*entity.PfeSubject
	for rows.Next() {
		s := &entity.PfeSubject{}
		if err := rows.Scan(
			&s.ID, &s.Title, &s.Description, &s.GroupType, &s.ProposerID, &s.ProposerRole, &s.CompanyID, &s.AcademicYearID,
			&s.Validator1ID, &s.Validator2ID, &s.Validator1Decision, &s.Validator2Decision,
			&s.Validator1Comment, &s.Validator2Comment, &s.Status, &s.CoSupervisorID, &s.PreAssignedStudentIDs, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subjects = append(subjects, s)
	}
	return subjects, rows.Err()
}

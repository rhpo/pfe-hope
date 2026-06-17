package repository

import (
	"database/sql"
	"time"

	"pfe-backend/internal/entity"
)

type PfeAssignmentRepository struct {
	db *sql.DB
}

func NewPfeAssignmentRepository(db *sql.DB) *PfeAssignmentRepository {
	return &PfeAssignmentRepository{db: db}
}

func (r *PfeAssignmentRepository) FindByID(id int64) (*entity.PfeAssignment, error) {
	query := `SELECT id, pfe_code, subject_id, academic_year_id, student_id, student2_id, student3_id,
		supervisor_id, co_supervisor_id, memoire_url, status, created_at, updated_at
		FROM pfe_assignments WHERE id = ?`
	row := r.db.QueryRow(query, id)
	a := &entity.PfeAssignment{}
	err := row.Scan(&a.ID, &a.PfeCode, &a.SubjectID, &a.AcademicYearID, &a.StudentID, &a.Student2ID, &a.Student3ID,
		&a.SupervisorID, &a.CoSupervisorID, &a.MemoireURL, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

func (r *PfeAssignmentRepository) FindByStudent(studentID, academicYearID int64) (*entity.PfeAssignment, error) {
	query := `SELECT id, pfe_code, subject_id, academic_year_id, student_id, student2_id, student3_id,
		supervisor_id, co_supervisor_id, memoire_url, status, created_at, updated_at
		FROM pfe_assignments WHERE (student_id = ? OR student2_id = ? OR student3_id = ?) AND academic_year_id = ? LIMIT 1`
	row := r.db.QueryRow(query, studentID, studentID, studentID, academicYearID)
	a := &entity.PfeAssignment{}
	err := row.Scan(&a.ID, &a.PfeCode, &a.SubjectID, &a.AcademicYearID, &a.StudentID, &a.Student2ID, &a.Student3ID,
		&a.SupervisorID, &a.CoSupervisorID, &a.MemoireURL, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

func (r *PfeAssignmentRepository) FindBySupervisor(supervisorID int64) ([]*entity.PfeAssignment, error) {
	query := `SELECT id, pfe_code, subject_id, academic_year_id, student_id, student2_id, student3_id,
		supervisor_id, co_supervisor_id, memoire_url, status, created_at, updated_at
		FROM pfe_assignments WHERE supervisor_id = ? OR co_supervisor_id = ? ORDER BY created_at DESC`
	rows, err := r.db.Query(query, supervisorID, supervisorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanAssignments(rows)
}

type MonthlyTimelineStat struct {
	Label         string
	WithSubject   int
	MemoireSubmit int
}

func (r *PfeAssignmentRepository) MonthlyTimelineStats(months int) ([]MonthlyTimelineStat, error) {
	results := make([]MonthlyTimelineStat, months)
	now := time.Now()

	for i := months - 1; i >= 0; i-- {

		target := now.AddDate(0, -i, 0)

		endOfMonth := time.Date(target.Year(), target.Month()+1, 1, 0, 0, 0, 0, time.UTC)

		monthNames := []string{"Jan", "Fév", "Mar", "Avr", "Mai", "Juin", "Juil", "Aoû", "Sep", "Oct", "Nov", "Déc"}
		label := monthNames[target.Month()-1]

		var withSubject, memoireSubmit int
		_ = r.db.QueryRow(
			`SELECT COUNT(*), COALESCE(SUM(CASE WHEN memoire_url IS NOT NULL THEN 1 ELSE 0 END), 0)
			 FROM pfe_assignments WHERE created_at < ?`,
			endOfMonth.Format("2006-01-02T15:04:05"),
		).Scan(&withSubject, &memoireSubmit)

		results[months-1-i] = MonthlyTimelineStat{
			Label:         label,
			WithSubject:   withSubject,
			MemoireSubmit: memoireSubmit,
		}
	}
	return results, nil
}

func (r *PfeAssignmentRepository) FindBySubjectID(subjectID int64) (*entity.PfeAssignment, error) {
	query := `SELECT id, pfe_code, subject_id, academic_year_id, student_id, student2_id, student3_id,
		supervisor_id, co_supervisor_id, memoire_url, status, created_at, updated_at
		FROM pfe_assignments WHERE subject_id = ? LIMIT 1`
	row := r.db.QueryRow(query, subjectID)
	a := &entity.PfeAssignment{}
	err := row.Scan(&a.ID, &a.PfeCode, &a.SubjectID, &a.AcademicYearID, &a.StudentID, &a.Student2ID, &a.Student3ID,
		&a.SupervisorID, &a.CoSupervisorID, &a.MemoireURL, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

func (r *PfeAssignmentRepository) FindAll() ([]*entity.PfeAssignment, error) {
	rows, err := r.db.Query(`SELECT id, pfe_code, subject_id, academic_year_id, student_id, student2_id, student3_id,
		supervisor_id, co_supervisor_id, memoire_url, status, created_at, updated_at
		FROM pfe_assignments ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanAssignments(rows)
}

func (r *PfeAssignmentRepository) FindByAcademicYear(academicYearID int64) ([]*entity.PfeAssignment, error) {
	query := `SELECT id, pfe_code, subject_id, academic_year_id, student_id, student2_id, student3_id,
		supervisor_id, co_supervisor_id, memoire_url, status, created_at, updated_at
		FROM pfe_assignments WHERE academic_year_id = ? ORDER BY created_at DESC`
	rows, err := r.db.Query(query, academicYearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanAssignments(rows)
}

func (r *PfeAssignmentRepository) FindByCompanySubject(companyID int64) ([]*entity.PfeAssignment, error) {
	query := `SELECT pa.id, pa.pfe_code, pa.subject_id, pa.academic_year_id, pa.student_id, pa.student2_id, pa.student3_id,
		pa.supervisor_id, pa.co_supervisor_id, pa.memoire_url, pa.status, pa.created_at, pa.updated_at
		FROM pfe_assignments pa
		INNER JOIN pfe_subjects ps ON ps.id = pa.subject_id
		WHERE ps.company_id = ? OR (ps.proposer_id = ? AND ps.proposer_role = 'company')
		ORDER BY pa.created_at DESC`
	rows, err := r.db.Query(query, companyID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanAssignments(rows)
}

func (r *PfeAssignmentRepository) Insert(a *entity.PfeAssignment) error {
	query := `INSERT INTO pfe_assignments (pfe_code, subject_id, academic_year_id, student_id, student2_id, student3_id,
		supervisor_id, co_supervisor_id, memoire_url, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.Exec(query, a.PfeCode, a.SubjectID, a.AcademicYearID, a.StudentID, a.Student2ID, a.Student3ID,
		a.SupervisorID, a.CoSupervisorID, a.MemoireURL, a.Status)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = id
	return nil
}

func (r *PfeAssignmentRepository) UpdateStatus(id int64, status string) error {
	_, err := r.db.Exec(`UPDATE pfe_assignments SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	return err
}

func (r *PfeAssignmentRepository) UpdateCoSupervisor(id int64, teacherID int64) error {
	_, err := r.db.Exec(`UPDATE pfe_assignments SET co_supervisor_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, teacherID, id)
	return err
}

func (r *PfeAssignmentRepository) RemoveCoSupervisor(id int64) error {
	_, err := r.db.Exec(`UPDATE pfe_assignments SET co_supervisor_id = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

func (r *PfeAssignmentRepository) UpdateMemoire(id int64, memoireURL string) error {
	_, err := r.db.Exec(`UPDATE pfe_assignments SET memoire_url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, memoireURL, id)
	return err
}

func (r *PfeAssignmentRepository) CountBySpecialityAndYear(academicYearID int64, specialityCode string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM pfe_assignments pa
		JOIN students s ON s.id = pa.student_id
		JOIN specialities sp ON sp.id = s.speciality_id
		WHERE pa.academic_year_id = ? AND sp.code = ?
	`, academicYearID, specialityCode).Scan(&count)
	return count, err
}

func (r *PfeAssignmentRepository) Update(a *entity.PfeAssignment) error {
	query := `UPDATE pfe_assignments SET pfe_code = ?, subject_id = ?, student_id = ?, student2_id = ?, student3_id = ?,
		supervisor_id = ?, co_supervisor_id = ?, memoire_url = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, a.PfeCode, a.SubjectID, a.StudentID, a.Student2ID, a.Student3ID,
		a.SupervisorID, a.CoSupervisorID, a.MemoireURL, a.Status, a.ID)
	return err
}

func (r *PfeAssignmentRepository) scanAssignments(rows *sql.Rows) ([]*entity.PfeAssignment, error) {
	var assignments []*entity.PfeAssignment
	for rows.Next() {
		a := &entity.PfeAssignment{}
		if err := rows.Scan(&a.ID, &a.PfeCode, &a.SubjectID, &a.AcademicYearID, &a.StudentID, &a.Student2ID, &a.Student3ID,
			&a.SupervisorID, &a.CoSupervisorID, &a.MemoireURL, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

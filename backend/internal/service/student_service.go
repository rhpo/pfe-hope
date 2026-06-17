package service

import (
	"pfe-backend/internal/entity"
	"pfe-backend/internal/repository"
	"pfe-backend/internal/shared/apperror"
)

type StudentService struct {
	profileRepo       *repository.ProfileRepository
	studentRepo       *repository.StudentRepository
	teacherRepo       *repository.TeacherRepository
	companyRepo       *repository.CompanyRepository
	specialityRepo    *repository.SpecialityRepository
	pfeSubjectRepo    *repository.PfeSubjectRepository
	wishRepo          *repository.WishRepository
	pfeAssignmentRepo *repository.PfeAssignmentRepository
	progressRepo      *repository.ProgressReportRepository
	defenseRepo       *repository.DefenseRepository
	defenseJuryRepo   *repository.DefenseJuryRepository
	supEvalRepo       *repository.SupervisorEvaluationRepository
	notificationRepo  *repository.NotificationRepository
	academicYearRepo  *repository.AcademicYearRepository
}

func NewStudentService(
	profileRepo *repository.ProfileRepository,
	studentRepo *repository.StudentRepository,
	teacherRepo *repository.TeacherRepository,
	companyRepo *repository.CompanyRepository,
	specialityRepo *repository.SpecialityRepository,
	pfeSubjectRepo *repository.PfeSubjectRepository,
	wishRepo *repository.WishRepository,
	pfeAssignmentRepo *repository.PfeAssignmentRepository,
	progressRepo *repository.ProgressReportRepository,
	defenseRepo *repository.DefenseRepository,
	defenseJuryRepo *repository.DefenseJuryRepository,
	supEvalRepo *repository.SupervisorEvaluationRepository,
	notificationRepo *repository.NotificationRepository,
	academicYearRepo *repository.AcademicYearRepository,
) *StudentService {
	return &StudentService{
		profileRepo:       profileRepo,
		studentRepo:       studentRepo,
		teacherRepo:       teacherRepo,
		companyRepo:       companyRepo,
		specialityRepo:    specialityRepo,
		pfeSubjectRepo:    pfeSubjectRepo,
		wishRepo:          wishRepo,
		pfeAssignmentRepo: pfeAssignmentRepo,
		progressRepo:      progressRepo,
		defenseRepo:       defenseRepo,
		defenseJuryRepo:   defenseJuryRepo,
		supEvalRepo:       supEvalRepo,
		notificationRepo:  notificationRepo,
		academicYearRepo:  academicYearRepo,
	}
}

func (s *StudentService) getActiveAcademicYear() (int64, error) {
	year, err := s.academicYearRepo.FindActive()
	if err != nil {
		return 0, err
	}
	if year == nil {
		return 0, apperror.BadRequest("Les soumissions de vœux ne sont pas encore ouvertes. Contactez l'administration.")
	}
	return year.ID, nil
}

func (s *StudentService) getStudent(profileID int64) (*entity.Student, error) {
	st, err := s.studentRepo.FindByProfileID(profileID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, apperror.NotFound("Profil étudiant introuvable")
	}
	return st, nil
}

func (s *StudentService) GetSettings() (map[string]any, error) {
	year, err := s.academicYearRepo.FindActive()
	if err != nil || year == nil {

		return map[string]any{
			"max_wishes":          5,
			"submission_open_at":  nil,
			"submission_close_at": nil,
		}, nil
	}
	return map[string]any{
		"max_wishes":          year.MaxWishes,
		"submission_open_at":  year.SubmissionOpenAt,
		"submission_close_at": year.SubmissionCloseAt,
	}, nil
}

func (s *StudentService) Dashboard(userID int64) (map[string]any, error) {
	academicYearID, err := s.getActiveAcademicYear()
	if err != nil {

		return map[string]any{"wishes_count": 0, "has_pfe": false}, nil
	}

	st, err := s.getStudent(userID)
	if err != nil {
		return nil, err
	}
	studentID := st.ID

	assignment, _ := s.pfeAssignmentRepo.FindByStudent(studentID, academicYearID)
	wishes, _ := s.wishRepo.FindByStudent(studentID, academicYearID)

	result := map[string]any{
		"wishes_count": len(wishes),
	}
	if assignment != nil {
		result["has_pfe"] = true
		result["pfe_status"] = assignment.Status
	} else {
		result["has_pfe"] = false
	}
	return result, nil
}

func (s *StudentService) ListCatalogue() ([]*entity.PfeSubject, error) {
	subjects, err := s.pfeSubjectRepo.FindByStatus("valide")
	if err != nil {
		return nil, err
	}
	for _, sub := range subjects {
		s.hydrateSubject(sub)
		a, _ := s.pfeAssignmentRepo.FindBySubjectID(sub.ID)
		sub.IsAssigned = a != nil
	}
	return subjects, nil
}

func (s *StudentService) GetCatalogueSubject(id int64) (*entity.PfeSubject, error) {
	sub, err := s.pfeSubjectRepo.FindByID(id)
	if err != nil || sub == nil {
		return sub, err
	}
	s.hydrateSubject(sub)
	a, _ := s.pfeAssignmentRepo.FindBySubjectID(sub.ID)
	sub.IsAssigned = a != nil
	return sub, nil
}

func (s *StudentService) ListWishes(userID int64) ([]*entity.Wish, error) {
	academicYearID, err := s.getActiveAcademicYear()
	if err != nil {

		return []*entity.Wish{}, nil
	}
	st, err := s.getStudent(userID)
	if err != nil {
		return nil, err
	}
	wishes, err := s.wishRepo.FindByStudent(st.ID, academicYearID)
	if err != nil {
		return nil, err
	}
	for _, w := range wishes {
		sub, _ := s.pfeSubjectRepo.FindByID(w.SubjectID)
		if sub != nil {
			s.hydrateSubject(sub)
			w.Subject = sub
		}
	}
	return wishes, nil
}

func (s *StudentService) CreateWish(userID, subjectID int64) error {

	subject, err := s.pfeSubjectRepo.FindByID(subjectID)
	if err != nil {
		return err
	}
	if subject == nil {
		return apperror.NotFound("Sujet introuvable")
	}
	if subject.Status != "valide" {
		return apperror.BadRequest("Ce sujet n'est pas disponible")
	}

	academicYearID, err := s.getActiveAcademicYear()
	if err != nil {
		return err
	}

	st, err := s.getStudent(userID)
	if err != nil {
		return err
	}
	studentID := st.ID

	wishes, err := s.wishRepo.FindByStudent(studentID, academicYearID)
	if err != nil {
		return err
	}
	for _, w := range wishes {
		if w.SubjectID == subjectID {
			return apperror.Conflict("Vous avez déjà un voeu pour ce sujet")
		}
	}

	wish := &entity.Wish{
		StudentID:      studentID,
		SubjectID:      subjectID,
		AcademicYearID: academicYearID,
		Status:         "en_attente",
	}
	return s.wishRepo.Insert(wish)
}

func (s *StudentService) DeleteWish(userID, wishID int64) error {
	wish, err := s.wishRepo.FindByID(wishID)
	if err != nil {
		return err
	}
	if wish == nil {
		return apperror.NotFound("Voeu introuvable")
	}
	st, err := s.getStudent(userID)
	if err != nil {
		return err
	}
	if wish.StudentID != st.ID {
		return apperror.Forbidden("Accès non autorisé à ce voeu")
	}
	return s.wishRepo.Delete(wishID)
}

func (s *StudentService) GetMyPFE(userID int64) (*entity.PfeAssignment, error) {
	academicYearID, err := s.getActiveAcademicYear()
	if err != nil {
		return nil, err
	}
	st, err := s.getStudent(userID)
	if err != nil {
		return nil, err
	}
	a, err := s.pfeAssignmentRepo.FindByStudent(st.ID, academicYearID)
	if err != nil || a == nil {
		return a, err
	}
	s.hydrateAssignment(a)
	return a, nil
}

func (s *StudentService) ListMyMeetings(assignmentID int64) ([]*entity.PfeProgressReport, error) {
	return s.progressRepo.FindByAssignment(assignmentID)
}

func (s *StudentService) AddMyMeeting(userID int64, report *entity.PfeProgressReport) error {
	academicYearID, err := s.getActiveAcademicYear()
	if err != nil {
		return err
	}
	st, err := s.getStudent(userID)
	if err != nil {
		return err
	}
	assignment, err := s.pfeAssignmentRepo.FindByStudent(st.ID, academicYearID)
	if err != nil {
		return err
	}
	if assignment == nil {
		return apperror.NotFound("aucun PFE assigné")
	}
	report.AssignmentID = assignment.ID
	if report.Status == "" {
		report.Status = "en_cours"
	}
	return s.progressRepo.Insert(report)
}

func (s *StudentService) UpdateMyMeeting(userID, meetingID int64, status string) error {
	validStatuses := map[string]bool{"a_faire": true, "en_cours": true, "termine": true}
	if !validStatuses[status] {
		return apperror.BadRequest("Statut invalide (a_faire, en_cours, termine)")
	}
	st, err := s.getStudent(userID)
	if err != nil {
		return err
	}
	academicYearID, err := s.getActiveAcademicYear()
	if err != nil {
		return err
	}
	assignment, err := s.pfeAssignmentRepo.FindByStudent(st.ID, academicYearID)
	if err != nil {
		return err
	}
	if assignment == nil {
		return apperror.NotFound("aucun PFE assigné")
	}
	report, err := s.progressRepo.FindByID(meetingID)
	if err != nil {
		return err
	}
	if report == nil || report.AssignmentID != assignment.ID {
		return apperror.NotFound("entrée de suivi introuvable")
	}
	report.Status = status
	return s.progressRepo.Update(report)
}

func (s *StudentService) SubmitMemoire(assignmentID int64, memoireURL string) error {
	return s.pfeAssignmentRepo.UpdateMemoire(assignmentID, memoireURL)
}

func (s *StudentService) GetSoutenance(userID int64) (map[string]any, error) {
	academicYearID, err := s.getActiveAcademicYear()
	if err != nil {
		return nil, err
	}
	st, err := s.getStudent(userID)
	if err != nil {
		return nil, err
	}
	assignment, err := s.pfeAssignmentRepo.FindByStudent(st.ID, academicYearID)
	if err != nil {
		return nil, err
	}
	if assignment == nil {
		return nil, apperror.NotFound("Aucun PFE assigné")
	}

	defense, err := s.defenseRepo.FindByAssignment(assignment.ID)
	if err != nil {
		return nil, err
	}
	if defense == nil {
		return map[string]any{"has_soutenance": false}, nil
	}

	var jury *entity.DefenseJury
	if defense.JuryID != 0 {
		jury, _ = s.defenseJuryRepo.FindByID(defense.JuryID)
		if jury != nil {
			if t, _ := s.teacherRepo.FindByID(jury.PresidentID); t != nil {
				t.Profile, _ = s.profileRepo.FindByID(t.ProfileID)
				jury.President = t
			}
			if t, _ := s.teacherRepo.FindByID(jury.MemberID); t != nil {
				t.Profile, _ = s.profileRepo.FindByID(t.ProfileID)
				jury.Member = t
			}
		}
	}

	supEval, _ := s.supEvalRepo.FindByAssignment(assignment.ID)

	return map[string]any{
		"has_soutenance":  true,
		"defense":         defense,
		"jury":            jury,
		"supervisor_note": supEval,
	}, nil
}

func (s *StudentService) ListNotifications(userID int64) ([]*entity.Notification, error) {
	return s.notificationRepo.FindByRecipient(userID)
}

func (s *StudentService) GetTeacherProfileID(teacherID int64) (int64, error) {
	t, err := s.teacherRepo.FindByID(teacherID)
	if err != nil {
		return 0, err
	}
	if t == nil {
		return 0, apperror.NotFound("Enseignant introuvable")
	}
	return t.ProfileID, nil
}

func (s *StudentService) GetSubjectProposerID(subjectID int64) (int64, string) {
	sub, err := s.pfeSubjectRepo.FindByID(subjectID)
	if err != nil || sub == nil {
		return 0, ""
	}
	return sub.ProposerID, sub.ProposerRole
}

func (s *StudentService) hydrateTeacher(id int64) *entity.Teacher {
	if id == 0 {
		return nil
	}
	t, _ := s.teacherRepo.FindByID(id)
	if t == nil {
		t, _ = s.teacherRepo.FindByProfileID(id)
	}
	if t != nil {
		t.Profile, _ = s.profileRepo.FindByID(t.ProfileID)
	}
	return t
}

func (s *StudentService) hydrateStudent(id int64) *entity.Student {
	if id == 0 {
		return nil
	}
	st, _ := s.studentRepo.FindByID(id)
	if st == nil {
		st, _ = s.studentRepo.FindByProfileID(id)
	}
	if st != nil {
		st.Profile, _ = s.profileRepo.FindByID(st.ProfileID)
		if st.SpecialityID != nil {
			st.Speciality, _ = s.specialityRepo.FindByID(*st.SpecialityID)
		}
	}
	return st
}

func (s *StudentService) hydrateSubject(sub *entity.PfeSubject) {
	sub.Proposer, _ = s.profileRepo.FindByID(sub.ProposerID)
	if sub.CompanyID.Valid {
		sub.Company, _ = s.companyRepo.FindByID(sub.CompanyID.Int64)
		if sub.Company == nil {
			sub.Company, _ = s.companyRepo.FindByProfileID(sub.CompanyID.Int64)
		}
	}
	if sub.Validator1ID.Valid {
		sub.Validator1 = s.hydrateTeacher(sub.Validator1ID.Int64)
	}
	if sub.Validator2ID.Valid {
		sub.Validator2 = s.hydrateTeacher(sub.Validator2ID.Int64)
	}
	if sub.CoSupervisorID.Valid {
		sub.CoSupervisor = s.hydrateTeacher(sub.CoSupervisorID.Int64)
	}
	sub.Domains, _ = s.pfeSubjectRepo.GetDomains(sub.ID)
}

func (s *StudentService) hydrateAssignment(a *entity.PfeAssignment) {
	sub, _ := s.pfeSubjectRepo.FindByID(a.SubjectID)
	if sub != nil {
		s.hydrateSubject(sub)
		a.Subject = sub
	}
	a.Student = s.hydrateStudent(a.StudentID)
	if a.Student2ID.Valid {
		a.Student2 = s.hydrateStudent(a.Student2ID.Int64)
	}
	if a.Student3ID.Valid {
		a.Student3 = s.hydrateStudent(a.Student3ID.Int64)
	}
	a.Supervisor = s.hydrateTeacher(a.SupervisorID)
	if a.CoSupervisorID.Valid {
		a.CoSupervisor = s.hydrateTeacher(a.CoSupervisorID.Int64)
	}
	ay, _ := s.academicYearRepo.FindByID(a.AcademicYearID)
	a.AcademicYear = ay
}

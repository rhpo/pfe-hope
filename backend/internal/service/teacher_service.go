package service

import (
	"database/sql"
	"fmt"
	"time"

	"pfe-backend/internal/entity"
	"pfe-backend/internal/repository"
	"pfe-backend/internal/shared/apperror"
	"pfe-backend/internal/shared/notify"
	pfe_code "pfe-backend/internal/shared/pfe_code"
)

type TeacherService struct {
	profileRepo       *repository.ProfileRepository
	teacherRepo       *repository.TeacherRepository
	studentRepo       *repository.StudentRepository
	companyRepo       *repository.CompanyRepository
	specialityRepo    *repository.SpecialityRepository
	pfeSubjectRepo    *repository.PfeSubjectRepository
	wishRepo          *repository.WishRepository
	pfeAssignmentRepo *repository.PfeAssignmentRepository
	progressRepo      *repository.ProgressReportRepository
	defenseJuryRepo   *repository.DefenseJuryRepository
	defenseRepo       *repository.DefenseRepository
	juryGradeRepo     *repository.JuryGradeRepository
	supEvalRepo       *repository.SupervisorEvaluationRepository
	notificationRepo  *repository.NotificationRepository
	academicYearRepo  *repository.AcademicYearRepository
	notifier          *notify.Notifier
}

func NewTeacherService(
	profileRepo *repository.ProfileRepository,
	teacherRepo *repository.TeacherRepository,
	studentRepo *repository.StudentRepository,
	companyRepo *repository.CompanyRepository,
	specialityRepo *repository.SpecialityRepository,
	pfeSubjectRepo *repository.PfeSubjectRepository,
	wishRepo *repository.WishRepository,
	pfeAssignmentRepo *repository.PfeAssignmentRepository,
	progressRepo *repository.ProgressReportRepository,
	defenseJuryRepo *repository.DefenseJuryRepository,
	defenseRepo *repository.DefenseRepository,
	juryGradeRepo *repository.JuryGradeRepository,
	supEvalRepo *repository.SupervisorEvaluationRepository,
	notificationRepo *repository.NotificationRepository,
	academicYearRepo *repository.AcademicYearRepository,
	notifier *notify.Notifier,
) *TeacherService {
	return &TeacherService{
		profileRepo:       profileRepo,
		teacherRepo:       teacherRepo,
		studentRepo:       studentRepo,
		companyRepo:       companyRepo,
		specialityRepo:    specialityRepo,
		pfeSubjectRepo:    pfeSubjectRepo,
		wishRepo:          wishRepo,
		pfeAssignmentRepo: pfeAssignmentRepo,
		progressRepo:      progressRepo,
		defenseJuryRepo:   defenseJuryRepo,
		defenseRepo:       defenseRepo,
		juryGradeRepo:     juryGradeRepo,
		supEvalRepo:       supEvalRepo,
		notificationRepo:  notificationRepo,
		academicYearRepo:  academicYearRepo,
		notifier:          notifier,
	}
}

func (s *TeacherService) Dashboard(userID int64) (map[string]any, error) {
	subjects, _ := s.pfeSubjectRepo.FindByProposer(userID)
	supervised, _ := s.pfeAssignmentRepo.FindBySupervisor(userID)

	return map[string]any{
		"proposed_subjects": len(subjects),
		"supervised_pfes":   len(supervised),
	}, nil
}

func (s *TeacherService) ListProposedSubjects(userID int64) ([]*entity.PfeSubject, error) {
	subjects, err := s.pfeSubjectRepo.FindByProposer(userID)
	if err != nil {
		return nil, err
	}
	for _, sub := range subjects {
		s.hydrateSubject(sub)
	}
	return subjects, nil
}

func (s *TeacherService) CreateProposedSubject(subject *entity.PfeSubject, domainIDs []int64) error {
	if err := s.pfeSubjectRepo.Insert(subject); err != nil {
		return err
	}
	for _, domainID := range domainIDs {
		if err := s.pfeSubjectRepo.AddDomain(subject.ID, domainID); err != nil {
			return err
		}
	}
	return nil
}

func (s *TeacherService) GetProposedSubject(userID, id int64) (*entity.PfeSubject, error) {
	subject, err := s.pfeSubjectRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, nil
	}
	if subject.ProposerID != userID {
		return nil, apperror.Forbidden("Vous n'êtes pas l'auteur de ce sujet")
	}
	return subject, nil
}

func (s *TeacherService) UpdateProposedSubject(userID int64, subject *entity.PfeSubject) error {
	existing, err := s.pfeSubjectRepo.FindByID(subject.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return apperror.NotFound("Sujet introuvable")
	}
	if existing.ProposerID != userID {
		return apperror.Forbidden("Vous n'êtes pas l'auteur de ce sujet")
	}
	if subject.Title != "" {
		existing.Title = subject.Title
	}
	if subject.Description != "" {
		existing.Description = subject.Description
	}
	if subject.GroupType != "" {
		existing.GroupType = subject.GroupType
	}
	return s.pfeSubjectRepo.Update(existing)
}

func (s *TeacherService) ResubmitSubject(userID, subjectID int64, title, description, groupType string, domainIDs []int64) error {
	existing, err := s.pfeSubjectRepo.FindByID(subjectID)
	if err != nil {
		return err
	}
	if existing == nil {
		return apperror.NotFound("Sujet introuvable")
	}
	if existing.ProposerID != userID {
		return apperror.Forbidden("Vous n'êtes pas l'auteur de ce sujet")
	}
	if existing.Status != "accepte_sous_reserve" && existing.Status != "refuse" {
		return apperror.BadRequest("Seuls les sujets « acceptés sous réserve » ou « refusés » peuvent être resoumis")
	}
	if title == "" {
		title = existing.Title
	}
	if description == "" {
		description = existing.Description
	}
	if groupType == "" {
		groupType = existing.GroupType
	}
	if err := s.pfeSubjectRepo.Resubmit(subjectID, title, description, groupType); err != nil {
		return err
	}

	if domainIDs != nil {
		if err := s.pfeSubjectRepo.RemoveDomain(subjectID, 0); err == nil {
			_ = err
		}

		_ = s.syncDomains(subjectID, domainIDs)
	}
	return nil
}

func (s *TeacherService) syncDomains(subjectID int64, domainIDs []int64) error {

	if _, err := s.pfeSubjectRepo.GetDomains(subjectID); err == nil {

		existing, _ := s.pfeSubjectRepo.GetDomains(subjectID)
		for _, d := range existing {
			_ = s.pfeSubjectRepo.RemoveDomain(subjectID, d.ID)
		}
	}
	for _, domainID := range domainIDs {
		_ = s.pfeSubjectRepo.AddDomain(subjectID, domainID)
	}
	return nil
}

func (s *TeacherService) ListCandidats(subjectID int64) ([]*entity.Wish, error) {
	wishes, err := s.wishRepo.FindBySubject(subjectID)
	if err != nil {
		return nil, err
	}
	for _, w := range wishes {
		s.hydrateWish(w)
	}
	return wishes, nil
}

func (s *TeacherService) AcceptCandidats(subjectID int64, studentIDs []int64) (*entity.PfeAssignment, error) {
	if len(studentIDs) == 0 {
		return nil, apperror.BadRequest("Aucun étudiant sélectionné")
	}

	subject, err := s.pfeSubjectRepo.FindByID(subjectID)
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, apperror.NotFound("Sujet introuvable")
	}

	existing, err := s.pfeAssignmentRepo.FindBySubjectID(subjectID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.BadRequest("Affectation déjà effectuée pour ce sujet")
	}

	ay, err := s.academicYearRepo.FindActive()
	if err != nil {
		return nil, err
	}
	if ay == nil {
		return nil, apperror.BadRequest("Aucune année académique active")
	}

	wishes, err := s.wishRepo.FindBySubject(subjectID)
	if err != nil {
		return nil, err
	}

	selectedSet := make(map[int64]bool, len(studentIDs))
	for _, sID := range studentIDs {
		selectedSet[sID] = true
	}

	for _, sID := range studentIDs {
		found := false
		for _, w := range wishes {
			if w.StudentID == sID {
				w.Status = "accepte"
				if err := s.wishRepo.Update(w); err != nil {
					return nil, err
				}
				found = true
				break
			}
		}
		if !found {
			newWish := &entity.Wish{StudentID: sID, SubjectID: subjectID, AcademicYearID: ay.ID, Status: "accepte"}
			if err := s.wishRepo.Insert(newWish); err != nil {
				return nil, err
			}
		}
	}

	for _, w := range wishes {
		if !selectedSet[w.StudentID] && w.Status == "en_attente" {
			w.Status = "refuse"
			if err := s.wishRepo.Update(w); err != nil {
				return nil, err
			}
		}
	}

	specialityCode := "GEN"
	student1, err := s.studentRepo.FindByID(studentIDs[0])
	if err == nil && student1 != nil && student1.SpecialityID != nil {
		if sp, err := s.specialityRepo.FindByID(*student1.SpecialityID); err == nil && sp != nil {
			specialityCode = sp.Code
		}
	}

	seq, err := s.pfeAssignmentRepo.CountBySpecialityAndYear(ay.ID, specialityCode)
	if err != nil {
		return nil, err
	}

	code := pfe_code.Generate(specialityCode, ay.Label, seq+1)

	supervisorID, err := s.resolveTeacherID(subject.ProposerID)
	if err != nil {
		return nil, err
	}

	assignment := &entity.PfeAssignment{
		PfeCode:        code,
		SubjectID:      subjectID,
		AcademicYearID: ay.ID,
		StudentID:      studentIDs[0],
		SupervisorID:   supervisorID,
		Status:         "en_cours",
	}
	if subject.CoSupervisorID.Valid {
		assignment.CoSupervisorID = entity.NullInt64{NullInt64: sql.NullInt64{Int64: subject.CoSupervisorID.Int64, Valid: true}}
	}
	if len(studentIDs) >= 2 {
		assignment.Student2ID = entity.NullInt64{NullInt64: sql.NullInt64{Int64: studentIDs[1], Valid: true}}
	}
	if len(studentIDs) >= 3 {
		assignment.Student3ID = entity.NullInt64{NullInt64: sql.NullInt64{Int64: studentIDs[2], Valid: true}}
	}

	if err := s.pfeAssignmentRepo.Insert(assignment); err != nil {
		return nil, err
	}
	return assignment, nil
}

func (s *TeacherService) RejectCandidat(subjectID, studentID int64) error {
	wishes, err := s.wishRepo.FindBySubject(subjectID)
	if err != nil {
		return err
	}
	for _, w := range wishes {
		if w.StudentID == studentID {
			w.Status = "refuse"
			return s.wishRepo.Update(w)
		}
	}
	return apperror.NotFound("Candidature introuvable")
}

func (s *TeacherService) ListSupervisedPFEs(userID int64) ([]*entity.PfeAssignment, error) {
	assignments, err := s.pfeAssignmentRepo.FindBySupervisor(userID)
	if err != nil {
		return nil, err
	}
	for _, a := range assignments {
		s.hydrateAssignment(a)
	}
	return assignments, nil
}

func (s *TeacherService) GetSupervisedPFE(id int64) (*entity.PfeAssignment, error) {
	a, err := s.pfeAssignmentRepo.FindByID(id)
	if err != nil || a == nil {
		return a, err
	}
	s.hydrateAssignment(a)
	return a, nil
}

func (s *TeacherService) AddMeeting(report *entity.PfeProgressReport) error {
	return s.progressRepo.Insert(report)
}

func (s *TeacherService) ListMeetings(assignmentID int64) ([]*entity.PfeProgressReport, error) {
	reports, err := s.progressRepo.FindByAssignment(assignmentID)
	if err != nil {
		return nil, err
	}
	if reports == nil {
		return []*entity.PfeProgressReport{}, nil
	}
	return reports, nil
}

func (s *TeacherService) GetEvaluation(assignmentID int64) (*entity.SupervisorEvaluation, error) {
	return s.supEvalRepo.FindByAssignment(assignmentID)
}

func (s *TeacherService) SubmitEvaluation(assignmentID, profileID int64, criterion5 float64) error {
	if criterion5 < 0 || criterion5 > 4 {
		return apperror.BadRequest("Le critère 5 doit être entre 0 et 4")
	}

	teacherID, err := s.resolveTeacherID(profileID)
	if err != nil {
		return err
	}

	assignment, err := s.pfeAssignmentRepo.FindByID(assignmentID)
	if err != nil {
		return err
	}
	if assignment == nil {
		return apperror.NotFound("Affectation introuvable")
	}
	if assignment.SupervisorID != teacherID {
		return apperror.Forbidden("Seul l'encadrant principal peut soumettre l'évaluation")
	}

	existing, err := s.supEvalRepo.FindByAssignment(assignmentID)
	if err != nil {
		return err
	}
	if existing != nil {
		existing.EvaluatorID = teacherID
		existing.Criterion5 = entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: criterion5, Valid: true}}
		return s.supEvalRepo.Update(existing)
	}
	eval := &entity.SupervisorEvaluation{
		PfeAssignmentID: assignmentID,
		EvaluatorID:     teacherID,
		Criterion5:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: criterion5, Valid: true}},
	}
	return s.supEvalRepo.Insert(eval)
}

func (s *TeacherService) UpdateAvailability(userID int64, status string, unavailableUntilStr string) error {
	validStatuses := map[string]bool{
		"disponible":            true,
		"indisponible":          true,
		"indisponible_jusqu_au": true,
	}
	if !validStatuses[status] {
		return apperror.BadRequest("Statut invalide: utilisez disponible, indisponible ou indisponible_jusqu_au")
	}
	var unavailableUntil *sql.NullTime
	if unavailableUntilStr != "" {
		t, err := time.Parse("2006-01-02", unavailableUntilStr)
		if err != nil {
			return apperror.BadRequest("Format de date invalide, utilisez YYYY-MM-DD")
		}
		unavailableUntil = &sql.NullTime{Time: t, Valid: true}
	}
	return s.teacherRepo.UpdateAvailability(userID, status, unavailableUntil)
}

func (s *TeacherService) resolveTeacherID(profileID int64) (int64, error) {
	t, err := s.teacherRepo.FindByProfileID(profileID)
	if err != nil || t == nil {
		return 0, apperror.NotFound("Profil enseignant introuvable")
	}
	return t.ID, nil
}

func (s *TeacherService) ListSubjectsToValidate(userID int64) ([]*entity.PfeSubject, error) {
	teacherID, err := s.resolveTeacherID(userID)
	if err != nil {
		return nil, err
	}
	subjects, err := s.pfeSubjectRepo.FindPendingValidation(teacherID)
	if err != nil {
		return nil, err
	}
	for _, sub := range subjects {
		s.hydrateSubject(sub)
	}
	return subjects, nil
}

func (s *TeacherService) GetSubjectToValidate(userID, id int64) (*entity.PfeSubject, error) {
	teacherID, err := s.resolveTeacherID(userID)
	if err != nil {
		return nil, err
	}
	subject, err := s.pfeSubjectRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, nil
	}
	if !isTeacherValidator(teacherID, subject) {
		return nil, apperror.Forbidden("Vous n'êtes pas validateur de ce sujet")
	}
	s.hydrateSubject(subject)
	return subject, nil
}

func (s *TeacherService) ValidateSubject(userID, id int64, decision, comment string) error {
	validDecisions := map[string]bool{"valide": true, "accepte_sous_reserve": true, "refuse": true}
	if !validDecisions[decision] {
		return apperror.BadRequest("Décision invalide: utilisez valide, accepte_sous_reserve ou refuse")
	}
	teacherID, err := s.resolveTeacherID(userID)
	if err != nil {
		return err
	}
	subject, err := s.pfeSubjectRepo.FindByID(id)
	if err != nil {
		return err
	}
	if subject == nil {
		return apperror.NotFound("Sujet introuvable")
	}
	if !isTeacherValidator(teacherID, subject) {
		return apperror.Forbidden("Vous n'êtes pas validateur de ce sujet")
	}

	var validatorField string
	if subject.Validator1ID.Valid && subject.Validator1ID.Int64 == teacherID {
		if subject.Validator1Decision.Valid {
			return apperror.BadRequest("Vous avez déjà soumis votre décision pour ce sujet")
		}
		validatorField = "validator1"
	} else if subject.Validator2ID.Valid && subject.Validator2ID.Int64 == teacherID {
		if subject.Validator2Decision.Valid {
			return apperror.BadRequest("Vous avez déjà soumis votre décision pour ce sujet")
		}
		validatorField = "validator2"
	} else {
		return apperror.Forbidden("Vous n'êtes pas validateur de ce sujet")
	}

	if err := s.pfeSubjectRepo.UpdateValidation(id, validatorField, decision, comment); err != nil {
		return err
	}

	subject, err = s.pfeSubjectRepo.FindByID(id)
	if err != nil {
		return err
	}
	newStatus := computeSubjectStatus(subject, decision)
	return s.pfeSubjectRepo.UpdateStatus(id, newStatus)
}

func (s *TeacherService) ListJuryDuties(userID int64) ([]*entity.Defense, error) {
	defenses, err := s.defenseRepo.FindByJuryMember(userID)
	if err != nil {
		return nil, err
	}
	for _, d := range defenses {
		s.hydrateDefense(d)
	}
	return defenses, nil
}

func (s *TeacherService) GetJuryDuty(id int64) (*entity.Defense, error) {
	d, err := s.defenseRepo.FindByID(id)
	if err != nil || d == nil {
		return d, err
	}
	s.hydrateDefense(d)
	return d, nil
}

type GradeContext struct {
	MyRole              string                       `json:"my_role"`              // "president" or "member"
	MyGrade             *entity.JuryGrade            `json:"my_grade"`             // null if not yet submitted
	MemberGrade         *entity.JuryGrade            `json:"member_grade"`         // member's eval (for president view)
	SupervisorEval      *entity.SupervisorEvaluation `json:"supervisor_eval"`      // supervisor criterion5
	MemberSubmitted     bool                         `json:"member_submitted"`     // has the member submitted?
	SupervisorSubmitted bool                         `json:"supervisor_submitted"` // has the supervisor submitted?
	FinalGradeSet       bool                         `json:"final_grade_set"`      // has the final grade been resolved?
}

func (s *TeacherService) GetGradeContext(defenseID, callerProfileID int64) (*GradeContext, error) {
	defense, err := s.defenseRepo.FindByID(defenseID)
	if err != nil {
		return nil, err
	}
	if defense == nil {
		return nil, apperror.NotFound("Soutenance introuvable")
	}

	jury, err := s.defenseJuryRepo.FindByID(defense.JuryID)
	if err != nil {
		return nil, err
	}
	if jury == nil {
		return nil, apperror.NotFound("Jury introuvable")
	}

	callerTeacherID, err := s.resolveTeacherID(callerProfileID)
	if err != nil {
		return nil, err
	}

	ctx := &GradeContext{}
	if jury.PresidentID == callerTeacherID {
		ctx.MyRole = "president"
	} else if jury.MemberID == callerTeacherID {
		ctx.MyRole = "member"
	} else {
		return nil, apperror.Forbidden("Vous ne faites pas partie de ce jury")
	}

	ctx.MyGrade, _ = s.juryGradeRepo.FindByDefenseAndMember(defenseID, callerTeacherID)

	memberGrade, _ := s.juryGradeRepo.FindByDefenseAndMember(defenseID, jury.MemberID)
	ctx.MemberGrade = memberGrade
	ctx.MemberSubmitted = memberGrade != nil

	if ctx.MemberGrade != nil {
		ctx.MemberGrade.JuryMember = s.hydrateTeacher(jury.MemberID)
	}
	if ctx.MyGrade != nil {
		ctx.MyGrade.JuryMember = s.hydrateTeacher(callerTeacherID)
	}

	assignment, _ := s.pfeAssignmentRepo.FindByID(defense.AssignmentID)
	if assignment != nil {
		supEval, _ := s.supEvalRepo.FindByAssignment(assignment.ID)
		ctx.SupervisorEval = supEval
		ctx.SupervisorSubmitted = supEval != nil && supEval.Criterion5.Valid
	}

	ctx.FinalGradeSet = defense.FinalGrade.Valid

	return ctx, nil
}

func (s *TeacherService) SubmitJuryGrade(defenseID, callerProfileID int64, c1, c2, c3, c4 float64, archiveDecision string) error {
	for _, v := range []float64{c1, c2, c3, c4} {
		if v < 0 || v > 4 {
			return apperror.BadRequest("Les critères doivent être entre 0 et 4")
		}
	}
	validDecisions := map[string]bool{"archivable": true, "minor_corrections": true, "major_corrections": true}
	if archiveDecision != "" && !validDecisions[archiveDecision] {
		return apperror.BadRequest("Décision d'archivage invalide")
	}

	defense, err := s.defenseRepo.FindByID(defenseID)
	if err != nil {
		return err
	}
	if defense == nil {
		return apperror.NotFound("Soutenance introuvable")
	}

	jury, err := s.defenseJuryRepo.FindByID(defense.JuryID)
	if err != nil {
		return err
	}
	if jury == nil {
		return apperror.NotFound("Jury introuvable")
	}

	callerTeacherID, err := s.resolveTeacherID(callerProfileID)
	if err != nil {
		return err
	}

	if jury.MemberID != callerTeacherID {
		return apperror.Forbidden("Seul l'examinateur (membre) peut soumettre une évaluation individuelle. Le président soumet la note finale.")
	}

	archiveNull := entity.NullString{NullString: sql.NullString{String: archiveDecision, Valid: archiveDecision != ""}}

	existing, err := s.juryGradeRepo.FindByDefenseAndMember(defenseID, callerTeacherID)
	if err != nil {
		return err
	}

	if existing != nil {
		existing.Criterion1 = entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c1, Valid: true}}
		existing.Criterion2 = entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c2, Valid: true}}
		existing.Criterion3 = entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c3, Valid: true}}
		existing.Criterion4 = entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c4, Valid: true}}
		existing.ArchiveDecision = archiveNull
		return s.juryGradeRepo.Update(existing)
	}

	grade := &entity.JuryGrade{
		DefenseID:       defenseID,
		JuryMemberID:    callerTeacherID,
		Criterion1:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c1, Valid: true}},
		Criterion2:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c2, Valid: true}},
		Criterion3:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c3, Valid: true}},
		Criterion4:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c4, Valid: true}},
		ArchiveDecision: archiveNull,
	}
	return s.juryGradeRepo.Insert(grade)
}

func (s *TeacherService) SubmitFinalGrade(defenseID, callerProfileID int64, choice string, c1, c2, c3, c4 float64, archiveDecision string) error {
	defense, err := s.defenseRepo.FindByID(defenseID)
	if err != nil {
		return err
	}
	if defense == nil {
		return apperror.NotFound("Soutenance introuvable")
	}

	jury, err := s.defenseJuryRepo.FindByID(defense.JuryID)
	if err != nil {
		return err
	}
	if jury == nil {
		return apperror.NotFound("Jury introuvable")
	}

	callerTeacherID, err := s.resolveTeacherID(callerProfileID)
	if err != nil {
		return err
	}

	if jury.PresidentID != callerTeacherID {
		return apperror.Forbidden("Seul le président du jury peut soumettre la note finale")
	}

	memberGrade, _ := s.juryGradeRepo.FindByDefenseAndMember(defenseID, jury.MemberID)
	if memberGrade == nil {
		return apperror.BadRequest("L'examinateur n'a pas encore soumis son évaluation")
	}

	assignment, err := s.pfeAssignmentRepo.FindByID(defense.AssignmentID)
	if err != nil {
		return err
	}
	if assignment == nil {
		return apperror.NotFound("Affectation introuvable")
	}
	supEval, _ := s.supEvalRepo.FindByAssignment(assignment.ID)
	if supEval == nil || !supEval.Criterion5.Valid {
		return apperror.BadRequest("L'évaluation de l'encadrant n'a pas encore été soumise")
	}

	var fc1, fc2, fc3, fc4 float64
	switch choice {
	case "member":
		fc1 = memberGrade.Criterion1.Float64
		fc2 = memberGrade.Criterion2.Float64
		fc3 = memberGrade.Criterion3.Float64
		fc4 = memberGrade.Criterion4.Float64
	case "new":
		for _, v := range []float64{c1, c2, c3, c4} {
			if v < 0 || v > 4 {
				return apperror.BadRequest("Les critères doivent être entre 0 et 4")
			}
		}
		fc1, fc2, fc3, fc4 = c1, c2, c3, c4
	default:
		return apperror.BadRequest("Choix invalide: utilisez 'member' ou 'new'")
	}

	validDecisions := map[string]bool{"archivable": true, "minor_corrections": true, "major_corrections": true}
	if archiveDecision != "" && !validDecisions[archiveDecision] {
		return apperror.BadRequest("Décision d'archivage invalide")
	}

	archiveNull := entity.NullString{NullString: sql.NullString{String: archiveDecision, Valid: archiveDecision != ""}}
	existing, _ := s.juryGradeRepo.FindByDefenseAndMember(defenseID, callerTeacherID)
	if existing != nil {
		existing.Criterion1 = entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: fc1, Valid: true}}
		existing.Criterion2 = entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: fc2, Valid: true}}
		existing.Criterion3 = entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: fc3, Valid: true}}
		existing.Criterion4 = entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: fc4, Valid: true}}
		existing.ArchiveDecision = archiveNull
		_ = s.juryGradeRepo.Update(existing)
	} else {
		grade := &entity.JuryGrade{
			DefenseID:       defenseID,
			JuryMemberID:    callerTeacherID,
			Criterion1:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: fc1, Valid: true}},
			Criterion2:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: fc2, Valid: true}},
			Criterion3:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: fc3, Valid: true}},
			Criterion4:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: fc4, Valid: true}},
			ArchiveDecision: archiveNull,
		}
		_ = s.juryGradeRepo.Insert(grade)
	}

	totalGrade := fc1 + fc2 + fc3 + fc4 + supEval.Criterion5.Float64

	result := "admitted"
	if totalGrade < 10 {
		result = "not_admitted"
	}

	if err := s.defenseRepo.UpdateResult(defenseID, result, totalGrade); err != nil {
		return err
	}

	go func() {
		subjectTitle := "votre PFE"
		if assignment.Subject != nil && assignment.Subject.Title != "" {
			subjectTitle = fmt.Sprintf("« %s »", assignment.Subject.Title)
		} else {
			sub, _ := s.pfeSubjectRepo.FindByID(assignment.SubjectID)
			if sub != nil {
				subjectTitle = fmt.Sprintf("« %s »", sub.Title)
			}
		}

		msg := fmt.Sprintf("La note finale de votre soutenance pour le sujet %s a été publiée : %.1f/20.", subjectTitle, totalGrade)

		studentIDs := []int64{assignment.StudentID}
		if assignment.Student2ID.Valid {
			studentIDs = append(studentIDs, assignment.Student2ID.Int64)
		}
		if assignment.Student3ID.Valid {
			studentIDs = append(studentIDs, assignment.Student3ID.Int64)
		}
		for _, sID := range studentIDs {
			st, err := s.studentRepo.FindByID(sID)
			if err == nil && st != nil {
				s.notifier.Send(st.ProfileID, notify.TypeJury, msg)
			}
		}
	}()

	return nil
}

func (s *TeacherService) GetMyGrade(defenseID, callerID int64) (*entity.JuryGrade, error) {
	return s.juryGradeRepo.FindByDefenseAndMember(defenseID, callerID)
}

func (s *TeacherService) ListNotifications(userID int64) ([]*entity.Notification, error) {
	return s.notificationRepo.FindByRecipient(userID)
}

func isTeacherValidator(userID int64, subject *entity.PfeSubject) bool {
	return (subject.Validator1ID.Valid && subject.Validator1ID.Int64 == userID) ||
		(subject.Validator2ID.Valid && subject.Validator2ID.Int64 == userID)
}

func setValidatorDecision(subject *entity.PfeSubject, userID int64, decision, comment string) {
	if subject.Validator1ID.Valid && subject.Validator1ID.Int64 == userID {
		subject.Validator1Decision = entity.NullString{NullString: sql.NullString{String: decision, Valid: true}}
		subject.Validator1Comment = entity.NullString{NullString: sql.NullString{String: comment, Valid: true}}
	} else if subject.Validator2ID.Valid && subject.Validator2ID.Int64 == userID {
		subject.Validator2Decision = entity.NullString{NullString: sql.NullString{String: decision, Valid: true}}
		subject.Validator2Comment = entity.NullString{NullString: sql.NullString{String: comment, Valid: true}}
	}
}

func (s *TeacherService) hydrateTeacher(id int64) *entity.Teacher {
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

func (s *TeacherService) GetSubjectTitle(subjectID int64) string {
	sub, err := s.pfeSubjectRepo.FindByID(subjectID)
	if err != nil || sub == nil {
		return fmt.Sprintf("sujet #%d", subjectID)
	}
	return sub.Title
}

func (s *TeacherService) GetStudentProfileID(studentID int64) (int64, error) {
	st, err := s.studentRepo.FindByID(studentID)
	if err != nil {
		return 0, err
	}
	if st == nil {
		return 0, apperror.NotFound("Étudiant introuvable")
	}
	return st.ProfileID, nil
}

func (s *TeacherService) hydrateStudent(id int64) *entity.Student {
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

func (s *TeacherService) hydrateSubject(sub *entity.PfeSubject) {
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

func (s *TeacherService) hydrateAssignment(a *entity.PfeAssignment) {
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

func (s *TeacherService) hydrateDefense(d *entity.Defense) {
	a, _ := s.pfeAssignmentRepo.FindByID(d.AssignmentID)
	if a != nil {
		s.hydrateAssignment(a)
		d.Assignment = a
	}
	if d.JuryID != 0 {
		jury, _ := s.defenseJuryRepo.FindByID(d.JuryID)
		if jury != nil {
			jury.President = s.hydrateTeacher(jury.PresidentID)
			jury.Member = s.hydrateTeacher(jury.MemberID)
			d.Jury = jury
		}
	}
}

func (s *TeacherService) hydrateWish(w *entity.Wish) {
	w.Student = s.hydrateStudent(w.StudentID)
	sub, _ := s.pfeSubjectRepo.FindByID(w.SubjectID)
	if sub != nil {
		s.hydrateSubject(sub)
		w.Subject = sub
	}
}

func computeSubjectStatus(subject *entity.PfeSubject, decision string) string {
	bothValid := subject.Validator1Decision.Valid && subject.Validator2Decision.Valid &&
		subject.Validator1Decision.String == "valide" && subject.Validator2Decision.String == "valide"
	bothRefused := subject.Validator1Decision.Valid && subject.Validator1Decision.String == "refuse" &&
		subject.Validator2Decision.Valid && subject.Validator2Decision.String == "refuse"

	if bothValid {
		return "valide"
	}
	if bothRefused {
		return "refuse"
	}
	if decision == "refuse" {
		return "refuse"
	}
	if decision == "accepte_sous_reserve" {
		return "accepte_sous_reserve"
	}
	return subject.Status
}

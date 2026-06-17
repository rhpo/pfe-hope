package service

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"pfe-backend/internal/entity"
	"pfe-backend/internal/repository"
	"pfe-backend/internal/shared/apperror"
	"pfe-backend/internal/shared/notify"
	"strings"
	"time"
)

type AdminService struct {
	profileRepo       *repository.ProfileRepository
	teacherRepo       *repository.TeacherRepository
	studentRepo       *repository.StudentRepository
	companyRepo       *repository.CompanyRepository
	departmentRepo    *repository.DepartmentRepository
	domainRepo        *repository.DomainRepository
	specialityRepo    *repository.SpecialityRepository
	promotionRepo     *repository.PromotionRepository
	academicYearRepo  *repository.AcademicYearRepository
	pfeSubjectRepo    *repository.PfeSubjectRepository
	wishRepo          *repository.WishRepository
	pfeAssignmentRepo *repository.PfeAssignmentRepository
	progressRepo      *repository.ProgressReportRepository
	defenseJuryRepo   *repository.DefenseJuryRepository
	defenseRepo       *repository.DefenseRepository
	juryGradeRepo     *repository.JuryGradeRepository
	supEvalRepo       *repository.SupervisorEvaluationRepository
	companyReportRepo *repository.CompanyReportRepository
	notificationRepo  *repository.NotificationRepository
	auditLogRepo      *repository.AuditLogRepository
	notifier          *notify.Notifier
	uploadDir         string
}

func NewAdminService(
	profileRepo *repository.ProfileRepository,
	teacherRepo *repository.TeacherRepository,
	studentRepo *repository.StudentRepository,
	companyRepo *repository.CompanyRepository,
	departmentRepo *repository.DepartmentRepository,
	domainRepo *repository.DomainRepository,
	specialityRepo *repository.SpecialityRepository,
	promotionRepo *repository.PromotionRepository,
	academicYearRepo *repository.AcademicYearRepository,
	pfeSubjectRepo *repository.PfeSubjectRepository,
	wishRepo *repository.WishRepository,
	pfeAssignmentRepo *repository.PfeAssignmentRepository,
	progressRepo *repository.ProgressReportRepository,
	defenseJuryRepo *repository.DefenseJuryRepository,
	defenseRepo *repository.DefenseRepository,
	juryGradeRepo *repository.JuryGradeRepository,
	supEvalRepo *repository.SupervisorEvaluationRepository,
	companyReportRepo *repository.CompanyReportRepository,
	notificationRepo *repository.NotificationRepository,
	auditLogRepo *repository.AuditLogRepository,
	notifier *notify.Notifier,
	uploadDir string,
) *AdminService {
	return &AdminService{
		profileRepo:       profileRepo,
		teacherRepo:       teacherRepo,
		studentRepo:       studentRepo,
		companyRepo:       companyRepo,
		departmentRepo:    departmentRepo,
		domainRepo:        domainRepo,
		specialityRepo:    specialityRepo,
		promotionRepo:     promotionRepo,
		academicYearRepo:  academicYearRepo,
		pfeSubjectRepo:    pfeSubjectRepo,
		wishRepo:          wishRepo,
		pfeAssignmentRepo: pfeAssignmentRepo,
		progressRepo:      progressRepo,
		defenseJuryRepo:   defenseJuryRepo,
		defenseRepo:       defenseRepo,
		juryGradeRepo:     juryGradeRepo,
		supEvalRepo:       supEvalRepo,
		companyReportRepo: companyReportRepo,
		notificationRepo:  notificationRepo,
		auditLogRepo:      auditLogRepo,
		notifier:          notifier,
		uploadDir:         uploadDir,
	}
}

func (s *AdminService) UploadDir() string {
	return s.uploadDir
}

func (s *AdminService) Dashboard() (map[string]any, error) {
	profiles, err := s.profileRepo.FindAll()
	if err != nil {
		return nil, err
	}
	teachers, _ := s.teacherRepo.FindAll()
	students, _ := s.studentRepo.FindAll()
	companies, _ := s.companyRepo.FindAll()
	subjects, _ := s.pfeSubjectRepo.FindAll()
	assignments, _ := s.pfeAssignmentRepo.FindAll()
	defenses, _ := s.defenseRepo.FindAll()
	reports, _ := s.companyReportRepo.FindAll()

	pendingSubjects := 0
	validatedSubjects := 0
	rejectedSubjects := 0

	for _, s := range subjects {
		switch s.Status {
		case "en_attente":
			pendingSubjects++
		case "valide", "accepte_sous_reserve":
			validatedSubjects++
		case "refuse":
			rejectedSubjects++
		}
	}

	const timelineMonths = 10
	monthlyStats, _ := s.pfeAssignmentRepo.MonthlyTimelineStats(timelineMonths)
	totalStudents := len(students)

	tlLabels := make([]string, timelineMonths)
	tlAvecSujet := make([]int, timelineMonths)
	tlSansSujet := make([]int, timelineMonths)
	tlSoumisMemoire := make([]int, timelineMonths)

	for i, m := range monthlyStats {
		tlLabels[i] = m.Label
		tlAvecSujet[i] = m.WithSubject
		sans := totalStudents - m.WithSubject
		if sans < 0 {
			sans = 0
		}
		tlSansSujet[i] = sans
		tlSoumisMemoire[i] = m.MemoireSubmit
	}

	timeline := map[string]interface{}{
		"labels":         tlLabels,
		"soumis_memoire": tlSoumisMemoire,
		"avec_sujet":     tlAvecSujet,
		"sans_sujet":     tlSansSujet,
	}

	return map[string]any{
		"total_users":        len(profiles),
		"total_teachers":     len(teachers),
		"total_students":     len(students),
		"total_companies":    len(companies),
		"total_subjects":     len(subjects),
		"pending_subjects":   pendingSubjects,
		"validated_subjects": validatedSubjects,
		"rejected_subjects":  rejectedSubjects,
		"assigned_subjects":  len(assignments),
		"total_assignments":  len(assignments),
		"total_defenses":     len(defenses),
		"total_reports":      len(reports),
		"timeline":           timeline,
	}, nil
}

func (s *AdminService) ListUsers() ([]*entity.Profile, error) {
	profiles, err := s.profileRepo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, p := range profiles {
		s.hydrateProfileData(p)
	}
	return profiles, nil
}

func (s *AdminService) GetUser(id int64) (*entity.Profile, error) {
	p, err := s.profileRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}

	s.hydrateProfileData(p)

	return p, nil
}

func (s *AdminService) hydrateProfileData(p *entity.Profile) {
	if p == nil {
		return
	}
	switch p.Role {
	case "teacher", "admin":
		p.Teacher, _ = s.teacherRepo.FindByProfileID(p.ID)
		if p.Teacher != nil {
			if p.Teacher.DepartmentID != nil {
				p.Teacher.Department, _ = s.departmentRepo.FindByID(*p.Teacher.DepartmentID)
			}
			p.Teacher.Domaines, _ = s.teacherRepo.GetDomains(p.Teacher.ID)
		}
	case "student":
		p.Student, _ = s.studentRepo.FindByProfileID(p.ID)
		if p.Student != nil {
			if p.Student.SpecialityID != nil {
				p.Student.Speciality, _ = s.specialityRepo.FindByID(*p.Student.SpecialityID)
			}
			if p.Student.PromotionID != nil {
				p.Student.Promotion, _ = s.promotionRepo.FindByID(*p.Student.PromotionID)
			}
		}
	case "company":
		p.Company, _ = s.companyRepo.FindByProfileID(p.ID)
	}
}

func (s *AdminService) CreateUser(profile *entity.Profile) error {
	return s.profileRepo.Insert(profile)
}

func (s *AdminService) UpdateUser(id int64, profile *entity.Profile) error {
	existing, err := s.profileRepo.FindByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return apperror.NotFound("Utilisateur introuvable")
	}
	if profile.FullName != "" {
		existing.FullName = profile.FullName
	}
	if profile.Email != "" {
		existing.Email = profile.Email
	}
	if profile.Role != "" {
		existing.Role = profile.Role
	}
	return s.profileRepo.Update(existing)
}

func (s *AdminService) DeactivateUser(id int64) error {
	profile, err := s.profileRepo.FindByID(id)
	if err != nil {
		return err
	}
	if profile == nil {
		return apperror.NotFound("Utilisateur introuvable")
	}
	profile.IsActive = false
	return s.profileRepo.Update(profile)
}

func (s *AdminService) ReactivateUser(id int64) error {
	profile, err := s.profileRepo.FindByID(id)
	if err != nil {
		return err
	}
	if profile == nil {
		return apperror.NotFound("Utilisateur introuvable")
	}
	profile.IsActive = true
	return s.profileRepo.Update(profile)
}

func (s *AdminService) ListCompanies() ([]*entity.Company, error) {
	companies, err := s.companyRepo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, c := range companies {
		c.Profile, _ = s.profileRepo.FindByID(c.ProfileID)
	}
	return companies, nil
}

func (s *AdminService) ValidateCompany(id int64) error {
	company, err := s.companyRepo.FindByID(id)
	if err != nil {
		return err
	}
	if company == nil {
		return apperror.NotFound("Entreprise introuvable")
	}
	return s.companyRepo.UpdateVerification(id, true)
}

func (s *AdminService) RejectCompany(id int64) error {
	company, err := s.companyRepo.FindByID(id)
	if err != nil {
		return err
	}
	if company == nil {
		return apperror.NotFound("Entreprise introuvable")
	}
	return s.companyRepo.UpdateVerification(id, false)
}

func (s *AdminService) ListReports() ([]*entity.CompanyReport, error) {
	reports, err := s.companyReportRepo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, r := range reports {
		r.Company, _ = s.companyRepo.FindByID(r.CompanyID)
	}
	return reports, nil
}

func (s *AdminService) ResolveReport(id int64) error {
	report, err := s.companyReportRepo.FindByID(id)
	if err != nil {
		return err
	}
	if report == nil {
		return apperror.NotFound("Report introuvable")
	}
	return s.companyReportRepo.UpdateStatus(id, "resolu")
}

func (s *AdminService) RejectReport(id int64) error {
	report, err := s.companyReportRepo.FindByID(id)
	if err != nil {
		return err
	}
	if report == nil {
		return apperror.NotFound("Report introuvable")
	}
	return s.companyReportRepo.UpdateStatus(id, "rejete")
}

func (s *AdminService) ListSubjects() ([]*entity.PfeSubject, error) {
	subjects, err := s.pfeSubjectRepo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, sub := range subjects {
		s.hydrateSubject(sub)
	}
	return subjects, nil
}

func (s *AdminService) GetSubject(id int64) (*entity.PfeSubject, error) {
	sub, err := s.pfeSubjectRepo.FindByID(id)
	if err != nil || sub == nil {
		return sub, err
	}
	s.hydrateSubject(sub)
	return sub, nil
}

func (s *AdminService) ListAssignments() ([]*entity.PfeAssignment, error) {
	assignments, err := s.pfeAssignmentRepo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, a := range assignments {
		s.hydrateAssignment(a)
	}
	return assignments, nil
}

func (s *AdminService) GetAssignment(id int64) (*entity.PfeAssignment, error) {
	a, err := s.pfeAssignmentRepo.FindByID(id)
	if err != nil || a == nil {
		return a, err
	}
	s.hydrateAssignment(a)
	return a, nil
}

func (s *AdminService) ListDefenses() ([]*entity.Defense, error) {
	defenses, err := s.defenseRepo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, d := range defenses {
		s.hydrateDefense(d)
	}
	return defenses, nil
}

func (s *AdminService) GetDefense(id int64) (*entity.Defense, error) {
	d, err := s.defenseRepo.FindByID(id)
	if err != nil || d == nil {
		return d, err
	}
	s.hydrateDefense(d)
	return d, nil
}

func (s *AdminService) ListAcademicYears() ([]*entity.AcademicYear, error) {
	return s.academicYearRepo.FindAll()
}

func (s *AdminService) CreateAcademicYear(ay *entity.AcademicYear) error {
	if ay.Status == "active" {
		existing, err := s.academicYearRepo.FindActive()
		if err != nil {
			return err
		}
		if existing != nil {
			return apperror.Conflict("Une année académique est déjà active (« " + existing.Label + " »). Clôturez-la d'abord.")
		}
	}

	if ay.MaxWishes <= 0 {
		ay.MaxWishes = 5
	}
	return s.academicYearRepo.Insert(ay)
}

func (s *AdminService) CloseAcademicYear(id int64) error {
	ay, err := s.academicYearRepo.FindByID(id)
	if err != nil {
		return err
	}
	if ay == nil {
		return apperror.NotFound("Année académique introuvable")
	}
	return s.academicYearRepo.Close(id)
}

func (s *AdminService) ListSpecialities() ([]*entity.Speciality, error) {
	return s.specialityRepo.FindAll()
}

func (s *AdminService) CreateSpeciality(sp *entity.Speciality) error {
	return s.specialityRepo.Insert(sp)
}

func (s *AdminService) DeleteSpeciality(id int64) error {
	return s.specialityRepo.Delete(id)
}

func (s *AdminService) ListDepartments() ([]*entity.Department, error) {
	return s.departmentRepo.FindAll()
}

func (s *AdminService) CreateDepartment(d *entity.Department) error {
	return s.departmentRepo.Insert(d)
}

func (s *AdminService) DeleteDepartment(id int64) error {
	return s.departmentRepo.Delete(id)
}

func (s *AdminService) ListDomains() ([]*entity.Domain, error) {
	return s.domainRepo.FindAll()
}

func (s *AdminService) CreateDomain(d *entity.Domain) error {
	return s.domainRepo.Insert(d)
}

func (s *AdminService) DeleteDomain(id int64) error {
	return s.domainRepo.Delete(id)
}

func (s *AdminService) ListPromotions() ([]*entity.Promotion, error) {
	return s.promotionRepo.FindAll()
}

func (s *AdminService) CreatePromotion(p *entity.Promotion) error {
	return s.promotionRepo.Insert(p)
}

func (s *AdminService) DeletePromotion(id int64) error {
	return s.promotionRepo.Delete(id)
}

func (s *AdminService) GetTeacherByID(id int64) (*entity.Teacher, error) {
	return s.teacherRepo.FindByID(id)
}

func (s *AdminService) GetTeacherProfileID(teacherID int64) int64 {
	t, err := s.teacherRepo.FindByID(teacherID)
	if err != nil || t == nil {

		t2, err2 := s.teacherRepo.FindByProfileID(teacherID)
		if err2 != nil || t2 == nil {
			return 0
		}
		return t2.ProfileID
	}
	return t.ProfileID
}

func (s *AdminService) GetStudentProfileID(studentID int64) int64 {
	st, err := s.studentRepo.FindByID(studentID)
	if err != nil || st == nil {
		return 0
	}
	return st.ProfileID
}

func (s *AdminService) AssignValidators(subjectID, validator1ID, validator2ID int64) error {
	subject, err := s.pfeSubjectRepo.FindByID(subjectID)
	if err != nil {
		return err
	}
	if subject == nil {
		return apperror.NotFound("Sujet introuvable")
	}
	return s.pfeSubjectRepo.AssignValidators(subjectID, validator1ID, validator2ID)
}

func (s *AdminService) AssignCoSupervisor(subjectID, coSupervisorID int64) error {
	subject, err := s.pfeSubjectRepo.FindByID(subjectID)
	if err != nil {
		return err
	}
	if subject == nil {
		return apperror.NotFound("Sujet introuvable")
	}
	return s.pfeSubjectRepo.AssignCoSupervisor(subjectID, coSupervisorID)
}

func (s *AdminService) AssignPfeCoSupervisor(assignmentID, teacherID int64) error {
	assignment, err := s.pfeAssignmentRepo.FindByID(assignmentID)
	if err != nil {
		return err
	}
	if assignment == nil {
		return apperror.NotFound("Affectation introuvable")
	}
	if assignment.SupervisorID == teacherID {
		return apperror.BadRequest("Le co-encadrant ne peut pas être le même que l'encadrant principal")
	}
	return s.pfeAssignmentRepo.UpdateCoSupervisor(assignmentID, teacherID)
}

func (s *AdminService) RemovePfeCoSupervisor(assignmentID int64) error {
	assignment, err := s.pfeAssignmentRepo.FindByID(assignmentID)
	if err != nil {
		return err
	}
	if assignment == nil {
		return apperror.NotFound("Affectation introuvable")
	}
	return s.pfeAssignmentRepo.RemoveCoSupervisor(assignmentID)
}

func (s *AdminService) RecommendCoSupervisor(assignmentID int64) (map[string]any, error) {
	assignment, err := s.pfeAssignmentRepo.FindByID(assignmentID)
	if err != nil {
		return nil, err
	}
	if assignment == nil {
		return nil, apperror.NotFound("Affectation introuvable")
	}

	subjectDomains, err := s.pfeSubjectRepo.GetDomains(assignment.SubjectID)
	if err != nil {
		return nil, err
	}
	subjectDomainIDs := make(map[int64]bool, len(subjectDomains))
	for _, d := range subjectDomains {
		subjectDomainIDs[d.ID] = true
	}

	teachers, err := s.teacherRepo.FindAll()
	if err != nil {
		return nil, err
	}

	type recommendation struct {
		Teacher         *entity.Teacher  `json:"teacher"`
		Score           int              `json:"score"`
		MatchingDomains []*entity.Domain `json:"matching_domains"`
	}

	var recommendations []recommendation
	for _, t := range teachers {
		if t.AvailabilityStatus != "disponible" {
			continue
		}

		if t.ID == assignment.SupervisorID {
			continue
		}

		teacherDomains, err := s.teacherRepo.GetDomains(t.ID)
		if err != nil {
			continue
		}

		profile, _ := s.profileRepo.FindByID(t.ProfileID)
		t.Profile = profile
		t.Domaines = teacherDomains

		var matching []*entity.Domain
		for _, td := range teacherDomains {
			if subjectDomainIDs[td.ID] {
				matching = append(matching, td)
			}
		}

		recommendations = append(recommendations, recommendation{
			Teacher:         t,
			Score:           len(matching),
			MatchingDomains: matching,
		})
	}

	for i := 0; i < len(recommendations); i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[j].Score > recommendations[i].Score {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}

	return map[string]any{
		"recommended":     recommendations,
		"assignment_id":   assignmentID,
		"subject_domains": subjectDomains,
	}, nil
}

func (s *AdminService) GetStatistics() (map[string]any, error) {
	return s.Dashboard()
}

func (s *AdminService) AuditLog() ([]*entity.AuditLog, error) {
	return s.auditLogRepo.FindAll()
}

func (s *AdminService) UserAction(id int64, action string) error {
	switch action {
	case "deactivate":
		return s.DeactivateUser(id)
	case "reactivate", "activate":
		return s.ReactivateUser(id)
	case "transfer-admin":
		return s.TransferAdmin(id)
	default:
		return apperror.BadRequest("Action non reconnue: " + action)
	}
}

func (s *AdminService) UpdateTeacherProfile(profileID int64, fullName, email, grade string, departmentID *int64, domainIDs []int64) error {
	profile, err := s.profileRepo.FindByID(profileID)
	if err != nil {
		return err
	}
	if profile == nil {
		return apperror.NotFound("Utilisateur introuvable")
	}
	if fullName != "" {
		profile.FullName = fullName
	}
	if email != "" {
		profile.Email = email
	}
	if err := s.profileRepo.Update(profile); err != nil {
		return err
	}

	teacher, err := s.teacherRepo.FindByProfileID(profileID)
	if err != nil {
		return err
	}
	if teacher != nil {
		if grade != "" {
			teacher.Grade = entity.NullString{NullString: sql.NullString{String: grade, Valid: true}}
		}
		teacher.DepartmentID = departmentID
		if err := s.teacherRepo.Update(teacher); err != nil {
			return err
		}

		if domainIDs != nil {

			existing, _ := s.teacherRepo.GetDomains(teacher.ID)
			for _, d := range existing {
				_ = s.teacherRepo.RemoveDomain(teacher.ID, d.ID)
			}
			for _, dID := range domainIDs {
				_ = s.teacherRepo.AddDomain(teacher.ID, dID)
			}
		}
	}
	return nil
}

func (s *AdminService) UpdateStudentProfile(profileID int64, fullName, email, studentNumber, level string, specialityID *int64, promotionID *int64) error {
	profile, err := s.profileRepo.FindByID(profileID)
	if err != nil {
		return err
	}
	if profile == nil {
		return apperror.NotFound("Utilisateur introuvable")
	}
	if fullName != "" {
		profile.FullName = fullName
	}
	if email != "" {
		profile.Email = email
	}
	if err := s.profileRepo.Update(profile); err != nil {
		return err
	}

	student, err := s.studentRepo.FindByProfileID(profileID)
	if err != nil {
		return err
	}
	if student != nil {
		if studentNumber != "" {
			s := studentNumber
			student.StudentNumber = &s
		}
		if level != "" {
			l := level
			student.Level = &l
		}
		student.SpecialityID = specialityID
		student.PromotionID = promotionID
		_ = s.studentRepo.Update(student)
	}
	return nil
}

func (s *AdminService) UpdateUserAvatar(profileID int64, url string) error {
	return s.profileRepo.UpdateAvatarURL(profileID, url)
}

func (s *AdminService) TransferAdmin(id int64) error {
	profile, err := s.profileRepo.FindByID(id)
	if err != nil {
		return err
	}
	if profile == nil {
		return apperror.NotFound("Utilisateur introuvable")
	}
	if profile.Role != "teacher" {
		return apperror.BadRequest("Le transfert admin n'est possible que vers un enseignant")
	}

	currentAdmin, err := s.FindAdmin()
	if err != nil {
		return err
	}
	if currentAdmin != nil {
		currentAdmin.Role = "teacher"
		if err := s.profileRepo.Update(currentAdmin); err != nil {
			return err
		}
	}
	profile.Role = "admin"
	return s.profileRepo.Update(profile)
}

func (s *AdminService) FindAdmin() (*entity.Profile, error) {
	all, err := s.profileRepo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, p := range all {
		if p.Role == "admin" {
			return p, nil
		}
	}
	return nil, nil
}

func (s *AdminService) CreateTeacher(fullName, email, grade string, departmentID *int64) (*entity.Profile, error) {
	if fullName == "" || email == "" {
		return nil, apperror.BadRequest("full_name et email sont requis")
	}
	if grade == "" {
		grade = "assistant"
	}

	profile := &entity.Profile{
		Role: "teacher", FullName: fullName, Email: email, IsActive: true,
	}
	if err := s.profileRepo.Insert(profile); err != nil {
		return nil, err
	}

	teacher := &entity.Teacher{
		ProfileID:          profile.ID,
		Grade:              entity.NullString{NullString: sql.NullString{String: grade, Valid: true}},
		DepartmentID:       departmentID,
		AvailabilityStatus: "disponible",
	}
	if err := s.teacherRepo.Insert(teacher); err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *AdminService) CreateStudent(fullName, email, studentNumber string, specialityID *int64, level string, promotionID *int64) (*entity.Profile, error) {
	if fullName == "" || email == "" || studentNumber == "" {
		return nil, apperror.BadRequest("full_name, email et student_number sont requis")
	}

	profile := &entity.Profile{
		Role: "student", FullName: fullName, Email: email, IsActive: true,
	}
	if err := s.profileRepo.Insert(profile); err != nil {
		return nil, err
	}

	student := &entity.Student{
		ProfileID:     profile.ID,
		StudentNumber: &studentNumber,
		SpecialityID:  specialityID,
		Level:         &level,
		PromotionID:   promotionID,
	}
	if err := s.studentRepo.Insert(student); err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *AdminService) ImportUsersCSV(csvData, csvType string, replace bool) error {
	r := csv.NewReader(strings.NewReader(csvData))
	records, err := r.ReadAll()
	if err != nil {
		return apperror.BadRequest("CSV invalide: " + err.Error())
	}
	if len(records) < 2 {
		return apperror.BadRequest("CSV vide ou sans données")
	}

	allDomains, _ := s.domainRepo.FindAll()
	domainByName := make(map[string]*entity.Domain)
	for _, d := range allDomains {
		domainByName[strings.ToLower(strings.TrimSpace(d.Name))] = d
	}

	allDepts, _ := s.departmentRepo.FindAll()
	deptByName := make(map[string]*entity.Department)
	for _, d := range allDepts {
		deptByName[strings.ToLower(strings.TrimSpace(d.Name))] = d
	}

	allSpecs, _ := s.specialityRepo.FindAll()
	specByCode := make(map[string]*entity.Speciality)
	for _, sp := range allSpecs {
		specByCode[strings.ToLower(strings.TrimSpace(sp.Code))] = sp
	}

	allPromos, _ := s.promotionRepo.FindAll()
	promoByLabel := make(map[string]*entity.Promotion)
	for _, p := range allPromos {
		promoByLabel[strings.ToLower(strings.TrimSpace(p.Label))] = p
	}

	for i, row := range records[1:] {
		lineNum := i + 2

		switch csvType {
		case "teachers":
			if len(row) < 2 {
				return apperror.BadRequest(fmt.Sprintf("Ligne %d: au moins nom_complet et email requis", lineNum))
			}
			fullName := strings.TrimSpace(row[0])
			email := strings.TrimSpace(row[1])
			grade := "assistant"
			var departmentID *int64
			if len(row) > 2 && strings.TrimSpace(row[2]) != "" {
				grade = strings.TrimSpace(row[2])
			}
			if len(row) > 3 && strings.TrimSpace(row[3]) != "" {
				if d, ok := deptByName[strings.ToLower(strings.TrimSpace(row[3]))]; ok {
					departmentID = &d.ID
				}
			}

			existing, _ := s.profileRepo.FindByEmail(email)
			var profileID int64
			if existing != nil {
				if !replace {

					profileID = existing.ID
				} else {
					existing.FullName = fullName

					if existing.Role != "admin" {
						existing.Role = "teacher"
					}
					if err := s.profileRepo.Update(existing); err != nil {
						return fmt.Errorf("erreur update ligne %d: %w", lineNum, err)
					}
					profileID = existing.ID
				}
			} else {
				profile := &entity.Profile{
					Role: "teacher", FullName: fullName, Email: email, IsActive: true,
				}
				if err := s.profileRepo.Insert(profile); err != nil {
					return fmt.Errorf("erreur import ligne %d: %w", lineNum, err)
				}
				profileID = profile.ID
			}

			existingTeacher, _ := s.teacherRepo.FindByProfileID(profileID)
			if existingTeacher == nil {
				teacher := &entity.Teacher{
					ProfileID:          profileID,
					Grade:              entity.NullString{NullString: sql.NullString{String: grade, Valid: true}},
					DepartmentID:       departmentID,
					AvailabilityStatus: "disponible",
				}
				if err := s.teacherRepo.Insert(teacher); err != nil {
					return fmt.Errorf("erreur création enseignant ligne %d: %w", lineNum, err)
				}
				existingTeacher = teacher
			} else if replace {
				existingTeacher.Grade = entity.NullString{NullString: sql.NullString{String: grade, Valid: true}}
				existingTeacher.DepartmentID = departmentID
				_ = s.teacherRepo.Update(existingTeacher)
			}

			if len(row) > 4 && strings.TrimSpace(row[4]) != "" {
				specCode := strings.ToLower(strings.TrimSpace(row[4]))
				if sp, ok := specByCode[specCode]; ok {
					_ = sp
				}
			}

			if len(row) > 5 && strings.TrimSpace(row[5]) != "" {
				domainNames := strings.Split(row[5], ";")
				for _, dn := range domainNames {
					dn = strings.TrimSpace(dn)
					if dn == "" {
						continue
					}
					if d, ok := domainByName[strings.ToLower(dn)]; ok {
						_ = s.teacherRepo.AddDomain(existingTeacher.ID, d.ID)
					}
				}
			}

		case "students":
			if len(row) < 3 {
				return apperror.BadRequest(fmt.Sprintf("Ligne %d: nom_complet, email, numero_etudiant requis", lineNum))
			}
			fullName := strings.TrimSpace(row[0])
			email := strings.TrimSpace(row[1])
			studentNumber := strings.TrimSpace(row[2])

			existing, _ := s.profileRepo.FindByEmail(email)
			var profileID int64
			if existing != nil {
				if !replace {
					continue
				}
				existing.FullName = fullName
				existing.Role = "student"
				if err := s.profileRepo.Update(existing); err != nil {
					return fmt.Errorf("erreur update ligne %d: %w", lineNum, err)
				}
				profileID = existing.ID
			} else {
				profile := &entity.Profile{
					Role: "student", FullName: fullName, Email: email, IsActive: true,
				}
				if err := s.profileRepo.Insert(profile); err != nil {
					return fmt.Errorf("erreur import ligne %d: %w", lineNum, err)
				}
				profileID = profile.ID
			}

			var specialityID *int64
			if len(row) > 3 && strings.TrimSpace(row[3]) != "" {
				if sp, ok := specByCode[strings.ToLower(strings.TrimSpace(row[3]))]; ok {
					specialityID = &sp.ID
				}
			}
			level := ""
			if len(row) > 4 {
				level = strings.TrimSpace(row[4])
			}
			var promotionID *int64
			if len(row) > 5 && strings.TrimSpace(row[5]) != "" {
				if p, ok := promoByLabel[strings.ToLower(strings.TrimSpace(row[5]))]; ok {
					promotionID = &p.ID
				}
			}

			existingStudent, _ := s.studentRepo.FindByProfileID(profileID)
			if existingStudent == nil {
				student := &entity.Student{
					ProfileID:     profileID,
					StudentNumber: &studentNumber,
					SpecialityID:  specialityID,
					Level:         &level,
					PromotionID:   promotionID,
				}
				if err := s.studentRepo.Insert(student); err != nil {
					return fmt.Errorf("erreur création étudiant ligne %d: %w", lineNum, err)
				}
			} else if replace {
				existingStudent.StudentNumber = &studentNumber
				existingStudent.SpecialityID = specialityID
				existingStudent.Level = &level
				existingStudent.PromotionID = promotionID
				_ = s.studentRepo.Update(existingStudent)
			}

		default:
			return apperror.BadRequest("Type CSV invalide: " + csvType)
		}
	}
	return nil
}

func (s *AdminService) CompanyAction(id int64, action string) error {
	switch action {
	case "validate":
		return s.ValidateCompany(id)
	case "reject":
		return s.RejectCompany(id)
	default:
		return apperror.BadRequest("Action non reconnue: " + action)
	}
}

func (s *AdminService) ReportAction(id int64, action string) error {
	switch action {
	case "resolve":
		return s.ResolveReport(id)
	case "reject":
		return s.RejectReport(id)
	default:
		return apperror.BadRequest("Action non reconnue: " + action)
	}
}

func (s *AdminService) SubjectAction(id int64, action string, validatorID, validator1ID, validator2ID int64) error {
	switch action {
	case "assign-validators":

		v1 := validator1ID
		if v1 == 0 {
			v1 = validatorID
		}
		v2 := validator2ID
		if v1 == 0 {
			return apperror.BadRequest("validator1_id requis")
		}
		return s.AssignValidators(id, v1, v2)
	case "assign-co-supervisor":
		if validatorID == 0 {
			return apperror.BadRequest("co_supervisor_id requis")
		}
		return s.AssignCoSupervisor(id, validatorID)
	case "unblock":
		return s.UnblockSubject(id)
	default:
		return apperror.BadRequest("Action non reconnue: " + action)
	}
}

func (s *AdminService) UnblockSubject(id int64) error {
	subject, err := s.pfeSubjectRepo.FindByID(id)
	if err != nil {
		return err
	}
	if subject == nil {
		return apperror.NotFound("Sujet introuvable")
	}
	return s.pfeSubjectRepo.UpdateStatus(id, "en_attente")
}

func (s *AdminService) CreateDefense(assignmentID, presidentID, memberID int64, scheduledAt, room string) (*entity.Defense, error) {
	if assignmentID == 0 || presidentID == 0 || memberID == 0 {
		return nil, apperror.BadRequest("assignment_id, president_id et member_id sont requis")
	}
	if presidentID == memberID {
		return nil, apperror.BadRequest("Le président et le membre du jury doivent être différents")
	}

	assignment, err := s.pfeAssignmentRepo.FindByID(assignmentID)
	if err != nil {
		return nil, err
	}
	if assignment == nil {
		return nil, apperror.NotFound("Affectation introuvable")
	}

	academicYear, err := s.academicYearRepo.FindByID(assignment.AcademicYearID)
	if err != nil || academicYear == nil {
		return nil, apperror.BadRequest("Année académique introuvable pour cette affectation")
	}
	if academicYear.Status != "active" {
		return nil, apperror.BadRequest("Impossible de planifier une soutenance : l'année académique est clôturée")
	}

	jury := &entity.DefenseJury{
		AssignmentID: assignmentID,
		PresidentID:  presidentID,
		MemberID:     memberID,
	}
	if err := s.defenseJuryRepo.Insert(jury); err != nil {
		return nil, err
	}

	var scheduledAtTime entity.NullTime
	if t, err := time.Parse(time.RFC3339, scheduledAt); err == nil {
		scheduledAtTime = entity.NullTime{NullTime: sql.NullTime{Time: t, Valid: true}}
	}

	defense := &entity.Defense{
		AssignmentID: assignmentID,
		JuryID:       jury.ID,
		ScheduledAt:  scheduledAtTime,
		Room:         entity.NullString{NullString: sql.NullString{String: room, Valid: room != ""}},
		Status:       "scheduled",
	}
	if err := s.defenseRepo.Insert(defense); err != nil {
		return nil, err
	}

	_ = s.pfeAssignmentRepo.UpdateStatus(assignmentID, "soutenance_planifiee")

	s.notifyDefenseScheduled(defense, jury)

	return defense, nil
}

func (s *AdminService) notifyDefenseScheduled(defense *entity.Defense, jury *entity.DefenseJury) {
	subject := s.hydrateSubjectFromAssignment(defense.AssignmentID)
	title := "votre PFE"
	if subject != nil {
		title = fmt.Sprintf("« %s »", subject.Title)
	}

	assignment, _ := s.pfeAssignmentRepo.FindByID(defense.AssignmentID)
	if assignment != nil {
		students := []int64{assignment.StudentID}
		if assignment.Student2ID.Valid {
			students = append(students, assignment.Student2ID.Int64)
		}
		if assignment.Student3ID.Valid {
			students = append(students, assignment.Student3ID.Int64)
		}
		for _, studentEntityID := range students {
			st, _ := s.studentRepo.FindByID(studentEntityID)
			if st != nil {
				go s.notifier.Send(st.ProfileID, notify.TypeJury,
					fmt.Sprintf("Votre soutenance pour le sujet %s a été planifiée.", title))
			}
		}
	}

	if t, _ := s.teacherRepo.FindByID(jury.PresidentID); t != nil {
		go s.notifier.Send(t.ProfileID, notify.TypeJury,
			fmt.Sprintf("Vous avez été désigné président du jury pour la soutenance du sujet %s.", title))
	}

	if t, _ := s.teacherRepo.FindByID(jury.MemberID); t != nil {
		go s.notifier.Send(t.ProfileID, notify.TypeJury,
			fmt.Sprintf("Vous avez été désigné membre du jury pour la soutenance du sujet %s.", title))
	}
}

func (s *AdminService) hydrateSubjectFromAssignment(assignmentID int64) *entity.PfeSubject {
	a, _ := s.pfeAssignmentRepo.FindByID(assignmentID)
	if a == nil {
		return nil
	}
	sub, _ := s.pfeSubjectRepo.FindByID(a.SubjectID)
	return sub
}

func (s *AdminService) RecommendJury(pfeID int64) (map[string]any, error) {
	subject, err := s.pfeSubjectRepo.FindByID(pfeID)
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, apperror.NotFound("Sujet introuvable")
	}

	subjectDomains, err := s.pfeSubjectRepo.GetDomains(pfeID)
	if err != nil {
		return nil, err
	}
	subjectDomainIDs := make(map[int64]bool, len(subjectDomains))
	for _, d := range subjectDomains {
		subjectDomainIDs[d.ID] = true
	}

	teachers, err := s.teacherRepo.FindAll()
	if err != nil {
		return nil, err
	}

	type recommendation struct {
		Teacher         *entity.Teacher  `json:"teacher"`
		Score           int              `json:"score"`
		MatchingDomains []*entity.Domain `json:"matching_domains"`
	}

	var recommendations []recommendation
	for _, t := range teachers {
		if t.AvailabilityStatus != "disponible" {
			continue
		}

		if t.ProfileID == subject.ProposerID {
			continue
		}

		teacherDomains, err := s.teacherRepo.GetDomains(t.ID)
		if err != nil {
			continue
		}

		profile, _ := s.profileRepo.FindByID(t.ProfileID)
		t.Profile = profile
		t.Domaines = teacherDomains

		var matching []*entity.Domain
		for _, td := range teacherDomains {
			if subjectDomainIDs[td.ID] {
				matching = append(matching, td)
			}
		}

		recommendations = append(recommendations, recommendation{
			Teacher:         t,
			Score:           len(matching),
			MatchingDomains: matching,
		})
	}

	for i := 0; i < len(recommendations); i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[j].Score > recommendations[i].Score {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}

	return map[string]any{
		"recommended":     recommendations,
		"pfe_id":          pfeID,
		"subject_domains": subjectDomains,
	}, nil
}

func (s *AdminService) SubmitGrade(defenseID, callerID int64, c1, c2, c3, c4 float64, archiveDecision string) error {

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

	archiveNull := entity.NullString{NullString: sql.NullString{String: archiveDecision, Valid: archiveDecision != ""}}

	existing, err := s.juryGradeRepo.FindByDefenseAndMember(defenseID, callerID)
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
		JuryMemberID:    callerID,
		Criterion1:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c1, Valid: true}},
		Criterion2:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c2, Valid: true}},
		Criterion3:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c3, Valid: true}},
		Criterion4:      entity.NullFloat64{NullFloat64: sql.NullFloat64{Float64: c4, Valid: true}},
		ArchiveDecision: archiveNull,
	}
	return s.juryGradeRepo.Insert(grade)
}

type ResolveGradeRequest struct {
	Choice     string
	Criterion1 float64
	Criterion2 float64
	Criterion3 float64
	Criterion4 float64
	Grades     map[string]float64
}

func (s *AdminService) ResolveGrade(defenseID int64, req ResolveGradeRequest) error {
	defense, err := s.defenseRepo.FindByID(defenseID)
	if err != nil {
		return err
	}
	if defense == nil {
		return apperror.NotFound("Soutenance introuvable")
	}

	var c1, c2, c3, c4 float64

	switch req.Choice {
	case "president", "member":

		jury, err := s.defenseJuryRepo.FindByID(defense.JuryID)
		if err != nil {
			return err
		}
		if jury == nil {
			return apperror.NotFound("Jury introuvable")
		}

		var juryMemberID int64
		if req.Choice == "president" {
			juryMemberID = jury.PresidentID
		} else {
			juryMemberID = jury.MemberID
		}

		grade, err := s.juryGradeRepo.FindByDefenseAndMember(defenseID, juryMemberID)
		if err != nil {
			return err
		}
		if grade == nil {
			return apperror.BadRequest("Aucune note soumise par ce membre du jury")
		}
		c1 = grade.Criterion1.Float64
		c2 = grade.Criterion2.Float64
		c3 = grade.Criterion3.Float64
		c4 = grade.Criterion4.Float64

	case "new":
		if req.Grades == nil {
			return apperror.BadRequest("Les notes sont requises pour le choix 'new'")
		}
		c1 = req.Grades["criterion1"]
		c2 = req.Grades["criterion2"]
		c3 = req.Grades["criterion3"]
		c4 = req.Grades["criterion4"]

	default:

		c1 = req.Criterion1
		c2 = req.Criterion2
		c3 = req.Criterion3
		c4 = req.Criterion4
	}

	assignment, err := s.pfeAssignmentRepo.FindByID(defense.AssignmentID)
	if err != nil {
		return err
	}
	if assignment == nil {
		return apperror.NotFound("Affectation introuvable")
	}

	supEval, _ := s.supEvalRepo.FindByAssignment(assignment.ID)
	criterion5 := 0.0
	if supEval != nil {
		criterion5 = supEval.Criterion5.Float64
	}

	totalGrade := c1 + c2 + c3 + c4 + criterion5
	return s.defenseRepo.UpdateResult(defenseID, "admitted", totalGrade)
}

func (s *AdminService) findJuryByIDOrDefenseID(id int64) (*entity.DefenseJury, error) {
	jury, err := s.defenseJuryRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if jury != nil {
		return jury, nil
	}

	defense, err := s.defenseRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if defense == nil {
		return nil, apperror.NotFound("Jury introuvable")
	}
	jury, err = s.defenseJuryRepo.FindByID(defense.JuryID)
	if err != nil {
		return nil, err
	}
	if jury == nil {
		return nil, apperror.NotFound("Jury introuvable")
	}
	return jury, nil
}

func (s *AdminService) ConfirmJury(id int64) error {
	jury, err := s.findJuryByIDOrDefenseID(id)
	if err != nil {
		return err
	}
	if err := s.defenseJuryRepo.ConfirmPresident(jury.ID); err != nil {
		return err
	}
	return s.defenseJuryRepo.ConfirmMember(jury.ID)
}

func (s *AdminService) DeclineJury(id int64) error {
	jury, err := s.findJuryByIDOrDefenseID(id)
	if err != nil {
		return err
	}
	return s.defenseJuryRepo.Delete(jury.ID)
}

func (s *AdminService) UpdateDeadlines(openAt, closeAt string, maxWishes int) error {

	years, err := s.academicYearRepo.FindAll()
	if err != nil {
		return err
	}
	for _, y := range years {
		if y.Status == "active" {

			if t, err := time.Parse(time.RFC3339, openAt); err == nil {
				y.SubmissionOpenAt = entity.NullTime{NullTime: sql.NullTime{Time: t, Valid: true}}
			}
			if t, err := time.Parse(time.RFC3339, closeAt); err == nil {
				y.SubmissionCloseAt = entity.NullTime{NullTime: sql.NullTime{Time: t, Valid: true}}
			}
			y.MaxWishes = maxWishes
			return s.academicYearRepo.Update(y)
		}
	}
	return apperror.NotFound("Aucune année académique active trouvée")
}

func (s *AdminService) hydrateTeacher(id int64) *entity.Teacher {
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

func (s *AdminService) hydrateStudent(id int64) *entity.Student {
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

func (s *AdminService) hydrateSubject(sub *entity.PfeSubject) {
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

func (s *AdminService) hydrateAssignment(a *entity.PfeAssignment) {
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

func (s *AdminService) hydrateDefense(d *entity.Defense) {
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

func (s *AdminService) GetCompany(id int64) (*entity.Company, error) {
	return s.companyRepo.FindByID(id)
}

func (s *AdminService) GetCompaniesByName(name string) ([]*entity.Company, error) {
	return s.companyRepo.FindAllByName(name)
}

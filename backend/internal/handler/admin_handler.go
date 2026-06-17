package handler

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pfe-backend/internal/entity"
	"pfe-backend/internal/service"
	"pfe-backend/internal/shared/middleware"
	"pfe-backend/internal/shared/notify"
	"pfe-backend/internal/shared/response"

	"github.com/gofiber/fiber/v3"
)

type AdminHandler struct {
	svc      *service.AdminService
	notifier *notify.Notifier
}

func NewAdminHandler(svc *service.AdminService, notifier *notify.Notifier) *AdminHandler {
	return &AdminHandler{svc: svc, notifier: notifier}
}

func parseID(c fiber.Ctx, param string) (int64, error) {
	return strconv.ParseInt(c.Params(param), 10, 64)
}

func (h *AdminHandler) Dashboard(c fiber.Ctx) error {
	data, err := h.svc.Dashboard()
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, data)
}

func (h *AdminHandler) ListUsers(c fiber.Ctx) error {
	users, err := h.svc.ListUsers()
	if err != nil {
		return response.Error(c, err)
	}
	if users == nil {
		users = []*entity.Profile{}
	}
	return response.OK(c, users)
}

func (h *AdminHandler) GetUser(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	user, err := h.svc.GetUser(id)
	if err != nil {
		return response.Error(c, err)
	}

	if user == nil {
		return response.NotFound(c, "Utilisateur introuvable")
	}

	return response.OK(c, user)
}

func (h *AdminHandler) CreateUser(c fiber.Ctx) error {
	var req struct {
		Role     string `json:"role"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if req.Role == "" || req.FullName == "" || req.Email == "" {
		return response.ValidationError(c, "role, full_name et email sont requis")
	}

	profile := &entity.Profile{
		Role:     req.Role,
		FullName: req.FullName,
		Email:    req.Email,
		IsActive: true,
	}

	if err := h.svc.CreateUser(profile); err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, profile)
}

func (h *AdminHandler) UpdateUser(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}

	profile, err := h.svc.GetUser(id)
	if err != nil {
		return response.Error(c, err)
	}
	if profile == nil {
		return response.NotFound(c, "Utilisateur introuvable")
	}

	var req struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`

		Grade        string  `json:"grade"`
		DepartmentID *int64  `json:"department_id"`
		DomainIDs    []int64 `json:"domain_ids"`

		StudentNumber string `json:"student_number"`
		Level         string `json:"level"`
		SpecialityID  *int64 `json:"speciality_id"`
		PromotionID   *int64 `json:"promotion_id"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}

	switch profile.Role {
	case "teacher", "admin":
		if err := h.svc.UpdateTeacherProfile(id, req.FullName, req.Email, req.Grade, req.DepartmentID, req.DomainIDs); err != nil {
			return response.Error(c, err)
		}
	case "student":
		if err := h.svc.UpdateStudentProfile(id, req.FullName, req.Email, req.StudentNumber, req.Level, req.SpecialityID, req.PromotionID); err != nil {
			return response.Error(c, err)
		}
	default:

		p := &entity.Profile{FullName: req.FullName, Email: req.Email}
		if err := h.svc.UpdateUser(id, p); err != nil {
			return response.Error(c, err)
		}
	}

	updated, _ := h.svc.GetUser(id)
	return response.OK(c, updated)
}

func (h *AdminHandler) UpdateUserAvatar(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return response.ValidationError(c, "Fichier requis (champ: file)")
	}

	const maxSize = 2 << 20 // 2 MB
	if file.Size > maxSize {
		return response.ValidationError(c, "Fichier trop volumineux (max 2MB)")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowed[ext] {
		return response.ValidationError(c, "Type de fichier non supporté (jpg, png, webp)")
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dst := filepath.Join(h.svc.UploadDir(), "avatars", filename)
	if err := c.SaveFile(file, dst); err != nil {
		return response.Error(c, fmt.Errorf("erreur sauvegarde fichier"))
	}

	url := "/uploads/avatars/" + filename
	if err := h.svc.UpdateUserAvatar(id, url); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"url": url})
}

func (h *AdminHandler) UserAction(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	var req struct {
		Action string `json:"action"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.UserAction(id, req.Action); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Action effectuée"})
}

func (h *AdminHandler) CreateTeacher(c fiber.Ctx) error {
	var req struct {
		FullName     string `json:"full_name"`
		Email        string `json:"email"`
		Grade        string `json:"grade"`
		DepartmentID *int64 `json:"department_id"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	profile, err := h.svc.CreateTeacher(req.FullName, req.Email, req.Grade, req.DepartmentID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, profile)
}

func (h *AdminHandler) ListDepartments(c fiber.Ctx) error {
	departments, err := h.svc.ListDepartments()
	if err != nil {
		return response.Error(c, err)
	}
	if departments == nil {
		departments = []*entity.Department{}
	}
	return response.OK(c, departments)
}

func (h *AdminHandler) CreateDepartment(c fiber.Ctx) error {
	var req entity.Department
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.CreateDepartment(&req); err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, req)
}

func (h *AdminHandler) DeleteDepartment(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	if err := h.svc.DeleteDepartment(id); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Département supprimé"})
}

func (h *AdminHandler) CreateStudent(c fiber.Ctx) error {
	var req struct {
		FullName      string `json:"full_name"`
		Email         string `json:"email"`
		StudentNumber string `json:"student_number"`
		SpecialityID  *int64 `json:"speciality_id"`
		Level         string `json:"level"`
		PromotionID   *int64 `json:"promotion_id"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}

	profile, err := h.svc.CreateStudent(req.FullName, req.Email, req.StudentNumber, req.SpecialityID, req.Level, req.PromotionID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, profile)
}

func (h *AdminHandler) ImportUsersCSV(c fiber.Ctx) error {
	var req struct {
		CSVData string `json:"csv_data"`
		CSVType string `json:"csv_type"`
		Replace bool   `json:"replace"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if req.CSVType == "" {
		req.CSVType = "teachers"
	}
	if err := h.svc.ImportUsersCSV(req.CSVData, req.CSVType, req.Replace); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Import effectué"})
}

func (h *AdminHandler) ListCompanies(c fiber.Ctx) error {
	companies, err := h.svc.ListCompanies()
	if err != nil {
		return response.Error(c, err)
	}
	if companies == nil {
		companies = []*entity.Company{}
	}
	return response.OK(c, companies)
}

func (h *AdminHandler) CompanyAction(c fiber.Ctx) error {
	id, err := parseID(c, "id")

	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	var req struct {
		Action string `json:"action"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}

	targetCompany, err := h.svc.GetCompany(id)
	if err != nil {
		return response.Error(c, err)
	}

	if err := h.svc.CompanyAction(id, req.Action); err != nil {
		return response.Error(c, err)
	}

	msg := "Votre compte entreprise a été approuvé."
	if req.Action == "reject" {
		msg = "Votre compte entreprise a été rejeté."
	}

	if targetCompany != nil && targetCompany.CompanyName != nil {
		allUsersInCompany, _ := h.svc.GetCompaniesByName(*targetCompany.CompanyName)
		for _, comp := range allUsersInCompany {
			go h.notifier.Send(comp.ProfileID, notify.TypeAffectation, msg)
		}
	} else if targetCompany != nil {
		go h.notifier.Send(targetCompany.ProfileID, notify.TypeAffectation, msg)
	}

	return response.OK(c, map[string]string{"message": "Action effectuée"})
}

func (h *AdminHandler) ListReports(c fiber.Ctx) error {
	reports, err := h.svc.ListReports()
	if err != nil {
		return response.Error(c, err)
	}
	if reports == nil {
		reports = []*entity.CompanyReport{}
	}
	return response.OK(c, reports)
}

func (h *AdminHandler) ReportAction(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	var req struct {
		Action string `json:"action"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.ReportAction(id, req.Action); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Action effectuée"})
}

func (h *AdminHandler) ListSubjects(c fiber.Ctx) error {
	subjects, err := h.svc.ListSubjects()
	if err != nil {
		return response.Error(c, err)
	}
	if subjects == nil {
		subjects = []*entity.PfeSubject{}
	}
	return response.OK(c, subjects)
}

func (h *AdminHandler) GetSubject(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	subject, err := h.svc.GetSubject(id)
	if err != nil {
		return response.Error(c, err)
	}
	if subject == nil {
		return response.NotFound(c, "Sujet introuvable")
	}
	return response.OK(c, subject)
}

func (h *AdminHandler) SubjectAction(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	var req struct {
		Action     string `json:"action"`
		Validator  int64  `json:"validator_id"`  // legacy / co-supervisor
		Validator1 int64  `json:"validator1_id"` // primary validator
		Validator2 int64  `json:"validator2_id"` // secondary validator
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.SubjectAction(id, req.Action, req.Validator, req.Validator1, req.Validator2); err != nil {
		return response.Error(c, err)
	}

	if subject, err := h.svc.GetSubject(id); err == nil && subject != nil {
		var msg string
		switch req.Action {
		case "approve", "validate":
			msg = fmt.Sprintf("Votre sujet « %s » a été validé par l'administration.", subject.Title)
		case "reject":
			msg = fmt.Sprintf("Votre sujet « %s » a été rejeté par l'administration.", subject.Title)
		case "assign":
			msg = fmt.Sprintf("Le sujet « %s » a été affecté à un étudiant.", subject.Title)
		default:
			msg = fmt.Sprintf("Une action a été effectuée sur votre sujet « %s ».", subject.Title)
		}
		go h.notifier.Send(subject.ProposerID, notify.TypeAffectation, msg)

		if req.Validator1 != 0 {
			if t, err := h.svc.GetTeacherByID(req.Validator1); err == nil && t != nil {
				go h.notifier.Send(t.ProfileID, notify.TypeValidationRequise,
					fmt.Sprintf("Vous avez été désigné validateur pour le sujet « %s ».", subject.Title))
			}
		}
		if req.Validator2 != 0 {
			if t, err := h.svc.GetTeacherByID(req.Validator2); err == nil && t != nil {
				go h.notifier.Send(t.ProfileID, notify.TypeValidationRequise,
					fmt.Sprintf("Vous avez été désigné validateur pour le sujet « %s ».", subject.Title))
			}
		}
	}

	return response.OK(c, map[string]string{"message": "Action effectuée"})
}

func (h *AdminHandler) AssignmentAction(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	var req struct {
		Action    string `json:"action"`
		TeacherID int64  `json:"teacher_id"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	switch req.Action {
	case "assign-co-supervisor":
		if req.TeacherID == 0 {
			return response.ValidationError(c, "teacher_id requis")
		}
		if err := h.svc.AssignPfeCoSupervisor(id, req.TeacherID); err != nil {
			return response.Error(c, err)
		}
		return response.OK(c, map[string]string{"message": "Co-encadrant assigné avec succès"})
	case "remove-co-supervisor":
		if err := h.svc.RemovePfeCoSupervisor(id); err != nil {
			return response.Error(c, err)
		}
		return response.OK(c, map[string]string{"message": "Co-encadrant retiré"})
	default:
		return response.ValidationError(c, "Action non reconnue: "+req.Action)
	}
}

func (h *AdminHandler) RecommendCoSupervisorHandler(c fiber.Ctx) error {
	idStr := c.Query("assignment_id")
	if idStr == "" {
		return response.ValidationError(c, "assignment_id requis")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return response.ValidationError(c, "assignment_id invalide")
	}
	result, err := h.svc.RecommendCoSupervisor(id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, result)
}

func (h *AdminHandler) ListAssignments(c fiber.Ctx) error {
	assignments, err := h.svc.ListAssignments()
	if err != nil {
		return response.Error(c, err)
	}
	if assignments == nil {
		assignments = []*entity.PfeAssignment{}
	}
	return response.OK(c, assignments)
}

func (h *AdminHandler) GetAssignment(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	assignment, err := h.svc.GetAssignment(id)
	if err != nil {
		return response.Error(c, err)
	}
	if assignment == nil {
		return response.NotFound(c, "Affectation introuvable")
	}
	return response.OK(c, assignment)
}

func (h *AdminHandler) ListDefenses(c fiber.Ctx) error {
	defenses, err := h.svc.ListDefenses()
	if err != nil {
		return response.Error(c, err)
	}
	if defenses == nil {
		defenses = []*entity.Defense{}
	}
	return response.OK(c, defenses)
}

func (h *AdminHandler) CreateDefense(c fiber.Ctx) error {
	var req struct {
		AssignmentID int64  `json:"assignment_id"`
		PresidentID  int64  `json:"president_id"`
		MemberID     int64  `json:"member_id"`
		ScheduledAt  string `json:"scheduled_at"`
		Room         string `json:"room"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	defense, err := h.svc.CreateDefense(req.AssignmentID, req.PresidentID, req.MemberID, req.ScheduledAt, req.Room)
	if err != nil {
		return response.Error(c, err)
	}

	go func() {
		dateFormatted := req.ScheduledAt
		if t, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			dateFormatted = t.Format("02/01/2006 à 15h04")
		} else if t, err := time.Parse("2006-01-02T15:04", req.ScheduledAt); err == nil {
			dateFormatted = t.Format("02/01/2006 à 15h04")
		}

		subjectTitle := "votre PFE"
		assignment, _ := h.svc.GetAssignment(req.AssignmentID)
		if assignment != nil && assignment.Subject != nil {
			subjectTitle = fmt.Sprintf("« %s »", assignment.Subject.Title)
		}

		if req.PresidentID != 0 {
			if pid := h.svc.GetTeacherProfileID(req.PresidentID); pid != 0 {
				h.notifier.Send(pid, notify.TypeJury,
					fmt.Sprintf("Vous avez été désigné président du jury pour la soutenance du sujet %s, prévue le %s en salle %s.", subjectTitle, dateFormatted, req.Room))
			}
		}
		if req.MemberID != 0 {
			if pid := h.svc.GetTeacherProfileID(req.MemberID); pid != 0 {
				h.notifier.Send(pid, notify.TypeJury,
					fmt.Sprintf("Vous avez été désigné examinateur pour la soutenance du sujet %s, prévue le %s en salle %s.", subjectTitle, dateFormatted, req.Room))
			}
		}
		if assignment != nil {
			studentMsg := fmt.Sprintf("Votre soutenance pour le sujet %s a été programmée le %s en salle %s.", subjectTitle, dateFormatted, req.Room)
			if assignment.Student != nil {
				h.notifier.Send(assignment.Student.ProfileID, notify.TypeJury, studentMsg)
			}
			if assignment.Supervisor != nil {
				h.notifier.Send(assignment.Supervisor.ProfileID, notify.TypeJury,
					fmt.Sprintf("La soutenance du sujet %s a été programmée le %s en salle %s.", subjectTitle, dateFormatted, req.Room))
			}
		}
	}()

	return response.Created(c, defense)
}

func (h *AdminHandler) GetDefense(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	defense, err := h.svc.GetDefense(id)
	if err != nil {
		return response.Error(c, err)
	}
	if defense == nil {
		return response.NotFound(c, "Soutenance introuvable")
	}
	return response.OK(c, defense)
}

func (h *AdminHandler) RecommendJury(c fiber.Ctx) error {
	pfeIDStr := c.Query("pfe_id")
	if pfeIDStr == "" {
		return response.ValidationError(c, "pfe_id requis")
	}
	pfeID, err := strconv.ParseInt(pfeIDStr, 10, 64)
	if err != nil {
		return response.ValidationError(c, "pfe_id invalide")
	}
	recommendation, err := h.svc.RecommendJury(pfeID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, recommendation)
}

func (h *AdminHandler) SubmitGrade(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	callerID := middleware.GetProfileID(c)
	var req struct {
		Criterion1      float64 `json:"criterion1"`
		Criterion2      float64 `json:"criterion2"`
		Criterion3      float64 `json:"criterion3"`
		Criterion4      float64 `json:"criterion4"`
		ArchiveDecision string  `json:"archive_decision"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.SubmitGrade(id, callerID, req.Criterion1, req.Criterion2, req.Criterion3, req.Criterion4, req.ArchiveDecision); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Note soumise"})
}

func (h *AdminHandler) ResolveGrade(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	var req struct {
		Choice     string             `json:"choice"`
		Criterion1 float64            `json:"criterion1"`
		Criterion2 float64            `json:"criterion2"`
		Criterion3 float64            `json:"criterion3"`
		Criterion4 float64            `json:"criterion4"`
		Grades     map[string]float64 `json:"grades"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	svcReq := service.ResolveGradeRequest{
		Choice:     req.Choice,
		Criterion1: req.Criterion1,
		Criterion2: req.Criterion2,
		Criterion3: req.Criterion3,
		Criterion4: req.Criterion4,
		Grades:     req.Grades,
	}
	if err := h.svc.ResolveGrade(id, svcReq); err != nil {
		return response.Error(c, err)
	}

	go func() {
		defense, err := h.svc.GetDefense(id)
		if err != nil || defense == nil || defense.AssignmentID == 0 {
			return
		}
		assignment, err := h.svc.GetAssignment(defense.AssignmentID)
		if err != nil || assignment == nil {
			return
		}
		subjectTitle := "votre PFE"
		if assignment.Subject != nil && assignment.Subject.Title != "" {
			subjectTitle = fmt.Sprintf("« %s »", assignment.Subject.Title)
		}
		msg := fmt.Sprintf("La note finale de votre soutenance pour le sujet %s a été délibérée. Consultez votre espace pour voir le résultat.", subjectTitle)
		if assignment.Student != nil {
			h.notifier.Send(assignment.Student.ProfileID, notify.TypeJury, msg)
		}

		if assignment.Supervisor != nil {
			h.notifier.Send(assignment.Supervisor.ProfileID, notify.TypeJury,
				fmt.Sprintf("La note finale pour le sujet %s a été délibérée par l'administration.", subjectTitle))
		}
	}()

	return response.OK(c, map[string]string{"message": "Note résolue"})
}

func (h *AdminHandler) ConfirmJury(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	if err := h.svc.ConfirmJury(id); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Jury confirmé"})
}

func (h *AdminHandler) DeclineJury(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	if err := h.svc.DeclineJury(id); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Jury décliné"})
}

func (h *AdminHandler) ListDeadlines(c fiber.Ctx) error {
	return h.ListAcademicYears(c)
}

func (h *AdminHandler) UpdateDeadlines(c fiber.Ctx) error {
	var req struct {
		OpenAt  string `json:"submission_open_at"`
		CloseAt string `json:"submission_close_at"`
		MaxWish int    `json:"max_wishes"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.UpdateDeadlines(req.OpenAt, req.CloseAt, req.MaxWish); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Délais mis à jour"})
}

func (h *AdminHandler) ListSpecialities(c fiber.Ctx) error {
	specialities, err := h.svc.ListSpecialities()
	if err != nil {
		return response.Error(c, err)
	}
	if specialities == nil {
		specialities = []*entity.Speciality{}
	}
	return response.OK(c, specialities)
}

func (h *AdminHandler) CreateSpeciality(c fiber.Ctx) error {
	var req entity.Speciality
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.CreateSpeciality(&req); err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, req)
}

func (h *AdminHandler) DeleteSpeciality(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	if err := h.svc.DeleteSpeciality(id); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Spécialité supprimée"})
}

func (h *AdminHandler) ListDomains(c fiber.Ctx) error {
	domains, err := h.svc.ListDomains()
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, domains)
}

func (h *AdminHandler) CreateDomain(c fiber.Ctx) error {
	var req entity.Domain
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.CreateDomain(&req); err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, req)
}

func (h *AdminHandler) DeleteDomain(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	if err := h.svc.DeleteDomain(id); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Domaine supprimé"})
}

func (h *AdminHandler) ListPromotions(c fiber.Ctx) error {
	promotions, err := h.svc.ListPromotions()
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, promotions)
}

func (h *AdminHandler) CreatePromotion(c fiber.Ctx) error {
	var req entity.Promotion
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.CreatePromotion(&req); err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, req)
}

func (h *AdminHandler) DeletePromotion(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	if err := h.svc.DeletePromotion(id); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Promotion supprimée"})
}

func (h *AdminHandler) Statistics(c fiber.Ctx) error {
	stats, err := h.svc.GetStatistics()
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, stats)
}

func (h *AdminHandler) AuditLog(c fiber.Ctx) error {
	logs, err := h.svc.AuditLog()
	if err != nil {
		return response.Error(c, err)
	}
	if logs == nil {
		logs = []*entity.AuditLog{}
	}
	return response.OK(c, logs)
}

func (h *AdminHandler) ExportAffectations(c fiber.Ctx) error {
	affectations, err := h.svc.ListAssignments()
	if err != nil {
		return response.Error(c, err)
	}
	if affectations == nil {
		affectations = []*entity.PfeAssignment{}
	}
	return response.OK(c, affectations)
}

func (h *AdminHandler) ExportPlannings(c fiber.Ctx) error {
	defenses, err := h.svc.ListDefenses()
	if err != nil {
		return response.Error(c, err)
	}
	if defenses == nil {
		defenses = []*entity.Defense{}
	}
	return response.OK(c, defenses)
}

func (h *AdminHandler) ExportStatistics(c fiber.Ctx) error {
	stats, err := h.svc.GetStatistics()
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, stats)
}

func (h *AdminHandler) ListAcademicYears(c fiber.Ctx) error {
	years, err := h.svc.ListAcademicYears()
	if err != nil {
		return response.Error(c, err)
	}
	if years == nil {
		years = []*entity.AcademicYear{}
	}
	return response.OK(c, years)
}

func (h *AdminHandler) CreateAcademicYear(c fiber.Ctx) error {
	var req entity.AcademicYear
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.CreateAcademicYear(&req); err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, req)
}

func (h *AdminHandler) CloseAcademicYear(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	if err := h.svc.CloseAcademicYear(id); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Année académique clôturée"})
}

package handler

import (
	"database/sql"
	"fmt"
	"time"

	"pfe-backend/internal/entity"
	"pfe-backend/internal/service"
	"pfe-backend/internal/shared/middleware"
	"pfe-backend/internal/shared/notify"
	"pfe-backend/internal/shared/response"

	"github.com/gofiber/fiber/v3"
)

type CompanyHandler struct {
	svc      *service.CompanyService
	notifier *notify.Notifier
}

func NewCompanyHandler(svc *service.CompanyService, notifier *notify.Notifier) *CompanyHandler {
	return &CompanyHandler{svc: svc, notifier: notifier}
}

func (h *CompanyHandler) Dashboard(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	data, err := h.svc.Dashboard(userID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, data)
}

func (h *CompanyHandler) ListSubjects(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	subjects, err := h.svc.ListSubjects(userID)
	if err != nil {
		return response.Error(c, err)
	}
	if subjects == nil {
		subjects = []*entity.PfeSubject{}
	}
	return response.OK(c, subjects)
}

func (h *CompanyHandler) CreateSubject(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	var req struct {
		entity.PfeSubject
		DomainIDs []int64 `json:"domain_ids"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if req.Title == "" || req.Description == "" {
		return response.ValidationError(c, "Titre et description requis")
	}
	if err := h.svc.CreateSubject(userID, &req.PfeSubject, req.DomainIDs); err != nil {
		return response.Error(c, err)
	}

	go h.notifier.NotifyAdmins(notify.TypeValidationRequise,
		fmt.Sprintf("Un sujet externe « %s » a été proposé par une entreprise.", req.Title))

	return response.Created(c, req.PfeSubject)
}

func (h *CompanyHandler) GetSubject(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	subject, err := h.svc.GetSubject(userID, id)
	if err != nil {
		return response.Error(c, err)
	}
	if subject == nil {
		return response.NotFound(c, "Sujet introuvable")
	}
	return response.OK(c, subject)
}

func (h *CompanyHandler) UpdateSubject(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	var req entity.PfeSubject
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	req.ID = id
	if err := h.svc.UpdateSubject(userID, &req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Sujet mis à jour"})
}

func (h *CompanyHandler) ListCandidats(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	candidats, err := h.svc.ListCandidats(id)
	if err != nil {
		return response.Error(c, err)
	}
	if candidats == nil {
		candidats = []*entity.Wish{}
	}
	return response.OK(c, candidats)
}

func (h *CompanyHandler) AcceptCandidat(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	var req struct {
		StudentIDs []int64 `json:"student_ids"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if len(req.StudentIDs) == 0 {
		return response.ValidationError(c, "Aucun étudiant sélectionné")
	}
	if err := h.svc.AcceptCandidats(id, req.StudentIDs); err != nil {
		return response.Error(c, err)
	}
	subjectTitle := h.svc.GetSubjectTitle(id)
	for _, studentID := range req.StudentIDs {
		go func(sID int64) {
			if profileID, err := h.svc.GetStudentProfileID(sID); err == nil {
				h.notifier.Send(profileID, notify.TypeAffectation,
					fmt.Sprintf("Votre candidature pour le sujet « %s » a été acceptée par l'entreprise. Votre PFE a été créé.", subjectTitle))
			}
		}(studentID)
	}
	return response.OK(c, map[string]string{"message": "Candidats acceptés et PFE créé"})
}

func (h *CompanyHandler) ListSupervisedPFEs(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	assignments, err := h.svc.ListSupervisedPFEs(userID)
	if err != nil {
		return response.Error(c, err)
	}
	if assignments == nil {
		assignments = []*entity.PfeAssignment{}
	}
	return response.OK(c, assignments)
}

func (h *CompanyHandler) GetSupervisedPFE(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	assignment, err := h.svc.GetSupervisedPFE(id)
	if err != nil {
		return response.Error(c, err)
	}
	if assignment == nil {
		return response.NotFound(c, "PFE introuvable")
	}
	return response.OK(c, assignment)
}

func (h *CompanyHandler) AddMeeting(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	var req struct {
		MeetingDate string `json:"meeting_date"`
		Duration    int    `json:"duration"`
		MeetingType string `json:"meeting_type"`
		Topics      string `json:"topics"`
		Status      string `json:"status"`
		Observation string `json:"observation"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if req.MeetingDate == "" {
		return response.ValidationError(c, "La date est requise")
	}
	if req.MeetingType == "" {
		return response.ValidationError(c, "Le type de réunion est requis")
	}
	if req.Duration == 0 {
		return response.ValidationError(c, "La durée est requise")
	}

	var meetingDate time.Time
	var parseErr error
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02"} {
		meetingDate, parseErr = time.Parse(layout, req.MeetingDate)
		if parseErr == nil {
			break
		}
	}
	if parseErr != nil {
		return response.ValidationError(c, "Format de date invalide (attendu YYYY-MM-DD)")
	}

	status := req.Status
	if status == "" {
		status = "a_faire"
	}

	report := &entity.PfeProgressReport{
		AssignmentID: id,
		MeetingDate:  meetingDate,
		Duration:     req.Duration,
		MeetingType:  req.MeetingType,
		Topics:       req.Topics,
		Status:       status,
	}
	if req.Observation != "" {
		report.Observation = entity.NullString{NullString: sql.NullString{String: req.Observation, Valid: true}}
	}

	if err := h.svc.AddMeeting(report); err != nil {
		return response.Error(c, err)
	}

	go func() {
		assignment, err := h.svc.GetSupervisedPFE(id)
		if err != nil || assignment == nil {
			return
		}
		subjectTitle := "votre PFE"
		if assignment.Subject != nil && assignment.Subject.Title != "" {
			subjectTitle = fmt.Sprintf("« %s »", assignment.Subject.Title)
		}
		dateStr := meetingDate.Format("02/01/2006")
		if assignment.Student != nil {
			h.notifier.Send(assignment.Student.ProfileID, notify.TypeDisponibilite,
				fmt.Sprintf("Une réunion de suivi pour le sujet %s a été planifiée le %s par votre encadrant.", subjectTitle, dateStr))
		}
	}()

	return response.Created(c, report)
}

func (h *CompanyHandler) ListMeetings(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	meetings, err := h.svc.ListMeetings(id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, meetings)
}

func (h *CompanyHandler) GetEvaluation(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	eval, err := h.svc.GetEvaluation(id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, eval)
}

func (h *CompanyHandler) SubmitEvaluation(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	userID := middleware.GetProfileID(c)
	var req struct {
		Criterion5 float64 `json:"criterion5"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if err := h.svc.SubmitEvaluation(id, userID, req.Criterion5); err != nil {
		return response.Error(c, err)
	}

	go func() {
		assignment, err := h.svc.GetSupervisedPFE(id)
		if err != nil || assignment == nil {
			return
		}
		subjectTitle := "votre PFE"
		if assignment.Subject != nil && assignment.Subject.Title != "" {
			subjectTitle = fmt.Sprintf("« %s »", assignment.Subject.Title)
		}
		if assignment.Student != nil {
			h.notifier.Send(assignment.Student.ProfileID, notify.TypeSujet,
				fmt.Sprintf("L'évaluation de votre encadrant pour le sujet %s a été soumise.", subjectTitle))
		}
		h.notifier.NotifyAdmins(notify.TypeValidationRequise,
			fmt.Sprintf("L'évaluation de l'encadrant (entreprise) pour le sujet %s a été soumise.", subjectTitle))
	}()

	return response.OK(c, map[string]string{"message": "Évaluation soumise"})
}

func (h *CompanyHandler) ListReports(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	reports, err := h.svc.ListReports(userID)
	if err != nil {
		return response.Error(c, err)
	}
	if reports == nil {
		reports = []*entity.CompanyReport{}
	}
	return response.OK(c, reports)
}

func (h *CompanyHandler) CreateReport(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	var req entity.CompanyReport
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if req.Description == "" || req.CorrectionType == "" {
		return response.ValidationError(c, "Description et correction_type requis")
	}
	if err := h.svc.CreateReport(userID, &req); err != nil {
		return response.Error(c, err)
	}

	go h.notifier.NotifyAdmins(notify.TypeValidationRequise,
		"Un nouveau signalement a été soumis par une entreprise.")

	return response.Created(c, req)
}

func (h *CompanyHandler) ListNotifications(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	notifications, err := h.svc.ListNotifications(userID)
	if err != nil {
		return response.Error(c, err)
	}
	if notifications == nil {
		notifications = []*entity.Notification{}
	}
	return response.OK(c, notifications)
}

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

type StudentHandler struct {
	svc      *service.StudentService
	notifier *notify.Notifier
}

func NewStudentHandler(svc *service.StudentService, notifier *notify.Notifier) *StudentHandler {
	return &StudentHandler{svc: svc, notifier: notifier}
}

func (h *StudentHandler) Dashboard(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	data, err := h.svc.Dashboard(userID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, data)
}

func (h *StudentHandler) ListCatalogue(c fiber.Ctx) error {
	subjects, err := h.svc.ListCatalogue()
	if err != nil {
		return response.Error(c, err)
	}
	if subjects == nil {
		subjects = []*entity.PfeSubject{}
	}
	return response.OK(c, subjects)
}

func (h *StudentHandler) GetCatalogueSubject(c fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	subject, err := h.svc.GetCatalogueSubject(id)
	if err != nil {
		return response.Error(c, err)
	}
	if subject == nil {
		return response.NotFound(c, "Sujet introuvable")
	}
	return response.OK(c, subject)
}

func (h *StudentHandler) ListWishes(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	wishes, err := h.svc.ListWishes(userID)
	if err != nil {
		return response.Error(c, err)
	}
	if wishes == nil {
		wishes = []*entity.Wish{}
	}
	return response.OK(c, wishes)
}

func (h *StudentHandler) CreateWish(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	var req struct {
		SubjectID int64 `json:"subject_id"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if req.SubjectID == 0 {
		return response.ValidationError(c, "L'ID du sujet est requis")
	}
	if err := h.svc.CreateWish(userID, req.SubjectID); err != nil {
		return response.Error(c, err)
	}

	go func() {
		subjectTitle := fmt.Sprintf("sujet #%d", req.SubjectID)
		if sub, err := h.svc.GetCatalogueSubject(req.SubjectID); err == nil && sub != nil {
			subjectTitle = sub.Title
		}
		h.notifier.NotifyAdmins(notify.TypeAffectation,
			fmt.Sprintf("Un étudiant a postulé au sujet « %s ».", subjectTitle))

		proposerID, _ := h.svc.GetSubjectProposerID(req.SubjectID)
		if proposerID != 0 {
			h.notifier.Send(proposerID, notify.TypeSujet,
				fmt.Sprintf("Un étudiant a postulé à votre sujet « %s ».", subjectTitle))
		}
	}()

	return response.Created(c, map[string]string{"message": "Voeu créé"})
}

func (h *StudentHandler) DeleteWish(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	if err := h.svc.DeleteWish(userID, id); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Voeu supprimé"})
}

func (h *StudentHandler) GetMyPFE(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	assignment, err := h.svc.GetMyPFE(userID)
	if err != nil {
		return response.Error(c, err)
	}
	if assignment == nil {
		return response.NotFound(c, "aucun PFE assigné")
	}
	return response.OK(c, assignment)
}

func (h *StudentHandler) ListMyMeetings(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	assignment, err := h.svc.GetMyPFE(userID)
	if err != nil {
		return response.Error(c, err)
	}
	if assignment == nil {
		return response.NotFound(c, "aucun PFE assigné")
	}
	meetings, err := h.svc.ListMyMeetings(assignment.ID)
	if err != nil {
		return response.Error(c, err)
	}
	if meetings == nil {
		meetings = []*entity.PfeProgressReport{}
	}
	return response.OK(c, meetings)
}

func (h *StudentHandler) AddMyMeeting(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)

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
		status = "en_cours"
	}

	report := &entity.PfeProgressReport{
		MeetingDate: meetingDate,
		Duration:    req.Duration,
		MeetingType: req.MeetingType,
		Topics:      req.Topics,
		Status:      status,
	}
	if req.Observation != "" {
		report.Observation = entity.NullString{NullString: sql.NullString{String: req.Observation, Valid: true}}
	}

	if err := h.svc.AddMyMeeting(userID, report); err != nil {
		return response.Error(c, err)
	}

	go func() {
		assignment, err := h.svc.GetMyPFE(userID)
		if err != nil || assignment == nil {
			return
		}
		subjectTitle := "un PFE"
		if assignment.Subject != nil && assignment.Subject.Title != "" {
			subjectTitle = fmt.Sprintf("« %s »", assignment.Subject.Title)
		}
		dateStr := meetingDate.Format("02/01/2006")
		if assignment.Supervisor != nil {
			h.notifier.Send(assignment.Supervisor.ProfileID, notify.TypeDisponibilite,
				fmt.Sprintf("Un étudiant a ajouté une réunion de suivi pour le sujet %s le %s.", subjectTitle, dateStr))
		}
	}()

	return response.Created(c, report)
}

func (h *StudentHandler) UpdateMyMeeting(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	id, err := parseID(c, "id")
	if err != nil {
		return response.ValidationError(c, "ID invalide")
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if req.Status == "" {
		return response.ValidationError(c, "Le statut est requis")
	}
	if err := h.svc.UpdateMyMeeting(userID, id, req.Status); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "Statut mis à jour"})
}

func (h *StudentHandler) SubmitMemoire(c fiber.Ctx) error {
	var req struct {
		MemoireURL string `json:"memoire_url"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.ValidationError(c, "Données invalides")
	}
	if req.MemoireURL == "" {
		return response.ValidationError(c, "URL du mémoire requis")
	}

	userID := middleware.GetProfileID(c)
	assignment, err := h.svc.GetMyPFE(userID)
	if err != nil {
		return response.Error(c, err)
	}
	if assignment == nil {
		return response.NotFound(c, "aucun PFE assigné")
	}
	if err := h.svc.SubmitMemoire(assignment.ID, req.MemoireURL); err != nil {
		return response.Error(c, err)
	}

	go func() {
		subjectTitle := "un PFE"
		if assignment.Subject != nil && assignment.Subject.Title != "" {
			subjectTitle = fmt.Sprintf("« %s »", assignment.Subject.Title)
		}
		h.notifier.NotifyAdmins(notify.TypeValidationRequise,
			fmt.Sprintf("Le mémoire pour le sujet %s a été déposé par l'étudiant.", subjectTitle))
		if assignment.Supervisor != nil {
			h.notifier.Send(assignment.Supervisor.ProfileID, notify.TypeValidationRequise,
				fmt.Sprintf("Un étudiant que vous encadrez a déposé son mémoire pour le sujet %s.", subjectTitle))
		}
	}()

	return response.OK(c, map[string]string{"message": "Mémoire soumis"})
}

func (h *StudentHandler) GetSoutenance(c fiber.Ctx) error {
	userID := middleware.GetProfileID(c)
	data, err := h.svc.GetSoutenance(userID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, data)
}

func (h *StudentHandler) GetSettings(c fiber.Ctx) error {
	settings, err := h.svc.GetSettings()
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, settings)
}

func (h *StudentHandler) ListNotifications(c fiber.Ctx) error {
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

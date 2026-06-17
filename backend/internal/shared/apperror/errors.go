package apperror

import (
	"fmt"
	"net/http"
)

type Error struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
	Err     error  `json:"-"`
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) StatusCode() int {
	return e.Code
}

func BadRequest(msg string) *Error {
	return &Error{Code: http.StatusBadRequest, Message: msg}
}

func Unauthorized(msg string) *Error {
	return &Error{Code: http.StatusUnauthorized, Message: msg}
}

func Forbidden(msg string) *Error {
	return &Error{Code: http.StatusForbidden, Message: msg}
}

func NotFound(msg string) *Error {
	return &Error{Code: http.StatusNotFound, Message: msg}
}

func Conflict(msg string) *Error {
	return &Error{Code: http.StatusConflict, Message: msg}
}

func Internal(msg string) *Error {
	return &Error{Code: http.StatusInternalServerError, Message: msg}
}

func Wrap(code int, msg string, err error) *Error {
	return &Error{Code: code, Message: msg, Err: err}
}

const (
	MsgInvalidInput         = "Données invalides"
	MsgUnauthenticated      = "Authentification requise"
	MsgForbidden            = "Accès non autorisé"
	MsgNotFound             = "Ressource introuvable"
	MsgConflict             = "Conflit avec une ressource existante"
	MsgInternalServer       = "Erreur interne du serveur"
	MsgDevOnly              = "Cet endpoint est disponible uniquement en développement"
	MsgAccountDeactivated   = "Ce compte a été désactivé"
	MsgInvalidFileType      = "Type de fichier non autorisé"
	MsgFileTooLarge         = "Le fichier dépasse la taille maximale autorisée"
	MsgAcademicYearClosed   = "L'année universitaire est clôturée"
	MsgSubmissionClosed     = "La période de soumission est fermée"
	MsgMaxWishesReached     = "Nombre maximum de vœux atteint"
	MsgSubjectNotAvailable  = "Ce sujet n'est plus disponible"
	MsgAlreadyAssigned      = "Un PFE est déjà assigné à cet étudiant"
	MsgJuryAlreadyConfirmed = "Ce membre du jury a déjà confirmé"
)

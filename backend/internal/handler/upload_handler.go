package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pfe-backend/internal/repository"
	"pfe-backend/internal/shared/middleware"
	"pfe-backend/internal/shared/response"

	"github.com/gofiber/fiber/v3"
)

const (
	maxAvatarSize  = 2 << 20  // 2 MB
	maxLogoSize    = 5 << 20  // 5 MB
	maxMemoireSize = 50 << 20 // 50 MB
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

var allowedPDFType = map[string]bool{
	"application/pdf": true,
}

type UploadHandler struct {
	profileRepo *repository.ProfileRepository
	companyRepo *repository.CompanyRepository
	uploadDir   string
}

func NewUploadHandler(profileRepo *repository.ProfileRepository, companyRepo *repository.CompanyRepository, uploadDir string) *UploadHandler {
	for _, sub := range []string{"avatars", "logos", "memoires"} {
		_ = os.MkdirAll(filepath.Join(uploadDir, sub), 0755)
	}
	return &UploadHandler{profileRepo: profileRepo, companyRepo: companyRepo, uploadDir: uploadDir}
}

func extToMIME(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	}
	return ""
}

func (h *UploadHandler) validateAndSave(c fiber.Ctx, subDir string, maxSize int64, allowedTypes map[string]bool) (string, error) {
	file, err := c.FormFile("file")
	if err != nil {
		return "", fmt.Errorf("Fichier requis (champ: file)")
	}
	if file.Size > maxSize {
		return "", fmt.Errorf("Fichier trop volumineux (max %dMB)", maxSize/(1<<20))
	}
	ext := filepath.Ext(file.Filename)
	mimeType := extToMIME(ext)
	if mimeType == "" || !allowedTypes[mimeType] {
		return "", fmt.Errorf("Type de fichier non supporté")
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), strings.ToLower(ext))
	dst := filepath.Join(h.uploadDir, subDir, filename)
	if err := c.SaveFile(file, dst); err != nil {
		return "", fmt.Errorf("Erreur sauvegarde fichier")
	}
	return "/uploads/" + subDir + "/" + filename, nil
}

func (h *UploadHandler) UploadAvatar(c fiber.Ctx) error {
	url, err := h.validateAndSave(c, "avatars", maxAvatarSize, allowedImageTypes)
	if err != nil {
		return response.ValidationError(c, err.Error())
	}
	profileID := middleware.GetProfileID(c)
	if profileID != 0 && h.profileRepo != nil {
		_ = h.profileRepo.UpdateAvatarURL(profileID, url)
	}
	return response.OK(c, map[string]string{"url": url})
}

func (h *UploadHandler) UploadCompanyLogo(c fiber.Ctx) error {
	role := middleware.GetRole(c)
	if role != "company" && role != "admin" {
		return response.ValidationError(c, "Rôle non autorisé (company ou admin requis)")
	}
	url, err := h.validateAndSave(c, "logos", maxLogoSize, allowedImageTypes)
	if err != nil {
		return response.ValidationError(c, err.Error())
	}
	profileID := middleware.GetProfileID(c)
	if profileID != 0 && h.companyRepo != nil {
		_ = h.companyRepo.UpdateLogoURLByProfileID(profileID, url)
	}
	return response.OK(c, map[string]string{"url": url})
}

func (h *UploadHandler) UploadMemoire(c fiber.Ctx) error {
	role := middleware.GetRole(c)
	if role != "student" && role != "admin" {
		return response.ValidationError(c, "Rôle non autorisé (student ou admin requis)")
	}
	url, err := h.validateAndSave(c, "memoires", maxMemoireSize, allowedPDFType)
	if err != nil {
		return response.ValidationError(c, err.Error())
	}
	return response.OK(c, map[string]string{"url": url})
}

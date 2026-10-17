package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"

	"github.com/caustin/apptracker/internal/auth"
	"github.com/caustin/apptracker/internal/storage"
	"github.com/caustin/apptracker/internal/store"
)

const maxResumeName = 200

var pdfMagic = []byte("%PDF-")

func resumeName(field, s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fiber.NewError(fiber.StatusBadRequest, field+": required")
	}
	if utf8.RuneCountInString(s) > maxResumeName {
		return "", fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("%s: at most %d characters", field, maxResumeName))
	}
	return s, nil
}

// listResumes godoc
//
//	@Summary	List resumes
//	@Tags		resumes
//	@Produce	json
//	@Success	200	{array}		domain.Resume
//	@Failure	401	{object}	ErrorResponse
//	@Router		/resumes [get]
func (h *handlers) listResumes(c fiber.Ctx) error {
	rows, err := h.store.ListResumes(c, auth.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(rows)
}

// uploadResume godoc
//
//	@Summary		Upload a resume PDF
//	@Description	PDF only (checked by content type or extension, and by the %PDF- signature), 15 MB max, 25 per user.
//	@Tags			resumes
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"The PDF"
//	@Param			name	formData	string	false	"Label; defaults to the file name without .pdf"
//	@Success		201		{object}	domain.Resume
//	@Failure		400		{object}	ErrorResponse
//	@Failure		413		{object}	ErrorResponse
//	@Router			/resumes [post]
func (h *handlers) uploadResume(c fiber.Ctx) error {
	uid := auth.UserID(c)
	fh, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is required")
	}
	if fh.Header.Get(fiber.HeaderContentType) != "application/pdf" &&
		!strings.HasSuffix(strings.ToLower(fh.Filename), ".pdf") {
		return fiber.NewError(fiber.StatusBadRequest, "only PDF files are accepted")
	}
	if fh.Size > MaxResumeBytes {
		return fiber.NewError(fiber.StatusBadRequest, "file too large (15 MB max)")
	}
	label := c.FormValue("name")
	if strings.TrimSpace(label) == "" {
		label = fh.Filename
		if strings.HasSuffix(strings.ToLower(label), ".pdf") {
			label = label[:len(label)-len(".pdf")]
		}
		if strings.TrimSpace(label) == "" {
			label = "resume"
		}
	}
	name, err := resumeName("name", label)
	if err != nil {
		return err
	}

	// Cheap early check so an over-limit user does not write a file; the
	// insert below re-checks atomically.
	n, err := h.store.CountResumes(c, uid)
	if err != nil {
		return err
	}
	if n >= MaxResumes {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("resume limit reached (%d max)", MaxResumes))
	}

	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	head := make([]byte, len(pdfMagic))
	if _, err := io.ReadFull(f, head); err != nil || !bytes.Equal(head, pdfMagic) {
		return fiber.NewError(fiber.StatusBadRequest, "only PDF files are accepted")
	}
	body := io.MultiReader(bytes.NewReader(head), io.LimitReader(f, MaxResumeBytes))

	key, err := h.files.Put(c, uid, body)
	if err != nil {
		return err
	}
	row, err := h.store.CreateResume(c, uid, name, key, MaxResumes)
	if err != nil {
		// Keep storage and the table in step: drop the file the row never got.
		if derr := h.files.Delete(c, key); derr != nil {
			slog.Error("remove orphaned resume file", "key", key, "err", derr)
		}
		if errors.Is(err, store.ErrResumeLimit) {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("resume limit reached (%d max)", MaxResumes))
		}
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(row)
}

// renameResume godoc
//
//	@Summary	Rename a resume
//	@Tags		resumes
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int					true	"Resume ID"
//	@Param		body	body		ResumeRenameRequest	true	"New label"
//	@Success	200		{object}	domain.Resume
//	@Failure	400		{object}	ErrorResponse
//	@Failure	404		{object}	ErrorResponse
//	@Failure	415		{object}	ErrorResponse
//	@Router		/resumes/{id} [put]
func (h *handlers) renameResume(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var req ResumeRenameRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	var raw string
	if req.Name != nil {
		raw = *req.Name
	}
	name, err := resumeName("name", raw)
	if err != nil {
		return err
	}
	row, err := h.store.RenameResume(c, auth.UserID(c), id, name)
	if err != nil {
		return storeErr(err)
	}
	return c.JSON(row)
}

// downloadResume godoc
//
//	@Summary		Download a resume
//	@Description	Always an attachment named resume.pdf with nosniff, so a PDF's embedded script never runs in the app's origin and the user's label never reaches a header.
//	@Tags			resumes
//	@Produce		application/pdf
//	@Param			id	path		int	true	"Resume ID"
//	@Success		200	{file}		binary
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/resumes/{id}/file [get]
func (h *handlers) downloadResume(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	row, err := h.store.GetResume(c, auth.UserID(c), id)
	if err != nil {
		return storeErr(err)
	}
	r, err := h.files.Open(c, row.StorageKey)
	if errors.Is(err, storage.ErrNotExist) {
		return fiber.NewError(fiber.StatusNotFound, "file missing")
	}
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, "application/pdf")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="resume.pdf"`)
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	// fasthttp closes the reader once the body has been sent.
	return c.SendStream(r)
}

// deleteResume godoc
//
//	@Summary		Delete a resume
//	@Description	Removes the file first; if that fails the row is kept and the response is 500. Positions using the resume are unlinked.
//	@Tags			resumes
//	@Produce		json
//	@Param			id	path		int	true	"Resume ID"
//	@Success		200	{object}	OKResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/resumes/{id} [delete]
func (h *handlers) deleteResume(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	uid := auth.UserID(c)
	row, err := h.store.GetResume(c, uid, id)
	if err != nil {
		return storeErr(err)
	}
	if err := h.files.Delete(c, row.StorageKey); err != nil {
		slog.Error("delete resume file", "key", row.StorageKey, "err", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete file")
	}
	if err := h.store.DeleteResume(c, uid, id); err != nil {
		return storeErr(err)
	}
	return c.JSON(OKResponse{OK: true})
}

package api

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"

	"github.com/gofiber/fiber/v3"

	"github.com/caustin/apptracker/internal/auth"
	"github.com/caustin/apptracker/internal/domain"
	"github.com/caustin/apptracker/internal/storage"
	"github.com/caustin/apptracker/internal/store"
)

var unsafeNameChars = regexp.MustCompile(`[^\w.-]+`)

// safeName makes a resume label safe to use as a file name inside the zip.
func safeName(s string) string {
	s = unsafeNameChars.ReplaceAllString(s, "_")
	if len(s) > 80 {
		s = s[:80]
	}
	if s == "" || s == "." || s == ".." {
		return "resume"
	}
	return s
}

// export godoc
//
//	@Summary		Export all your data
//	@Description	A zip holding data.json (every row you own) and resumes/<id>-<name>.pdf for each stored file. Streamed, so there is no size cap.
//	@Tags			export
//	@Produce		application/zip
//	@Success		200	{file}		binary
//	@Failure		401	{object}	ErrorResponse
//	@Router			/export [get]
func (h *handlers) export(c fiber.Ctx) error {
	uid := auth.UserID(c)
	var data ExportData
	var err error
	if data.Positions, err = h.store.ListPositions(c, uid); err != nil {
		return err
	}
	if data.People, err = h.store.ListPeople(c, uid); err != nil {
		return err
	}
	if data.Events, err = h.store.ListEvents(c, uid); err != nil {
		return err
	}
	if data.Resumes, err = h.store.ListResumes(c, uid); err != nil {
		return err
	}
	switch g, err := h.store.GetGoals(c, uid); {
	case err == nil:
		data.Goals = &g
	case !errors.Is(err, store.ErrNotFound):
		return err
	}
	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, "application/zip")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="apptracker-export.zip"`)
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	files := h.files
	// The writer runs after the handler returns, so it must not touch c.
	return c.SendStreamWriter(func(w *bufio.Writer) {
		if err := writeExportZip(w, files, dataJSON, data.Resumes); err != nil {
			// Headers are already sent; the client sees a truncated zip.
			slog.Error("export stream", "err", err)
		}
	})
}

func writeExportZip(w io.Writer, files storage.Storage, dataJSON []byte, resumes []domain.Resume) error {
	zw := zip.NewWriter(w)
	f, err := zw.Create("data.json")
	if err != nil {
		return err
	}
	if _, err := f.Write(dataJSON); err != nil {
		return err
	}
	for _, r := range resumes {
		if err := addResume(zw, files, r); err != nil {
			return err
		}
	}
	return zw.Close()
}

// addResume copies one stored PDF into the zip; a missing file is skipped.
func addResume(zw *zip.Writer, files storage.Storage, r domain.Resume) error {
	src, err := files.Open(context.Background(), r.StorageKey)
	if errors.Is(err, storage.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := zw.Create(fmt.Sprintf("resumes/%d-%s.pdf", r.ID, safeName(r.Name)))
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	return err
}

package api

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/caustin/apptracker/internal/auth"
	"github.com/caustin/apptracker/internal/domain"
	"github.com/caustin/apptracker/internal/store"
)

// fields validates the body. appliedAt is the value sent; stamping happens in
// the handler, which knows the stored row.
func (r PositionRequest) fields() (domain.PositionFields, error) {
	var v validator
	f := domain.PositionFields{
		Company:     v.required("company", r.Company),
		Title:       v.required("title", r.Title),
		URL:         v.httpURL("url", r.URL),
		Location:    v.str(r.Location),
		Source:      v.str(r.Source),
		Description: v.text(r.Description),
		Impressions: v.text(r.Impressions),
		ResumeID:    v.ref("resumeId", r.ResumeID),
		SalaryMin:   v.money("salaryMin", r.SalaryMin),
		SalaryMax:   v.money("salaryMax", r.SalaryMax),
		Status:      v.enum("status", r.Status, domain.ValidStatus, domain.StatusLead),
		AppliedAt:   v.date("appliedAt", r.AppliedAt),
	}
	return f, v.err()
}

// checkPositionRefs makes sure a linked resume belongs to the same user.
func (h *handlers) checkPositionRefs(ctx context.Context, userID string, f domain.PositionFields) error {
	if f.ResumeID == nil {
		return nil
	}
	if _, err := h.store.GetResume(ctx, userID, *f.ResumeID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "resume not found")
		}
		return err
	}
	return nil
}

// listPositions godoc
//
//	@Summary	List positions
//	@Tags		positions
//	@Produce	json
//	@Success	200	{array}		domain.Position
//	@Failure	401	{object}	ErrorResponse
//	@Router		/positions [get]
func (h *handlers) listPositions(c fiber.Ctx) error {
	rows, err := h.store.ListPositions(c, auth.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(rows)
}

// getPosition godoc
//
//	@Summary	Get a position
//	@Tags		positions
//	@Produce	json
//	@Param		id	path		int	true	"Position ID"
//	@Success	200	{object}	domain.Position
//	@Failure	400	{object}	ErrorResponse
//	@Failure	404	{object}	ErrorResponse
//	@Router		/positions/{id} [get]
func (h *handlers) getPosition(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	p, err := h.store.GetPosition(c, auth.UserID(c), id)
	if err != nil {
		return storeErr(err)
	}
	return c.JSON(p)
}

// createPosition godoc
//
//	@Summary		Create a position
//	@Description	appliedAt is stamped with today's date (UTC) when the status is applied or beyond and none is sent.
//	@Tags			positions
//	@Accept			json
//	@Produce		json
//	@Param			body	body		PositionRequest	true	"Position"
//	@Success		201		{object}	domain.Position
//	@Failure		400		{object}	ErrorResponse
//	@Failure		415		{object}	ErrorResponse
//	@Router			/positions [post]
func (h *handlers) createPosition(c fiber.Ctx) error {
	var req PositionRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	f, err := req.fields()
	if err != nil {
		return err
	}
	uid := auth.UserID(c)
	if err := h.checkPositionRefs(c, uid, f); err != nil {
		return err
	}
	f.AppliedAt = domain.StampApplied(f.Status, f.AppliedAt, nil, domain.Today())
	p, err := h.store.CreatePosition(c, uid, f)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

// updatePosition godoc
//
//	@Summary		Replace a position
//	@Description	Full replace: fields not sent are cleared, except appliedAt, which keeps its stored value when omitted. It is stamped with today's date (UTC) the first time the status reaches applied or beyond.
//	@Tags			positions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Position ID"
//	@Param			body	body		PositionRequest	true	"Position"
//	@Success		200		{object}	domain.Position
//	@Failure		400		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		415		{object}	ErrorResponse
//	@Router			/positions/{id} [put]
func (h *handlers) updatePosition(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	uid := auth.UserID(c)
	existing, err := h.store.GetPosition(c, uid, id)
	if err != nil {
		return storeErr(err)
	}
	var req PositionRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	f, err := req.fields()
	if err != nil {
		return err
	}
	if err := h.checkPositionRefs(c, uid, f); err != nil {
		return err
	}
	f.AppliedAt = domain.StampApplied(f.Status, f.AppliedAt, existing.AppliedAt, domain.Today())
	p, err := h.store.UpdatePosition(c, uid, id, f)
	if err != nil {
		return storeErr(err)
	}
	return c.JSON(p)
}

// deletePosition godoc
//
//	@Summary		Delete a position
//	@Description	Also deletes the position's events.
//	@Tags			positions
//	@Produce		json
//	@Param			id	path		int	true	"Position ID"
//	@Success		200	{object}	OKResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/positions/{id} [delete]
func (h *handlers) deletePosition(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.store.DeletePosition(c, auth.UserID(c), id); err != nil {
		return storeErr(err)
	}
	return c.JSON(OKResponse{OK: true})
}

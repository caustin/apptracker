package api

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/caustin/apptracker/internal/auth"
	"github.com/caustin/apptracker/internal/domain"
	"github.com/caustin/apptracker/internal/store"
)

func (r EventRequest) fields() (domain.EventFields, error) {
	var v validator
	f := domain.EventFields{
		PositionID: v.ref("positionId", r.PositionID),
		PersonID:   v.ref("personId", r.PersonID),
		Type:       v.enum("type", r.Type, domain.ValidEventType, ""),
		Date:       v.requiredDate("date", r.Date),
		Notes:      v.text(r.Notes),
		Feedback:   v.text(r.Feedback),
		Outcome:    v.text(r.Outcome),
		FollowupOn: v.date("followupOn", r.FollowupOn),
	}
	return f, v.err()
}

// checkEventRefs makes sure a linked position and person belong to the same user.
func (h *handlers) checkEventRefs(ctx context.Context, userID string, f domain.EventFields) error {
	if f.PositionID != nil {
		if _, err := h.store.GetPosition(ctx, userID, *f.PositionID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "position not found")
			}
			return err
		}
	}
	if f.PersonID != nil {
		if _, err := h.store.GetPerson(ctx, userID, *f.PersonID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "person not found")
			}
			return err
		}
	}
	return nil
}

// listEvents godoc
//
//	@Summary	List events
//	@Tags		events
//	@Produce	json
//	@Success	200	{array}		domain.Event
//	@Failure	401	{object}	ErrorResponse
//	@Router		/events [get]
func (h *handlers) listEvents(c fiber.Ctx) error {
	rows, err := h.store.ListEvents(c, auth.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(rows)
}

// getEvent godoc
//
//	@Summary	Get an event
//	@Tags		events
//	@Produce	json
//	@Param		id	path		int	true	"Event ID"
//	@Success	200	{object}	domain.Event
//	@Failure	400	{object}	ErrorResponse
//	@Failure	404	{object}	ErrorResponse
//	@Router		/events/{id} [get]
func (h *handlers) getEvent(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	e, err := h.store.GetEvent(c, auth.UserID(c), id)
	if err != nil {
		return storeErr(err)
	}
	return c.JSON(e)
}

// createEvent godoc
//
//	@Summary	Log an event
//	@Tags		events
//	@Accept		json
//	@Produce	json
//	@Param		body	body		EventRequest	true	"Event"
//	@Success	201		{object}	domain.Event
//	@Failure	400		{object}	ErrorResponse
//	@Failure	415		{object}	ErrorResponse
//	@Router		/events [post]
func (h *handlers) createEvent(c fiber.Ctx) error {
	var req EventRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	f, err := req.fields()
	if err != nil {
		return err
	}
	uid := auth.UserID(c)
	if err := h.checkEventRefs(c, uid, f); err != nil {
		return err
	}
	e, err := h.store.CreateEvent(c, uid, f)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(e)
}

// updateEvent godoc
//
//	@Summary		Replace an event
//	@Description	Full replace: fields not sent are cleared.
//	@Tags			events
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Event ID"
//	@Param			body	body		EventRequest	true	"Event"
//	@Success		200		{object}	domain.Event
//	@Failure		400		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		415		{object}	ErrorResponse
//	@Router			/events/{id} [put]
func (h *handlers) updateEvent(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var req EventRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	f, err := req.fields()
	if err != nil {
		return err
	}
	uid := auth.UserID(c)
	if err := h.checkEventRefs(c, uid, f); err != nil {
		return err
	}
	e, err := h.store.UpdateEvent(c, uid, id, f)
	if err != nil {
		return storeErr(err)
	}
	return c.JSON(e)
}

// deleteEvent godoc
//
//	@Summary	Delete an event
//	@Tags		events
//	@Produce	json
//	@Param		id	path		int	true	"Event ID"
//	@Success	200	{object}	OKResponse
//	@Failure	400	{object}	ErrorResponse
//	@Failure	404	{object}	ErrorResponse
//	@Router		/events/{id} [delete]
func (h *handlers) deleteEvent(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.store.DeleteEvent(c, auth.UserID(c), id); err != nil {
		return storeErr(err)
	}
	return c.JSON(OKResponse{OK: true})
}

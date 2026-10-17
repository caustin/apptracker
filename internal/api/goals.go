package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/caustin/apptracker/internal/auth"
	"github.com/caustin/apptracker/internal/domain"
	"github.com/caustin/apptracker/internal/store"
)

// getGoals godoc
//
//	@Summary		Get goals
//	@Description	Returns defaults (5 applications a week, everything else null) until goals are saved.
//	@Tags			goals
//	@Produce		json
//	@Success		200	{object}	domain.Goals
//	@Failure		401	{object}	ErrorResponse
//	@Router			/goals [get]
func (h *handlers) getGoals(c fiber.Ctx) error {
	uid := auth.UserID(c)
	g, err := h.store.GetGoals(c, uid)
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(domain.DefaultGoals(uid))
	}
	if err != nil {
		return err
	}
	return c.JSON(g)
}

// putGoals godoc
//
//	@Summary		Save goals
//	@Description	Full replace (upsert). weeklyApplications must be at least 1.
//	@Tags			goals
//	@Accept			json
//	@Produce		json
//	@Param			body	body		GoalsRequest	true	"Goals"
//	@Success		200		{object}	domain.Goals
//	@Failure		400		{object}	ErrorResponse
//	@Failure		415		{object}	ErrorResponse
//	@Router			/goals [put]
func (h *handlers) putGoals(c fiber.Ctx) error {
	var req GoalsRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	var v validator
	weekly := int64(0)
	switch {
	case req.WeeklyApplications == nil:
		v.fail("weeklyApplications", "required")
	case *req.WeeklyApplications < 1:
		v.fail("weeklyApplications", "must be at least 1")
	default:
		weekly = *req.WeeklyApplications
	}
	g := domain.Goals{
		UserID:             auth.UserID(c),
		WeeklyApplications: weekly,
		SalaryMin:          v.money("salaryMin", req.SalaryMin),
		SalaryMax:          v.money("salaryMax", req.SalaryMax),
		TargetRole:         v.str(req.TargetRole),
		TargetDate:         v.date("targetDate", req.TargetDate),
		Notes:              v.text(req.Notes),
	}
	if err := v.err(); err != nil {
		return err
	}
	saved, err := h.store.UpsertGoals(c, g)
	if err != nil {
		return err
	}
	return c.JSON(saved)
}

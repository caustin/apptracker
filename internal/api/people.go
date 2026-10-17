package api

import (
	"github.com/gofiber/fiber/v3"

	"github.com/caustin/apptracker/internal/auth"
	"github.com/caustin/apptracker/internal/domain"
)

func (r PersonRequest) fields() (domain.PersonFields, error) {
	var v validator
	f := domain.PersonFields{
		Name:     v.required("name", r.Name),
		Role:     v.str("role", r.Role),
		Company:  v.str("company", r.Company),
		Email:    v.email("email", r.Email),
		Phone:    v.str("phone", r.Phone),
		LinkedIn: v.httpURL("linkedin", r.LinkedIn),
		Notes:    v.text("notes", r.Notes),
	}
	return f, v.err()
}

// listPeople godoc
//
//	@Summary	List people
//	@Tags		people
//	@Produce	json
//	@Success	200	{array}		domain.Person
//	@Failure	401	{object}	ErrorResponse
//	@Router		/people [get]
func (h *handlers) listPeople(c fiber.Ctx) error {
	rows, err := h.store.ListPeople(c, auth.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(rows)
}

// getPerson godoc
//
//	@Summary	Get a person
//	@Tags		people
//	@Produce	json
//	@Param		id	path		int	true	"Person ID"
//	@Success	200	{object}	domain.Person
//	@Failure	400	{object}	ErrorResponse
//	@Failure	404	{object}	ErrorResponse
//	@Router		/people/{id} [get]
func (h *handlers) getPerson(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	p, err := h.store.GetPerson(c, auth.UserID(c), id)
	if err != nil {
		return storeErr(err)
	}
	return c.JSON(p)
}

// createPerson godoc
//
//	@Summary	Create a person
//	@Tags		people
//	@Accept		json
//	@Produce	json
//	@Param		body	body		PersonRequest	true	"Person"
//	@Success	201		{object}	domain.Person
//	@Failure	400		{object}	ErrorResponse
//	@Failure	415		{object}	ErrorResponse
//	@Router		/people [post]
func (h *handlers) createPerson(c fiber.Ctx) error {
	var req PersonRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	f, err := req.fields()
	if err != nil {
		return err
	}
	p, err := h.store.CreatePerson(c, auth.UserID(c), f)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

// updatePerson godoc
//
//	@Summary		Replace a person
//	@Description	Full replace: fields not sent are cleared.
//	@Tags			people
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Person ID"
//	@Param			body	body		PersonRequest	true	"Person"
//	@Success		200		{object}	domain.Person
//	@Failure		400		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		415		{object}	ErrorResponse
//	@Router			/people/{id} [put]
func (h *handlers) updatePerson(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var req PersonRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	f, err := req.fields()
	if err != nil {
		return err
	}
	p, err := h.store.UpdatePerson(c, auth.UserID(c), id, f)
	if err != nil {
		return storeErr(err)
	}
	return c.JSON(p)
}

// deletePerson godoc
//
//	@Summary		Delete a person
//	@Description	Events that mention this person keep their other details; their person link is cleared.
//	@Tags			people
//	@Produce		json
//	@Param			id	path		int	true	"Person ID"
//	@Success		200	{object}	OKResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/people/{id} [delete]
func (h *handlers) deletePerson(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.store.DeletePerson(c, auth.UserID(c), id); err != nil {
		return storeErr(err)
	}
	return c.JSON(OKResponse{OK: true})
}

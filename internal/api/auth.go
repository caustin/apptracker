package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/caustin/apptracker/internal/auth"
	"github.com/caustin/apptracker/internal/store"
)

// signup godoc
//
//	@Summary		Create an account and sign in
//	@Description	Email is lowercased and must be unused. Password is 12–128 characters. Sets the session cookie.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		SignupRequest	true	"New account"
//	@Success		201		{object}	SessionResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse	"Email already registered"
//	@Failure		415		{object}	ErrorResponse
//	@Failure		429		{object}	ErrorResponse
//	@Router			/auth/signup [post]
func (h *handlers) signup(c fiber.Ctx) error {
	var req SignupRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	u, err := h.auth.SignUp(c, req.Email, req.Name, req.Password)
	switch {
	case errors.Is(err, store.ErrEmailTaken):
		return fiber.NewError(fiber.StatusConflict, "an account with that email already exists")
	case errors.Is(err, auth.ErrInvalidEmail), errors.Is(err, auth.ErrPasswordLength), errors.Is(err, auth.ErrNameTooLong):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case err != nil:
		return err
	}
	if err := h.auth.SignIn(c, u.ID); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(SessionResponse{User: u})
}

// login godoc
//
//	@Summary		Sign in
//	@Description	Starts a new session (a fresh session ID) and sets the session cookie.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginRequest	true	"Credentials"
//	@Success		200		{object}	SessionResponse
//	@Failure		401		{object}	ErrorResponse	"Invalid email or password"
//	@Failure		415		{object}	ErrorResponse
//	@Failure		429		{object}	ErrorResponse
//	@Router			/auth/login [post]
func (h *handlers) login(c fiber.Ctx) error {
	var req LoginRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	u, err := h.auth.Authenticate(c, req.Email, req.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	if err != nil {
		return err
	}
	if err := h.auth.SignIn(c, u.ID); err != nil {
		return err
	}
	return c.JSON(SessionResponse{User: u})
}

// logout godoc
//
//	@Summary	Sign out
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	OKResponse
//	@Router		/auth/logout [post]
func (h *handlers) logout(c fiber.Ctx) error {
	if err := h.auth.SignOut(c); err != nil {
		return err
	}
	return c.JSON(OKResponse{OK: true})
}

// session godoc
//
//	@Summary	Current user
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	SessionResponse
//	@Failure	401	{object}	ErrorResponse
//	@Router		/auth/session [get]
func (h *handlers) session(c fiber.Ctx) error {
	uid, err := h.auth.CurrentUserID(c)
	if err != nil {
		return err
	}
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	u, err := h.store.GetUserByID(c, uid)
	if errors.Is(err, store.ErrNotFound) {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if err != nil {
		return err
	}
	return c.JSON(SessionResponse{User: u})
}

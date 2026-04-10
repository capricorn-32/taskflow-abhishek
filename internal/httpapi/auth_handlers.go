package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"taskflow/backend/internal/repository"
)

// register godoc
// @Summary Register a new user
// @Description Creates a user account and returns a JWT token.
// @Description Validation rules: `name` and `email` are required, `password` must be at least 8 characters.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body RegisterRequest true "Registration payload"
// @Success 201 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/register [post]
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		WriteValidationError(w, map[string]string{"body": err.Error()})
		return
	}

	fields := map[string]string{}
	if strings.TrimSpace(req.Name) == "" {
		fields["name"] = "is required"
	}
	if strings.TrimSpace(req.Email) == "" {
		fields["email"] = "is required"
	}
	if len(req.Password) < 8 {
		fields["password"] = "must be at least 8 characters"
	}
	if len(fields) > 0 {
		WriteValidationError(w, fields)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		WriteInternal(w)
		return
	}

	user, err := h.store.CreateUser(r.Context(), strings.TrimSpace(req.Name), strings.ToLower(strings.TrimSpace(req.Email)), string(hash))
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			WriteValidationError(w, map[string]string{"email": "already in use"})
			return
		}
		WriteInternal(w)
		return
	}

	token, err := h.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		WriteInternal(w)
		return
	}

	WriteJSON(w, http.StatusCreated, AuthResponse{
		Token: token,
		User: UserResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

// login godoc
// @Summary Login user
// @Description Authenticates user credentials and returns a JWT token.
// @Description Validation rules: `email` and `password` are required.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body LoginRequest true "Login payload"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		WriteValidationError(w, map[string]string{"body": err.Error()})
		return
	}

	fields := map[string]string{}
	if strings.TrimSpace(req.Email) == "" {
		fields["email"] = "is required"
	}
	if strings.TrimSpace(req.Password) == "" {
		fields["password"] = "is required"
	}
	if len(fields) > 0 {
		WriteValidationError(w, fields)
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteUnauthorized(w)
			return
		}
		WriteInternal(w)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		WriteUnauthorized(w)
		return
	}

	token, err := h.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		WriteInternal(w)
		return
	}

	WriteJSON(w, http.StatusOK, AuthResponse{
		Token: token,
		User: UserResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

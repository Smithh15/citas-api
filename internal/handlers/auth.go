package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Smithh15/citas-api/internal/auth"
	"github.com/Smithh15/citas-api/internal/db/sqlc"
)

type AuthHandler struct {
	Queries   sqlc.Querier
	JWTSecret string
}

func NewAuthHandler(queries sqlc.Querier, jwtSecret string) *AuthHandler {
	return &AuthHandler{Queries: queries, JWTSecret: jwtSecret}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=patient doctor"`
}

// Register godoc
// @Summary      Registra un nuevo usuario
// @Description  Crea un paciente, doctor o admin
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Datos de registro"
// @Success      201 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not process password"})
		return
	}

	user, err := h.Queries.CreateUser(c.Request.Context(), sqlc.CreateUserParams{
		Email:        req.Email,
		PasswordHash: hash,
		FullName:     req.FullName,
		Role:         sqlc.UserRole(req.Role),
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	token, _ := auth.GenerateToken(user.ID.String(), string(user.Role), h.JWTSecret)
	c.JSON(http.StatusCreated, gin.H{"token": token})
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login godoc
// @Summary      Inicia sesión
// @Description  Valida credenciales y devuelve un JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Credenciales"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.Queries.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || !auth.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, _ := auth.GenerateToken(user.ID.String(), string(user.Role), h.JWTSecret)
	c.JSON(http.StatusOK, gin.H{"token": token})
}

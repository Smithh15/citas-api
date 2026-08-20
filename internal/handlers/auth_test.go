package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Smithh15/citas-api/internal/auth"
	"github.com/Smithh15/citas-api/internal/handlers"
	"github.com/Smithh15/citas-api/internal/db/sqlc"
)

// Mock que implementa la interfaz sqlc.Querier
type MockQuerier struct {
	mock.Mock
	sqlc.Querier // embebido para no tener que implementar TODOS los métodos, solo los que usamos
}

func (m *MockQuerier) CreateUser(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlc.User), args.Error(1)
}

func (m *MockQuerier) GetUserByEmail(ctx context.Context, email string) (sqlc.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(sqlc.User), args.Error(1)
}

func (m *MockQuerier) CreateDoctorProfile(ctx context.Context, arg sqlc.CreateDoctorProfileParams) (sqlc.DoctorProfile, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlc.DoctorProfile), args.Error(1)
}

func TestRegister_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQuerier := new(MockQuerier)
	mockQuerier.On("CreateUser", mock.Anything, mock.Anything).
		Return(sqlc.User{Email: "doc@test.com", Role: "doctor"}, nil)
	mockQuerier.On("CreateDoctorProfile", mock.Anything, mock.Anything).
		Return(sqlc.DoctorProfile{Specialty: "Cardiología"}, nil)

	h := &handlers.AuthHandler{Queries: mockQuerier, JWTSecret: "test-secret"}

	router := gin.New()
	router.POST("/auth/register", h.Register)

	body, _ := json.Marshal(map[string]string{
		"email": "doc@test.com", "password": "password123",
		"full_name": "Dra. Ana", "role": "doctor", "specialty": "Cardiología",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockQuerier.AssertExpectations(t)
}

func TestRegister_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &handlers.AuthHandler{Queries: new(MockQuerier), JWTSecret: "test-secret"}
	router := gin.New()
	router.POST("/auth/register", h.Register)

	body, _ := json.Marshal(map[string]string{
		"email": "no-es-un-email", "password": "password123",
		"full_name": "Dra. Ana", "role": "doctor",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// No debería siquiera llegar a llamar CreateUser: el binding de Gin lo rechaza antes
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_WrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hash, err := auth.HashPassword("password123")
	assert.NoError(t, err)

	mockQuerier := new(MockQuerier)
	mockQuerier.On("GetUserByEmail", mock.Anything, "doc@test.com").
		Return(sqlc.User{Email: "doc@test.com", PasswordHash: hash, Role: "doctor"}, nil)

	h := &handlers.AuthHandler{Queries: mockQuerier, JWTSecret: "test-secret"}

	router := gin.New()
	router.POST("/auth/login", h.Login)

	body, _ := json.Marshal(map[string]string{
		"email": "doc@test.com", "password": "wrong-password",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockQuerier.AssertExpectations(t)
}

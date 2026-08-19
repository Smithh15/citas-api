//go:build integration

package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/Smithh15/citas-api/internal/auth"
	"github.com/Smithh15/citas-api/internal/db/sqlc"
	"github.com/Smithh15/citas-api/internal/handlers"
)

// Valor por defecto = el DATABASE_URL documentado en .env.example para el
// Postgres local de docker-compose (no es un secreto real).
const defaultTestDatabaseURL = "postgres://citas:citas@localhost:5433/citas?sslmode=disable"

// Prueba de integración: dispara reservas concurrentes reales contra Postgres
// para demostrar que idx_unique_active_slot (el índice único parcial de
// migrations/000004) es lo que impide la doble reserva, no código Go.
// Un mock de sqlc.Querier nunca podría fallar por condición de carrera
// porque no tiene concurrencia real — por eso este test vive separado
// bajo el build tag "integration" y necesita una base de datos real.
func TestBookAppointment_ConcurrentRequestsOnlyOneWins(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultTestDatabaseURL
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	// t.Cleanup (no defer): debe cerrar el pool DESPUÉS de que
	// cleanupConcurrencyTestData lo use, y los t.Cleanup corren en orden LIFO,
	// así que registrándolo primero queda último en ejecutar.
	t.Cleanup(pool.Close)

	queries := sqlc.New(pool)
	h := &handlers.AppointmentHandler{Queries: queries}

	const numRequests = 30
	doctorID, patientIDs, slotStart := setupConcurrencyTestData(t, ctx, pool, queries, numRequests)

	gin.SetMode(gin.TestMode)
	engine := gin.New() // solo para construir el *gin.Context; no se enruta a través de él

	var wg sync.WaitGroup
	var successCount int32

	for _, patientID := range patientIDs {
		wg.Add(1)
		go func(patientID string) {
			defer wg.Done()

			body := fmt.Sprintf(`{"doctor_id":%q,"slot_start":%q}`, doctorID, slotStart)
			req := httptest.NewRequest(http.MethodPost, "/appointments", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c := gin.CreateTestContextOnly(w, engine)
			c.Request = req
			c.Set("userID", patientID)

			h.Create(c)

			if w.Code == http.StatusCreated {
				atomic.AddInt32(&successCount, 1)
			}
		}(patientID)
	}
	wg.Wait()

	require.Equal(t, int32(1), successCount, "exactamente una reserva debe ganar la condición de carrera")

	var count int
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM appointments WHERE doctor_id = $1 AND slot_start = $2 AND status = 'pending'`,
		doctorID, slotStart).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// setupConcurrencyTestData crea un doctor y numPatients pacientes distintos.
// Pacientes distintos (no uno repetido) son necesarios porque con un solo
// paciente no se puede distinguir un 409 por condición de carrera de un 409
// por "este paciente ya tiene una cita en ese horario".
func setupConcurrencyTestData(t *testing.T, ctx context.Context, pool *pgxpool.Pool, queries *sqlc.Queries, numPatients int) (doctorID string, patientIDs []string, slotStart string) {
	t.Helper()

	hash, err := auth.HashPassword("password123")
	require.NoError(t, err)

	suffix := time.Now().UnixNano()

	doctorUser, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        fmt.Sprintf("concurrency-doctor-%d@test.local", suffix),
		PasswordHash: hash,
		FullName:     "Concurrency Test Doctor",
		Role:         sqlc.UserRoleDoctor,
	})
	require.NoError(t, err)

	doctorProfile, err := queries.CreateDoctorProfile(ctx, sqlc.CreateDoctorProfileParams{
		UserID:    doctorUser.ID,
		Specialty: "Concurrencia",
	})
	require.NoError(t, err)

	patientUserIDs := make([]string, 0, numPatients)
	for i := 0; i < numPatients; i++ {
		patient, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
			Email:        fmt.Sprintf("concurrency-patient-%d-%d@test.local", suffix, i),
			PasswordHash: hash,
			FullName:     fmt.Sprintf("Concurrency Test Patient %d", i),
			Role:         sqlc.UserRolePatient,
		})
		require.NoError(t, err)
		patientUserIDs = append(patientUserIDs, patient.ID.String())
	}

	allUserIDs := append(append([]string{}, patientUserIDs...), doctorUser.ID.String())
	t.Cleanup(func() {
		cleanupConcurrencyTestData(t, ctx, pool, doctorProfile.ID.String(), allUserIDs)
	})

	// slot_start no necesita coincidir con una franja real de availability:
	// CreateAppointment inserta directo contra el índice único, no valida
	// disponibilidad — eso lo hace GetAvailableSlots, un endpoint distinto.
	return doctorProfile.ID.String(), patientUserIDs, "2035-01-08T08:00:00-05:00"
}

// cleanupConcurrencyTestData usa SQL directo porque sqlc no genera queries de
// borrado (la app en producción no las necesita) — solo hacen falta aquí para
// dejar la base limpia después del test.
func cleanupConcurrencyTestData(t *testing.T, ctx context.Context, pool *pgxpool.Pool, doctorProfileID string, userIDs []string) {
	t.Helper()

	_, err := pool.Exec(ctx, `DELETE FROM appointments WHERE doctor_id = $1`, doctorProfileID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `DELETE FROM doctor_profiles WHERE id = $1`, doctorProfileID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `DELETE FROM users WHERE id = ANY($1)`, userIDs)
	require.NoError(t, err)
}

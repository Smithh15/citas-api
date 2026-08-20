package tasks

const (
	TypeReleaseExpiredAppointments = "appointments:release_expired"
	TypeSendAppointmentReminder    = "appointments:send_reminder"
)

type SendReminderPayload struct {
	AppointmentID string `json:"appointment_id"`
}

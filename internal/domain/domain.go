// Package domain holds the row types the API returns and the shared
// status / event-type rules. JSON field names stay camelCase, as the React app
// expects.
package domain

import (
	"slices"
	"time"
)

type User struct {
	ID           string  `json:"id" example:"4f7c1b9e-2a1d-4c55-9a51-0d6f3b1e8a77"`
	Email        string  `json:"email" example:"ada@example.com"`
	Name         string  `json:"name" example:"Ada Lovelace"`
	PasswordHash *string `json:"-"`
	CreatedAt    string  `json:"createdAt" example:"2026-10-13T09:30:00Z"`
	UpdatedAt    string  `json:"updatedAt" example:"2026-10-13T09:30:00Z"`
}

type Position struct {
	ID          int64   `json:"id" example:"12"`
	UserID      string  `json:"userId"`
	Company     string  `json:"company" example:"Acme"`
	Title       string  `json:"title" example:"Staff Engineer"`
	URL         *string `json:"url" example:"https://acme.example/jobs/42"`
	Location    *string `json:"location" example:"Remote"`
	Source      *string `json:"source" example:"LinkedIn"`
	Description *string `json:"description"`
	Impressions *string `json:"impressions"`
	ResumeID    *int64  `json:"resumeId" example:"3"`
	SalaryMin   *int64  `json:"salaryMin" example:"180000"`
	SalaryMax   *int64  `json:"salaryMax" example:"220000"`
	Status      string  `json:"status" example:"applied" enums:"lead,applied,screening,interviewing,offer,accepted,rejected,withdrawn"`
	AppliedAt   *string `json:"appliedAt" example:"2026-10-13"`
	CreatedAt   string  `json:"createdAt" example:"2026-10-12"`
}

// PositionFields are the writable columns of a position, already validated.
type PositionFields struct {
	Company     string
	Title       string
	URL         *string
	Location    *string
	Source      *string
	Description *string
	Impressions *string
	ResumeID    *int64
	SalaryMin   *int64
	SalaryMax   *int64
	Status      string
	AppliedAt   *string
}

type Person struct {
	ID        int64   `json:"id" example:"7"`
	UserID    string  `json:"userId"`
	Name      string  `json:"name" example:"Grace Hopper"`
	Role      *string `json:"role" example:"Recruiter"`
	Company   *string `json:"company" example:"Acme"`
	Email     *string `json:"email" example:"grace@acme.example"`
	Phone     *string `json:"phone"`
	LinkedIn  *string `json:"linkedin" example:"https://www.linkedin.com/in/grace"`
	Notes     *string `json:"notes"`
	CreatedAt string  `json:"createdAt" example:"2026-10-12"`
}

type PersonFields struct {
	Name     string
	Role     *string
	Company  *string
	Email    *string
	Phone    *string
	LinkedIn *string
	Notes    *string
}

// Event is an interaction: an email, call, interview, follow-up and so on.
type Event struct {
	ID         int64   `json:"id" example:"31"`
	UserID     string  `json:"userId"`
	PositionID *int64  `json:"positionId" example:"12"`
	PersonID   *int64  `json:"personId" example:"7"`
	Type       string  `json:"type" example:"recruiter_call" enums:"email,recruiter_call,interview,followup,meeting,note"`
	Date       string  `json:"date" example:"2026-10-13"`
	Notes      *string `json:"notes"`
	Feedback   *string `json:"feedback"`
	Outcome    *string `json:"outcome"`
	FollowupOn *string `json:"followupOn" example:"2026-10-20"`
	CreatedAt  string  `json:"createdAt" example:"2026-10-13"`
}

type EventFields struct {
	PositionID *int64
	PersonID   *int64
	Type       string
	Date       string
	Notes      *string
	Feedback   *string
	Outcome    *string
	FollowupOn *string
}

type Resume struct {
	ID     int64  `json:"id" example:"3"`
	UserID string `json:"userId"`
	Name   string `json:"name" example:"Platform-focused v3"`
	// StorageKey locates the file in Storage; never sent to the browser.
	StorageKey string `json:"-"`
	CreatedAt  string `json:"createdAt" example:"2026-10-12"`
}

type Goals struct {
	UserID             string  `json:"userId"`
	WeeklyApplications int64   `json:"weeklyApplications" example:"5"`
	SalaryMin          *int64  `json:"salaryMin" example:"180000"`
	SalaryMax          *int64  `json:"salaryMax" example:"220000"`
	TargetRole         *string `json:"targetRole" example:"Staff Engineer"`
	TargetDate         *string `json:"targetDate" example:"2026-12-31"`
	Notes              *string `json:"notes"`
}

const DefaultWeeklyApplications = 5

// DefaultGoals is what GET /goals returns before the user saves any.
func DefaultGoals(userID string) Goals {
	return Goals{UserID: userID, WeeklyApplications: DefaultWeeklyApplications}
}

// Statuses in the order the UI lists them. rejected and withdrawn are
// terminal side-exits.
var Statuses = []string{"lead", "applied", "screening", "interviewing", "offer", "accepted", "rejected", "withdrawn"}

var EventTypes = []string{"email", "recruiter_call", "interview", "followup", "meeting", "note"}

const StatusLead = "lead"

func ValidStatus(s string) bool    { return slices.Contains(Statuses, s) }
func ValidEventType(t string) bool { return slices.Contains(EventTypes, t) }

// IsAppliedOrBeyond reports whether a position in this status has been
// applied for, which is when appliedAt gets stamped.
func IsAppliedOrBeyond(status string) bool {
	switch status {
	case "applied", "screening", "interviewing", "offer", "accepted":
		return true
	}
	return false
}

// StampApplied returns the appliedAt a saved position should have: the value
// sent, else the one already stored, else today if the status is applied or
// beyond. An existing value is never overwritten by the stamp.
func StampApplied(status string, sent, existing *string, today string) *string {
	if sent != nil {
		return sent
	}
	if existing != nil {
		return existing
	}
	if IsAppliedOrBeyond(status) {
		return &today
	}
	return nil
}

// Today is the server's current date, YYYY-MM-DD in UTC.
func Today() string { return time.Now().UTC().Format(time.DateOnly) }

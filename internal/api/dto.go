package api

import "github.com/caustin/apptracker/internal/domain"

// Request bodies. Fields are pointers so a missing key and an explicit null
// both read as "not set". PUT is a full replace: anything not sent is cleared.

type ErrorResponse struct {
	Error string `json:"error" example:"not found"`
}

type OKResponse struct {
	OK bool `json:"ok" example:"true"`
}

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

type SignupRequest struct {
	Email    string `json:"email" example:"ada@example.com"`
	Name     string `json:"name" example:"Ada Lovelace"`
	Password string `json:"password" example:"correct horse battery staple" minLength:"12" maxLength:"128"`
}

type LoginRequest struct {
	Email    string `json:"email" example:"ada@example.com"`
	Password string `json:"password" example:"correct horse battery staple"`
}

type SessionResponse struct {
	User domain.User `json:"user"`
}

type PositionRequest struct {
	Company     *string `json:"company" example:"Acme" validate:"required"`
	Title       *string `json:"title" example:"Staff Engineer" validate:"required"`
	URL         *string `json:"url" example:"https://acme.example/jobs/42"`
	Location    *string `json:"location" example:"Remote"`
	Source      *string `json:"source" example:"LinkedIn"`
	Description *string `json:"description"`
	Impressions *string `json:"impressions"`
	ResumeID    *int64  `json:"resumeId" example:"3"`
	SalaryMin   *int64  `json:"salaryMin" example:"180000"`
	SalaryMax   *int64  `json:"salaryMax" example:"220000"`
	// Defaults to "lead".
	Status *string `json:"status" example:"applied" enums:"lead,applied,screening,interviewing,offer,accepted,rejected,withdrawn"`
	// Stamped with today's date (UTC) the first time status reaches applied or
	// beyond. An existing value is kept when this is omitted.
	AppliedAt *string `json:"appliedAt" example:"2026-10-13" format:"date"`
}

type PersonRequest struct {
	Name     *string `json:"name" example:"Grace Hopper" validate:"required"`
	Role     *string `json:"role" example:"Recruiter"`
	Company  *string `json:"company" example:"Acme"`
	Email    *string `json:"email" example:"grace@acme.example"`
	Phone    *string `json:"phone"`
	LinkedIn *string `json:"linkedin" example:"https://www.linkedin.com/in/grace"`
	Notes    *string `json:"notes"`
}

type EventRequest struct {
	PositionID *int64  `json:"positionId" example:"12"`
	PersonID   *int64  `json:"personId" example:"7"`
	Type       *string `json:"type" example:"recruiter_call" enums:"email,recruiter_call,interview,followup,meeting,note" validate:"required"`
	Date       *string `json:"date" example:"2026-10-13" format:"date" validate:"required"`
	Notes      *string `json:"notes"`
	Feedback   *string `json:"feedback"`
	Outcome    *string `json:"outcome"`
	FollowupOn *string `json:"followupOn" example:"2026-10-20" format:"date"`
}

type ResumeRenameRequest struct {
	Name *string `json:"name" example:"Platform-focused v4" validate:"required"`
}

type GoalsRequest struct {
	WeeklyApplications *int64  `json:"weeklyApplications" example:"5" minimum:"1" validate:"required"`
	SalaryMin          *int64  `json:"salaryMin" example:"180000"`
	SalaryMax          *int64  `json:"salaryMax" example:"220000"`
	TargetRole         *string `json:"targetRole" example:"Staff Engineer"`
	TargetDate         *string `json:"targetDate" example:"2026-12-31" format:"date"`
	Notes              *string `json:"notes"`
}

// ExportData is data.json inside the export zip.
type ExportData struct {
	Positions []domain.Position `json:"positions"`
	People    []domain.Person   `json:"people"`
	Events    []domain.Event    `json:"events"`
	Resumes   []domain.Resume   `json:"resumes"`
	Goals     *domain.Goals     `json:"goals"`
}

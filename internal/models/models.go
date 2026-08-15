package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RolePatient      Role = "PATIENT"
	RoleDoctor       Role = "DOCTOR"
	RolePharmacist   Role = "PHARMACIST"
	RoleNurse        Role = "NURSE"
	RoleLabScientist Role = "LAB_SCIENTIST"
	RoleAdmin        Role = "ADMIN"
)

type PostCategory string

const (
	CategoryGeneral       PostCategory = "GENERAL"
	CategoryDrugRequest   PostCategory = "DRUG_REQUEST"
	CategoryMedicalAdvice PostCategory = "MEDICAL_ADVICE"
	CategoryAnnouncement  PostCategory = "ANNOUNCEMENT"
)

type PostStatus string

const (
	PostStatusOpen     PostStatus = "OPEN"
	PostStatusAnswered PostStatus = "ANSWERED"
	PostStatusClosed   PostStatus = "CLOSED"
)

// User represents a user in the system
type User struct {
	ID            string    `json:"id" bson:"_id"`
	Name          string    `json:"name" bson:"name"`
	Email         string    `json:"email" bson:"email"`
	PasswordHash  string    `json:"-" bson:"password_hash"` // Never serialized to JSON
	Role          Role      `json:"role" bson:"role"`
	Phone         *string   `json:"phone,omitempty" bson:"phone,omitempty"`
	Address       *string   `json:"address,omitempty" bson:"address,omitempty"`
	Latitude      *float64  `json:"latitude,omitempty" bson:"latitude,omitempty"`
	Longitude     *float64  `json:"longitude,omitempty" bson:"longitude,omitempty"`
	Specialty     *string   `json:"specialty,omitempty" bson:"specialty,omitempty"`
	LicenseNumber *string   `json:"license_number,omitempty" bson:"license_number,omitempty"`
	IsVerified    bool      `json:"is_verified" bson:"is_verified"`
	IsAvailable   bool      `json:"is_available" bson:"is_available"`
	AvatarURL     *string   `json:"avatar_url,omitempty" bson:"avatar_url,omitempty"`
	CreatedAt     time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" bson:"updated_at"`
}

// NewUser creates a new user with generated ID and timestamps
func NewUser(name, email, passwordHash string, role Role) *User {
	now := time.Now()
	return &User{
		ID:           uuid.New().String(),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		IsVerified:   false,
		IsAvailable:  true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

type ChatRoom struct {
	ID             string    `json:"id"`
	PatientID      string    `json:"patient_id"`
	ProfessionalID string    `json:"professional_id"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
	Sender    *User     `json:"sender,omitempty"`
}

type Booking struct {
	ID             string    `json:"id"`
	PatientID      string    `json:"patient_id"`
	ProfessionalID string    `json:"professional_id"`
	RoomID         *string   `json:"room_id,omitempty"`
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	ScheduledAt    *string   `json:"scheduled_at,omitempty"`
	Notes          *string   `json:"notes,omitempty"`
	Address        *string   `json:"address,omitempty"`
	Latitude       *float64  `json:"latitude,omitempty"`
	Longitude      *float64  `json:"longitude,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	Patient        *User     `json:"patient,omitempty"`
	Professional   *User     `json:"professional,omitempty"`
	Room           *ChatRoom `json:"room,omitempty"`
}

type MatchRequest struct {
	ID            string    `json:"id"`
	PatientID     string    `json:"patient_id"`
	Specialty     string    `json:"specialty"`
	BookingType   string    `json:"booking_type"`
	Urgency       string    `json:"urgency"`
	Description   *string   `json:"description,omitempty"`
	AssignedProID *string   `json:"assigned_pro_id,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type AmbulanceDispatch struct {
	ID               string    `json:"id" bson:"_id"`
	PatientID        string    `json:"patient_id" bson:"patient_id"`
	Latitude         float64   `json:"latitude" bson:"latitude"`
	Longitude        float64   `json:"longitude" bson:"longitude"`
	Address          *string   `json:"address,omitempty" bson:"address,omitempty"`
	Severity         string    `json:"severity" bson:"severity"`
	Status           string    `json:"status" bson:"status"`
	DispatchedAt     time.Time `json:"dispatched_at" bson:"dispatched_at"`
	EstimatedArrival *string   `json:"estimated_arrival,omitempty" bson:"estimated_arrival,omitempty"`
	CompletedAt      *string   `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	Notes            *string   `json:"notes,omitempty" bson:"notes,omitempty"`
	Patient          *User     `json:"patient,omitempty" bson:"-"`
}

// --- Request / Response DTOs ---

// ProfileRequest is submitted (via CompleteProfile) right after Supabase Auth
// signup succeeds on the frontend. It captures the app-specific fields
// Supabase's own auth.users table doesn't store — name, app role, and the
// professional-only fields. Replaces the old RegisterRequest.
type ProfileRequest struct {
	Name          string `json:"name"`
	Role          string `json:"role"`
	Phone         string `json:"phone,omitempty"`
	Specialty     string `json:"specialty,omitempty"`
	LicenseNumber string `json:"licenseNumber,omitempty"`
}

type MatchingRequest struct {
	BookingType string   `json:"bookingType"`
	Specialty   string   `json:"specialty,omitempty"`
	Urgency     string   `json:"urgency,omitempty"`
	Description string   `json:"description,omitempty"`
	Address     string   `json:"address,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
}

type AmbulanceRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
	Severity  string  `json:"severity"`
	Notes     string  `json:"notes,omitempty"`
}

type BookingCreateRequest struct {
	ProfessionalID string   `json:"professionalId"`
	Type           string   `json:"type"`
	ScheduledAt    string   `json:"scheduledAt,omitempty"`
	Notes          string   `json:"notes,omitempty"`
	Address        string   `json:"address,omitempty"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`
}

type BookingUpdateRequest struct {
	BookingID string `json:"bookingId"`
	Status    string `json:"status"`
}

// ========== FORUM MODELS ==========

// Post represents a forum post
// Can be a question, drug request, medical advice, or announcement
type Post struct {
	ID         string       `json:"id" bson:"_id"`
	AuthorID   string       `json:"author_id" bson:"author_id"`
	Title      string       `json:"title" bson:"title"`
	Content    string       `json:"content" bson:"content"`
	Category   PostCategory `json:"category" bson:"category"`
	Status     PostStatus   `json:"status" bson:"status"`
	Tags       []string     `json:"tags" bson:"tags"`
	DrugName   *string      `json:"drug_name,omitempty" bson:"drug_name,omitempty"` // For DRUG_REQUEST
	IsPinned   bool         `json:"is_pinned" bson:"is_pinned"`
	IsLocked   bool         `json:"is_locked" bson:"is_locked"`
	ViewCount  int          `json:"view_count" bson:"view_count"`
	ReplyCount int          `json:"reply_count" bson:"reply_count"`
	CreatedAt  time.Time    `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at" bson:"updated_at"`
	Author     *User        `json:"author,omitempty" bson:"-"`
}

// NewPost creates a new forum post
func NewPost(authorID, title, content string, category PostCategory, tags []string, drugName *string) *Post {
	now := time.Now()
	return &Post{
		ID:         uuid.New().String(),
		AuthorID:   authorID,
		Title:      title,
		Content:    content,
		Category:   category,
		Status:     PostStatusOpen,
		Tags:       tags,
		DrugName:   drugName,
		ViewCount:  0,
		ReplyCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Comment represents a reply to a forum post
type Comment struct {
	ID         string    `json:"id" bson:"_id"`
	PostID     string    `json:"post_id" bson:"post_id"`
	AuthorID   string    `json:"author_id" bson:"author_id"`
	Content    string    `json:"content" bson:"content"`
	IsAccepted bool      `json:"is_accepted" bson:"is_accepted"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" bson:"updated_at"`
	Author     *User     `json:"author,omitempty" bson:"-"`
}

// NewComment creates a new comment
func NewComment(postID, authorID, content string) *Comment {
	now := time.Now()
	return &Comment{
		ID:         uuid.New().String(),
		PostID:     postID,
		AuthorID:   authorID,
		Content:    content,
		IsAccepted: false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// ========== DTOs ==========

// Auth DTOs
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

// Forum DTOs
type CreatePostRequest struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	DrugName string   `json:"drug_name,omitempty"`
}

type UpdatePostRequest struct {
	Title    string   `json:"title,omitempty"`
	Content  string   `json:"content,omitempty"`
	Category string   `json:"category,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	DrugName string   `json:"drug_name,omitempty"`
}

type CreateCommentRequest struct {
	Content string `json:"content"`
}

type PostListQuery struct {
	Category  string
	Status    string
	AuthorID  string
	Search    string
	Page      int
	Limit     int
	SortBy    string // created_at, view_count, reply_count
	SortOrder string // asc, desc
}

// Admin DTOs
type AdminUserUpdateRequest struct {
	Name          string `json:"name,omitempty"`
	Role          string `json:"role,omitempty"`
	IsVerified    *bool  `json:"is_verified,omitempty"`
	IsAvailable   *bool  `json:"is_available,omitempty"`
	Specialty     string `json:"specialty,omitempty"`
	LicenseNumber string `json:"license_number,omitempty"`
}

type AdminStatsResponse struct {
	TotalUsers         int `json:"total_users"`
	TotalPatients      int `json:"total_patients"`
	TotalDoctors       int `json:"total_doctors"`
	TotalPharmacists   int `json:"total_pharmacists"`
	TotalNurses        int `json:"total_nurses"`
	TotalLabScientists int `json:"total_lab_scientists"`
	TotalAdmins        int `json:"total_admins"`
	TotalPosts         int `json:"total_posts"`
	TotalBookings      int `json:"total_bookings"`
	TotalAmbulance     int `json:"total_ambulance"`
}

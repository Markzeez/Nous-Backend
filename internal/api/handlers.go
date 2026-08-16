package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"medcon/internal/auth"
	"medcon/internal/models"
	"medcon/internal/service"
)

type Handler struct {
	AuthService      *service.AuthService
	UserService      *service.UserService
	PostService      *service.PostService
	CommentService   *service.CommentService
	BookingService   *service.BookingService
	MatchService     *service.MatchService
	ChatService      *service.ChatService
	AmbulanceService *service.AmbulanceService
	VitalsService    *service.VitalsService
	AdminService     *service.AdminService
	JWTSecret        string
}

// ── Auth ──

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" || req.Role == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "name, email, password, and role are required")
		return
	}

	resp, err := h.AuthService.Register(r.Context(), &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	auth.JSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "email and password are required")
		return
	}

	resp, err := h.AuthService.Login(r.Context(), &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, err.Error())
		return
	}

	auth.JSON(w, http.StatusOK, resp)
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	token, err := h.AuthService.RefreshToken(r.Context(), claims.UserID())
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to refresh token")
		return
	}

	auth.JSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.AuthService.GetUser(r.Context(), claims.UserID())
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		auth.ErrorJSON(w, http.StatusNotFound, "User not found")
		return
	}

	auth.JSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.UserService.UpdateProfile(r.Context(), claims.UserID(), req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	auth.JSON(w, http.StatusOK, user)
}

// ── Admin ──

func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.AdminService.GetStats(r.Context())
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}
	auth.JSON(w, http.StatusOK, stats)
}

func (h *Handler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	users, total, err := h.AdminService.ListUsers(r.Context(), page, limit)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to list users")
		return
	}

	auth.JSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) AdminGetUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user, err := h.AdminService.GetUser(r.Context(), id)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		auth.ErrorJSON(w, http.StatusNotFound, "User not found")
		return
	}
	auth.JSON(w, http.StatusOK, user)
}

func (h *Handler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req models.AdminUserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.AdminService.UpdateUser(r.Context(), id, &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to update user")
		return
	}
	auth.JSON(w, http.StatusOK, user)
}

func (h *Handler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.AdminService.DeleteUser(r.Context(), id); err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}
	auth.JSON(w, http.StatusOK, map[string]string{"message": "User deleted"})
}

func (h *Handler) AdminListPosts(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	posts, total, err := h.AdminService.ListPosts(r.Context(), page, limit)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to list posts")
		return
	}

	auth.JSON(w, http.StatusOK, map[string]interface{}{
		"posts": posts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) AdminUpdatePost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	post, err := h.AdminService.UpdatePost(r.Context(), id, req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to update post")
		return
	}
	auth.JSON(w, http.StatusOK, post)
}

func (h *Handler) AdminDeletePost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.AdminService.DeletePost(r.Context(), id); err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to delete post")
		return
	}
	auth.JSON(w, http.StatusOK, map[string]string{"message": "Post deleted"})
}

// ── Forum ──

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" || req.Content == "" || req.Category == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "title, content, and category are required")
		return
	}

	post, err := h.PostService.CreatePost(r.Context(), claims.UserID(), &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JSON(w, http.StatusCreated, post)
}

func (h *Handler) ListPosts(w http.ResponseWriter, r *http.Request) {
	query := &models.PostListQuery{
		Category:  r.URL.Query().Get("category"),
		Status:    r.URL.Query().Get("status"),
		AuthorID:  r.URL.Query().Get("author_id"),
		Search:    r.URL.Query().Get("search"),
		SortBy:    r.URL.Query().Get("sort_by"),
		SortOrder: r.URL.Query().Get("sort_order"),
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	query.Page = page

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query.Limit = limit

	posts, total, err := h.PostService.ListPosts(r.Context(), query)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to list posts")
		return
	}

	auth.JSON(w, http.StatusOK, map[string]interface{}{
		"posts": posts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	post, err := h.PostService.GetPost(r.Context(), id)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to get post")
		return
	}
	if post == nil {
		auth.ErrorJSON(w, http.StatusNotFound, "Post not found")
		return
	}

	// Increment view count
	h.PostService.IncrementViewCount(r.Context(), id)

	auth.JSON(w, http.StatusOK, post)
}

func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	var req models.UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	post, err := h.PostService.UpdatePost(r.Context(), id, claims.UserID(), &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JSON(w, http.StatusOK, post)
}

func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	// Check if admin
	isAdmin := claims.AppRole() == "ADMIN"

	if err := h.PostService.DeletePost(r.Context(), id, claims.UserID(), isAdmin); err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JSON(w, http.StatusOK, map[string]string{"message": "Post deleted"})
}

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	var req models.CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Content == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "content is required")
		return
	}

	comment, err := h.CommentService.CreateComment(r.Context(), id, claims.UserID(), &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JSON(w, http.StatusCreated, comment)
}

func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	comments, total, err := h.CommentService.ListComments(r.Context(), id, page, limit)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to list comments")
		return
	}

	auth.JSON(w, http.StatusOK, map[string]interface{}{
		"comments": comments,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

func (h *Handler) AcceptComment(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	comment, err := h.CommentService.AcceptComment(r.Context(), id, claims.UserID())
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JSON(w, http.StatusOK, comment)
}

func (h *Handler) IncrementPostView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.PostService.IncrementViewCount(r.Context(), id); err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to increment view count")
		return
	}
	auth.JSON(w, http.StatusOK, map[string]string{"message": "View count incremented"})
}

// ── Matching ──

func (h *Handler) CreateMatch(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil || claims.AppRole() != string(models.RolePatient) {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Only patients can create matches")
		return
	}

	var req models.MatchingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.BookingType == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "bookingType is required")
		return
	}

	result, err := h.MatchService.CreateMatch(r.Context(), claims.UserID(), &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JSON(w, http.StatusOK, result)
}

func (h *Handler) ListProfessionals(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	role := r.URL.Query().Get("role")
	specialty := r.URL.Query().Get("specialty")

	professionals, err := h.UserService.GetVerifiedProfessionals(r.Context(), role, specialty)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to get professionals")
		return
	}

	auth.JSON(w, http.StatusOK, professionals)
}

// ── Bookings ──

func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil || claims.AppRole() != string(models.RolePatient) {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Only patients can create bookings")
		return
	}

	var req models.BookingCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ProfessionalID == "" || req.Type == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "professionalId and type are required")
		return
	}

	booking, err := h.BookingService.CreateBooking(r.Context(), claims.UserID(), &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JSON(w, http.StatusCreated, booking)
}

func (h *Handler) ListBookings(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	isPatient := claims.AppRole() == string(models.RolePatient)
	bookings, total, err := h.BookingService.ListBookings(r.Context(), claims.UserID(), isPatient, page, limit)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to list bookings")
		return
	}

	auth.JSON(w, http.StatusOK, map[string]interface{}{
		"bookings": bookings,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

func (h *Handler) UpdateBooking(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.BookingUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.BookingID == "" || req.Status == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "bookingId and status are required")
		return
	}

	booking, err := h.BookingService.UpdateBooking(r.Context(), req.BookingID, claims.UserID(), req.Status)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JSON(w, http.StatusOK, booking)
}

// ── Chat ──

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	_ = r.PathValue("roomId")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// This would use messageRepo - for now return empty
	auth.JSON(w, http.StatusOK, map[string]interface{}{
		"messages": []interface{}{},
		"total":    0,
		"page":     page,
		"limit":    limit,
	})
}

func (h *Handler) PostMessage(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	_ = r.PathValue("roomId")
	var body struct {
		Content string `json:"content"`
		Type    string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if body.Content == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "content is required")
		return
	}
	if body.Type == "" {
		body.Type = "text"
	}

	// This would use messageRepo - for now return success
	auth.JSON(w, http.StatusCreated, map[string]string{"message": "Message sent"})
}

// ── Ambulance ──

func (h *Handler) DispatchAmbulance(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.AmbulanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Latitude == 0 || req.Longitude == 0 || req.Severity == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "latitude, longitude, and severity are required")
		return
	}

	dispatch, err := h.AmbulanceService.DispatchAmbulance(r.Context(), claims.UserID(), &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JSON(w, http.StatusCreated, dispatch)
}

func (h *Handler) ListDispatches(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	isAdmin := claims.AppRole() == string(models.RoleAdmin)
	var dispatches []*models.AmbulanceDispatch
	var total int64
	var err error

	if isAdmin {
		dispatches, total, err = h.AmbulanceService.ListAllDispatches(r.Context(), page, limit)
	} else {
		dispatches, total, err = h.AmbulanceService.ListDispatches(r.Context(), claims.UserID(), page, limit)
	}

	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to list dispatches")
		return
	}

	auth.JSON(w, http.StatusOK, map[string]interface{}{
		"dispatches": dispatches,
		"total":      total,
		"page":       page,
		"limit":      limit,
	})
}

func (h *Handler) UpdateDispatch(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil || claims.AppRole() != string(models.RoleAdmin) {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Only admins can update dispatches")
		return
	}

	var body struct {
		DispatchID string `json:"dispatchId"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if body.DispatchID == "" || body.Status == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "dispatchId and status are required")
		return
	}

	dispatch, err := h.AmbulanceService.UpdateDispatch(r.Context(), body.DispatchID, body.Status)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JSON(w, http.StatusOK, dispatch)
}

// RequestPasswordReset sends a password reset email
func (h *Handler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Email == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "Email is required")
		return
	}

	// Always return success to prevent email enumeration
	if err := h.AuthService.RequestPasswordReset(r.Context(), req.Email); err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to process request")
		return
	}
	auth.JSON(w, http.StatusOK, map[string]string{"message": "If the email exists, a reset link has been sent"})
}

// ResetPassword validates token and updates password
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Token == "" || req.NewPassword == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "Token and new password are required")
		return
	}
	if len(req.NewPassword) < 8 {
		auth.ErrorJSON(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	if err := h.AuthService.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	auth.JSON(w, http.StatusOK, map[string]string{"message": "Password has been reset successfully"})
}

// ── Vitals ──

func (h *Handler) CreateVitals(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.VitalsCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.PatientID == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "Patient ID is required")
		return
	}

	// Validate pain level range
	if req.PainLevel != nil && (*req.PainLevel < 0 || *req.PainLevel > 10) {
		auth.ErrorJSON(w, http.StatusBadRequest, "Pain level must be between 0 and 10")
		return
	}

	// Validate oxygen saturation range
	if req.OxygenSaturation != nil && (*req.OxygenSaturation < 0 || *req.OxygenSaturation > 100) {
		auth.ErrorJSON(w, http.StatusBadRequest, "Oxygen saturation must be between 0 and 100")
		return
	}

	// Validate blood pressure ranges
	if req.BloodPressureSystolic != nil && (*req.BloodPressureSystolic < 30 || *req.BloodPressureSystolic > 300) {
		auth.ErrorJSON(w, http.StatusBadRequest, "Systolic blood pressure must be between 30 and 300")
		return
	}
	if req.BloodPressureDiastolic != nil && (*req.BloodPressureDiastolic < 20 || *req.BloodPressureDiastolic > 200) {
		auth.ErrorJSON(w, http.StatusBadRequest, "Diastolic blood pressure must be between 20 and 200")
		return
	}

	vitals, err := h.VitalsService.CreateVitals(r.Context(), claims.UserID(), &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	auth.JSON(w, http.StatusCreated, vitals)
}

func (h *Handler) GetVitals(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "Vitals ID is required")
		return
	}

	vitals, err := h.VitalsService.GetVitals(r.Context(), id)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to get vitals")
		return
	}
	if vitals == nil {
		auth.ErrorJSON(w, http.StatusNotFound, "Vitals record not found")
		return
	}

	auth.JSON(w, http.StatusOK, vitals)
}

func (h *Handler) UpdateVitals(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "Vitals ID is required")
		return
	}

	var req models.VitalsUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate pain level range
	if req.PainLevel != nil && (*req.PainLevel < 0 || *req.PainLevel > 10) {
		auth.ErrorJSON(w, http.StatusBadRequest, "Pain level must be between 0 and 10")
		return
	}

	// Validate oxygen saturation range
	if req.OxygenSaturation != nil && (*req.OxygenSaturation < 0 || *req.OxygenSaturation > 100) {
		auth.ErrorJSON(w, http.StatusBadRequest, "Oxygen saturation must be between 0 and 100")
		return
	}

	vitals, err := h.VitalsService.UpdateVitals(r.Context(), id, &req)
	if err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	auth.JSON(w, http.StatusOK, vitals)
}

func (h *Handler) DeleteVitals(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "Vitals ID is required")
		return
	}

	if err := h.VitalsService.DeleteVitals(r.Context(), id); err != nil {
		auth.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	auth.JSON(w, http.StatusOK, map[string]string{"message": "Vitals record deleted successfully"})
}

func (h *Handler) ListVitals(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	patientID := r.URL.Query().Get("patient_id")
	if patientID == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "Patient ID is required")
		return
	}

	page := 1
	limit := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	sortBy := r.URL.Query().Get("sort_by")
	if sortBy == "" {
		sortBy = "recorded_at"
	}
	sortOrder := r.URL.Query().Get("sort_order")
	if sortOrder == "" {
		sortOrder = "desc"
	}

	var dateFrom, dateTo *time.Time
	if df := r.URL.Query().Get("date_from"); df != "" {
		if parsed, err := time.Parse(time.RFC3339, df); err == nil {
			dateFrom = &parsed
		}
	}
	if dt := r.URL.Query().Get("date_to"); dt != "" {
		if parsed, err := time.Parse(time.RFC3339, dt); err == nil {
			dateTo = &parsed
		}
	}

	query := &models.VitalsListQuery{
		PatientID: patientID,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		DateFrom:  dateFrom,
		DateTo:    dateTo,
	}

	response, err := h.VitalsService.ListVitals(r.Context(), query)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to list vitals")
		return
	}

	auth.JSON(w, http.StatusOK, response)
}

func (h *Handler) GetLatestVitals(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r)
	if claims == nil {
		auth.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	patientID := r.URL.Query().Get("patient_id")
	if patientID == "" {
		auth.ErrorJSON(w, http.StatusBadRequest, "Patient ID is required")
		return
	}

	vitals, err := h.VitalsService.GetLatestVitals(r.Context(), patientID)
	if err != nil {
		auth.ErrorJSON(w, http.StatusInternalServerError, "Failed to get latest vitals")
		return
	}

	auth.JSON(w, http.StatusOK, vitals)
}

package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/pacahar/pr-reviewer-assignment/internal/models"
	"github.com/pacahar/pr-reviewer-assignment/internal/storage"
	storageErrors "github.com/pacahar/pr-reviewer-assignment/internal/storage/errors"
)

type Handler struct {
	Storage *storage.Storage
	Log     *slog.Logger
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /team/add", h.CreateTeam)
	mux.HandleFunc("GET /team/get", h.GetTeam)

	mux.HandleFunc("POST /users/setIsActive", h.SetUserActive)
	mux.HandleFunc("GET /users/getReview", h.GetUserReviews)

	mux.HandleFunc("POST /pullRequest/create", h.CreatePullRequest)
	mux.HandleFunc("POST /pullRequest/merge", h.MergePullRequest)
	mux.HandleFunc("POST /pullRequest/reassign", h.ReassignReviewer)

}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamName string `json:"team_name"`
		Members  []struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
			IsActive bool   `json:"is_active"`
		} `json:"members"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}

	if req.TeamName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}

	_, err := h.Storage.TeamStorage.GetTeamByName(r.Context(), req.TeamName)
	if err == nil {
		writeError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
		return
	}
	if err != nil && !errors.Is(err, storageErrors.ErrTeamNotFound) {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	if err := h.Storage.TeamStorage.CreateTeam(r.Context(), req.TeamName); err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	for _, m := range req.Members {

		_, err := h.Storage.UserStorage.GetUserByID(r.Context(), m.UserID)

		switch {
		case errors.Is(err, storageErrors.ErrUserNotFound):
			if err := h.Storage.UserStorage.CreateUser(
				r.Context(), m.UserID, m.Username, req.TeamName,
			); err != nil {
				writeError(w, 500, "UNKNOWN", err.Error())
				return
			}

		case err == nil:
			if err := h.Storage.UserStorage.SetUserTeam(
				r.Context(), m.UserID, req.TeamName,
			); err != nil {
				writeError(w, 500, "UNKNOWN", err.Error())
				return
			}

		default:
			writeError(w, 500, "UNKNOWN", err.Error())
			return
		}

		if err := h.Storage.UserStorage.SetUserActiveStatus(
			r.Context(), m.UserID, m.IsActive,
		); err != nil {
			writeError(w, 500, "UNKNOWN", err.Error())
			return
		}
	}

	resp := map[string]any{
		"team": map[string]any{
			"team_name": req.TeamName,
			"members":   req.Members,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")

	if teamName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}

	team, err := h.Storage.TeamStorage.GetTeamByName(r.Context(), teamName)
	if errors.Is(err, storageErrors.ErrTeamNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "team not found")
		return
	}
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	users, err := h.Storage.TeamStorage.GetUsersByTeam(r.Context(), teamName)
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	resp := map[string]any{
		"team_name": team.TeamName,
		"members":   []any{},
	}

	members := make([]any, 0, len(users))
	for _, u := range users {
		members = append(members, map[string]any{
			"user_id":   u.UserID,
			"username":  u.Username,
			"is_active": u.IsActive,
		})
	}

	resp["members"] = members

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}

	ctx := r.Context()

	_, err := h.Storage.UserStorage.GetUserByID(ctx, req.UserID)
	if errors.Is(err, storageErrors.ErrUserNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	if err := h.Storage.UserStorage.SetUserActiveStatus(ctx, req.UserID, req.IsActive); err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	updated, err := h.Storage.UserStorage.GetUserByID(ctx, req.UserID)
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	resp := map[string]any{
		"user": updated,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PRID   string `json:"pull_request_id"`
		PRName string `json:"pull_request_name"`
		Author string `json:"author_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}

	if req.PRID == "" || req.PRName == "" || req.Author == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "missing required fields")
		return
	}

	ctx := r.Context()

	_, err := h.Storage.PullRequestStorage.GetPullRequestByID(ctx, req.PRID)
	if err == nil {
		writeError(w, http.StatusConflict, "PR_EXISTS", "PR id already exists")
		return
	}
	if err != nil && !errors.Is(err, storageErrors.ErrPRNotFound) {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	author, err := h.Storage.UserStorage.GetUserByID(ctx, req.Author)
	if errors.Is(err, storageErrors.ErrUserNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "author not found")
		return
	}
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	teammates, err := h.Storage.UserStorage.GetActiveUsersByTeam(ctx, author.TeamName)
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	reviewers := make([]models.User, 0)
	for _, u := range teammates {
		if u.UserID != author.UserID {
			reviewers = append(reviewers, u)
		}
	}

	assigned := []string{}
	if len(reviewers) > 0 {
		assigned = append(assigned, reviewers[0].UserID)
	}
	if len(reviewers) > 1 {
		assigned = append(assigned, reviewers[1].UserID)
	}

	if err := h.Storage.PullRequestStorage.CreatePullRequest(ctx, req.PRID, req.PRName, req.Author); err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	for _, rID := range assigned {
		if err := h.Storage.PullRequestStorage.AddReviewer(ctx, req.PRID, rID); err != nil {
			writeError(w, 500, "UNKNOWN", err.Error())
			return
		}
	}

	respPR := models.PullRequest{
		PullRequestID:     req.PRID,
		PullRequestName:   req.PRName,
		AuthorID:          req.Author,
		Status:            "OPEN",
		AssignedReviewers: assigned,
		CreatedAt:         nil,
		MergedAt:          nil,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pr": respPR,
	})
}

func (h *Handler) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PRID string `json:"pull_request_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}

	if req.PRID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "missing pull_request_id")
		return
	}

	ctx := r.Context()

	pr, err := h.Storage.PullRequestStorage.GetPullRequestByID(ctx, req.PRID)
	if errors.Is(err, storageErrors.ErrPRNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "pull request not found")
		return
	}
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	reviewers, err := h.Storage.PullRequestStorage.GetReviewersByPR(ctx, req.PRID)
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	if pr.Status == "MERGED" {
		resp := models.PullRequest{
			PullRequestID:     pr.PullRequestID,
			PullRequestName:   pr.PullRequestName,
			AuthorID:          pr.AuthorID,
			Status:            pr.Status,
			AssignedReviewers: reviewers,
			CreatedAt:         pr.CreatedAt,
			MergedAt:          pr.MergedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"pr": resp})
		return
	}

	now := time.Now().UTC()
	if err := h.Storage.PullRequestStorage.SetPullRequestStatus(ctx, req.PRID, "MERGED", now); err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	resp := models.PullRequest{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            "MERGED",
		AssignedReviewers: reviewers,
		CreatedAt:         pr.CreatedAt,
		MergedAt:          &now,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"pr": resp})
}

// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_reviewer_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}

	if req.PullRequestID == "" || req.OldUserID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "missing required fields")
		return
	}

	ctx := r.Context()

	pr, err := h.Storage.PullRequestStorage.GetPullRequestByID(ctx, req.PullRequestID)
	if errors.Is(err, storageErrors.ErrPRNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "pull request not found")
		return
	}
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	if pr.Status == "MERGED" {
		writeError(w, http.StatusConflict, "PR_MERGED", "cannot reassign on merged PR")
		return
	}

	reviewers, err := h.Storage.PullRequestStorage.GetReviewersByPR(ctx, req.PullRequestID)
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	isAssigned := false
	for _, id := range reviewers {
		if id == req.OldUserID {
			isAssigned = true
			break
		}
	}

	if !isAssigned {
		writeError(w, http.StatusConflict, "NOT_ASSIGNED", "reviewer is not assigned to this PR")
		return
	}

	user, err := h.Storage.UserStorage.GetUserByID(ctx, req.OldUserID)
	if errors.Is(err, storageErrors.ErrUserNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "old user not found")
		return
	}
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	teamMembers, err := h.Storage.TeamStorage.GetUsersByTeam(ctx, user.TeamName)
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	assignedSet := map[string]struct{}{}
	for _, rid := range reviewers {
		assignedSet[rid] = struct{}{}
	}

	var replacement *models.User

	for _, m := range teamMembers {
		if !m.IsActive {
			continue
		}
		if m.UserID == pr.AuthorID {
			continue
		}
		if m.UserID == req.OldUserID {
			continue
		}
		if _, exists := assignedSet[m.UserID]; exists {
			continue
		}
		replacement = &m
		break
	}

	if replacement == nil {
		writeError(w, http.StatusConflict, "NO_CANDIDATE", "no active replacement candidate in team")
		return
	}

	if err := h.Storage.PullRequestStorage.RemoveReviewer(ctx, req.PullRequestID, req.OldUserID); err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	if err := h.Storage.PullRequestStorage.AddReviewer(ctx, req.PullRequestID, replacement.UserID); err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	updatedPR, err := h.Storage.PullRequestStorage.GetPullRequestByID(ctx, req.PullRequestID)
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	resp := map[string]any{
		"pr":          updatedPR,
		"replaced_by": replacement.UserID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "missing user_id")
		return
	}

	ctx := r.Context()

	user, err := h.Storage.UserStorage.GetUserByID(ctx, userID)
	if errors.Is(err, storageErrors.ErrUserNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	prs, err := h.Storage.PullRequestStorage.GetPullRequestsByReviewer(ctx, userID)
	if err != nil {
		writeError(w, 500, "UNKNOWN", err.Error())
		return
	}

	response := map[string]any{
		"user_id":       user.UserID,
		"pull_requests": prs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func NewHandler(storage *storage.Storage, log *slog.Logger) *Handler {
	return &Handler{
		Storage: storage,
		Log:     log,
	}
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": msg,
		},
	})
}

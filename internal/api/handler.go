package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gh-pin-repo-go/internal/model"
	"gh-pin-repo-go/internal/service"
)

type Handler struct {
	pinService service.PinService
}

func NewHandler(pinService service.PinService) *Handler {
	return &Handler{
		pinService: pinService,
	}
}

// GetPinnedRepos godoc
// @Summary      Get GitHub pinned repositories
// @Description  Retrieve pinned repositories for a specific GitHub user or organization
// @Tags         pinned
// @Accept       json
// @Produce      json
// @Param        username path string true "GitHub username or organization name" example(torvalds)
// @Param        refresh query boolean false "Force scrape and bypass cache" default(false)
// @Success      200 {object} model.ApiResponse
// @Failure      400 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Failure      429 {object} model.ErrorResponse
// @Failure      500 {object} model.ErrorResponse
// @Router       /api/v1/pinned/{username} [get]
func (h *Handler) GetPinnedRepos(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Success: false,
			Message: "Username is required",
		})
		return
	}

	forceRefresh := c.DefaultQuery("refresh", "false") == "true"

	repos, cached, err := h.pinService.GetPinnedRepos(c.Request.Context(), username, forceRefresh)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, model.ErrorResponse{
				Success: false,
				Message: "GitHub user or organization not found",
				Error:   err.Error(),
			})
			return
		}
		if errors.Is(err, service.ErrRateLimited) {
			c.JSON(http.StatusTooManyRequests, model.ErrorResponse{
				Success: false,
				Message: "Rate limited by GitHub upstream, please try again later",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Success: false,
			Message: "Failed to fetch pinned repositories",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.ApiResponse{
		Success: true,
		Message: "Successfully retrieved pinned repositories",
		Total:   len(repos),
		Cached:  cached,
		Data:    repos,
	})
}

// HealthCheck godoc
// @Summary      Health check
// @Description  Check API service health status
// @Tags         health
// @Produce      json
// @Success      200 {object} map[string]string
// @Router       /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Service is healthy and ready to serve requests",
	})
}

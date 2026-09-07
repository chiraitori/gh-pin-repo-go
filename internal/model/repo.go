package model

// PinnedRepo represents a GitHub pinned repository
type PinnedRepo struct {
	Owner         string `json:"owner" example:"torvalds"`
	Repo          string `json:"repo" example:"AudioNoise"`
	Description   string `json:"description" example:"Random digital audio effects"`
	Language      string `json:"language" example:"C"`
	LanguageColor string `json:"languageColor" example:"#555555"`
	Stars         int    `json:"stars" example:"4500"`
	StarsDisplay  string `json:"starsDisplay" example:"4.5k"`
	Forks         int    `json:"forks" example:"217"`
	ForksDisplay  string `json:"forksDisplay" example:"217"`
	Link          string `json:"link" example:"https://github.com/torvalds/AudioNoise"`
	IsFork        bool   `json:"isFork" example:"false"`
}

// ApiResponse represents the standard API response format
type ApiResponse struct {
	Success bool         `json:"success" example:"true"`
	Message string       `json:"message,omitempty" example:"Successfully fetched pinned repositories"`
	Total   int          `json:"total" example:"6"`
	Cached  bool         `json:"cached" example:"false"`
	Data    []PinnedRepo `json:"data"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Message string `json:"message" example:"User not found or has no pinned repositories"`
	Error   string `json:"error,omitempty" example:"404 Not Found"`
}

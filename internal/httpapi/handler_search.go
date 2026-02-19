package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type searchHandler struct {
	deps Dependencies
}

func newSearchHandler(deps Dependencies) *searchHandler {
	return &searchHandler{deps: deps}
}

type searchRequest struct {
	Query  string  `json:"query" binding:"required"`
	Year   *int    `json:"year"`
	Depth1 *string `json:"depth1"`
	Depth2 *string `json:"depth2"`
	Alias  *string `json:"alias"`
	Limit  *int    `json:"limit"`
}

func (h *searchHandler) search(c *gin.Context) {
	uid := MustUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req searchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	embRes, err := h.deps.Embedding.Embed(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	limit := 20
	if req.Limit != nil && *req.Limit > 0 && *req.Limit <= 100 {
		limit = *req.Limit
	}

	items, err := h.deps.Search.Search(c.Request.Context(), uid, MustGroupIDs(c), embRes.Vector, req.Year, req.Depth1, req.Depth2, req.Alias, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type documentHandler struct {
	deps Dependencies
}

func newDocumentHandler(deps Dependencies) *documentHandler {
	return &documentHandler{deps: deps}
}

type createDocumentRequest struct {
	Year    int      `json:"year" binding:"required"`
	Depth1  string   `json:"depth1" binding:"required"`
	Depth2  string   `json:"depth2" binding:"required"`
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content"`
	Aliases []string `json:"aliases"`
}

func (h *documentHandler) create(c *gin.Context) {
	uid := MustUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req createDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	embInput := strings.TrimSpace(req.Title + "\n\n" + req.Content)
	embRes, err := h.deps.Embedding.Embed(c.Request.Context(), embInput)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.deps.Search.CreateDocumentWithEmbedding(
		c.Request.Context(),
		userID,
		req.Year, req.Depth1, req.Depth2, req.Title, req.Content,
		req.Aliases,
		embRes,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": doc.ID})
}

func (h *documentHandler) get(c *gin.Context) {
	uid := MustUserID(c)
	groupIDs := MustGroupIDs(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	ok, err := h.deps.Auth.CanReadDocument(c.Request.Context(), userID, groupIDs, docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	doc, aliases, err := h.deps.Search.GetDocument(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      doc.ID,
		"year":    doc.Year,
		"depth1":  doc.Depth1,
		"depth2":  doc.Depth2,
		"title":   doc.Title,
		"content": doc.Content,
		"aliases": aliases,
	})
}

type setACLRequest struct {
	GroupID string `json:"group_id" binding:"required"`
	Effect  string `json:"effect" binding:"required"` // deny|read|read_write
}

func (h *documentHandler) setACL(c *gin.Context) {
	uid := MustUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	var req setACLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	groupID, err := uuid.Parse(req.GroupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	ok, err := h.deps.Auth.CanWriteDocument(c.Request.Context(), userID, MustGroupIDs(c), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if err := h.deps.Auth.SetDocumentACL(c.Request.Context(), docID, groupID, req.Effect); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

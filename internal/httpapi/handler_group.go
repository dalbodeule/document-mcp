package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type groupHandler struct {
	deps Dependencies
}

func newGroupHandler(deps Dependencies) *groupHandler {
	return &groupHandler{deps: deps}
}

type createGroupRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *groupHandler) create(c *gin.Context) {
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, err := h.deps.Auth.CreateGroup(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": g.ID, "name": g.Name})
}

type addMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *groupHandler) addMember(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}
	var req addMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if err := h.deps.Auth.AddUserToGroup(c.Request.Context(), userID, groupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *groupHandler) myGroups(c *gin.Context) {
	uid := MustUserID(c)
	userID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	gs, err := h.deps.Auth.ListUserGroups(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(gs))
	for _, g := range gs {
		items = append(items, gin.H{"id": g.ID, "name": g.Name})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

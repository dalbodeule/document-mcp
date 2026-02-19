package httpapi

import (
	"database/sql"
	"net/http"

	"document-mdp/ent"
	"document-mdp/internal/config"
	"document-mdp/internal/embedding"
	"document-mdp/internal/service/auth"
	"document-mdp/internal/service/search"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	Ent *ent.Client
	SQL *sql.DB
	Cfg config.Config

	Embedding embedding.Provider
	Auth      *auth.Service
	Search    *search.Service
}

func NewRouter(deps Dependencies) http.Handler {
	if deps.Cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	a := newAuthHandler(deps)
	d := newDocumentHandler(deps)
	s := newSearchHandler(deps)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", a.register)
		authGroup.POST("/login", a.login)
		authGroup.POST("/refresh", a.refresh)
		authGroup.POST("/logout", a.logout)
	}

	protected := r.Group("/")
	protected.Use(AccessTokenMiddleware(deps))
	{
		g := newGroupHandler(deps)
		protected.POST("/groups", g.create)
		protected.POST("/groups/:id/members", g.addMember)
		protected.GET("/me/groups", g.myGroups)

		protected.POST("/documents", d.create)
		protected.GET("/documents/:id", d.get)
		protected.POST("/documents/:id/acl", d.setACL)
		protected.POST("/search", s.search)
	}

	return r
}

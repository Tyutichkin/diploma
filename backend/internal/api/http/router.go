package http

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	authstory "planner-backend/internal/domain/auth/story"
	routestory "planner-backend/internal/domain/route/story"
	taskstory "planner-backend/internal/domain/task/story"
)

type Deps struct {
	Auth        *authstory.Story
	Task        *taskstory.Story
	Route       *routestory.Story
	CORSOrigins []string
}

func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     d.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	authH := NewAuthHandlers(d.Auth)
	taskH := NewTaskHandlers(d.Task)
	routeH := NewRouteHandlers(d.Route)

	api := r.Group("/api")

	api.POST("/auth/register", authH.Register)
	api.POST("/auth/login", authH.Login)
	api.POST("/auth/refresh", authH.Refresh)
	api.POST("/auth/logout", authH.Logout)

	secured := api.Group("")
	secured.Use(AuthMiddleware(d.Auth))

	secured.GET("/tasks", taskH.List)
	secured.POST("/tasks", taskH.Create)
	secured.PATCH("/tasks/order", taskH.Reorder)
	secured.PATCH("/tasks/:id", taskH.Update)
	secured.DELETE("/tasks/:id", taskH.Delete)

	secured.POST("/routes", routeH.Create)
	secured.POST("/routes/optimize", routeH.Optimize)
	secured.GET("/routes", routeH.List)
	secured.DELETE("/routes", routeH.DeleteAll)
	secured.GET("/routes/:id", routeH.Get)
	secured.DELETE("/routes/:id", routeH.Delete)
	secured.PATCH("/routes/:id/name", routeH.Rename)

	return r
}

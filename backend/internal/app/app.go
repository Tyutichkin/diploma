package app

import (
	"context"
	"net/http"
	"time"

	apihttp "planner-backend/internal/api/http"
	"planner-backend/internal/config"
	authstory "planner-backend/internal/domain/auth/story"
	routeopt "planner-backend/internal/domain/route/optimizer"
	routestory "planner-backend/internal/domain/route/story"
	taskstory "planner-backend/internal/domain/task/story"
	importstory "planner-backend/internal/domain/taskimport/story"
	"planner-backend/internal/platform/distance"
	"planner-backend/internal/platform/postgres/db"
	"planner-backend/internal/platform/postgres/repository"
)

type App struct {
	cfg config.Config
	srv *http.Server
}

func New(cfg config.Config) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DBDSN)
	if err != nil {
		return nil, err
	}

	userRepo := repository.NewUserRepo(pool)
	refreshRepo := repository.NewRefreshRepo(pool)
	taskRepo := repository.NewTaskRepo(pool)
	routeRepo := repository.NewRouteRepo(pool)

	distProvider := distance.NewOSRMProvider(cfg.OSRMBaseURL)
	optimizer := routeopt.NewNearestNeighborTW()

	authStory := authstory.New(userRepo, refreshRepo, cfg.JWTSecret, cfg.AccessTTLMin, cfg.RefreshTTLDays)
	taskStory := taskstory.New(taskRepo)
	routeStory := routestory.New(routeRepo, taskRepo, distProvider, optimizer)
	importStory := importstory.New(taskStory)

	router := apihttp.NewRouter(apihttp.Deps{
		Auth: authStory, Task: taskStory, Route: routeStory, Import: importStory,
		CORSOrigins: cfg.CORSOrigins,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{cfg: cfg, srv: srv}, nil
}

func (a *App) Run() error {
	return a.srv.ListenAndServe()
}

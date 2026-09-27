package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"react-go-cms-content-service/internal/application/service/category"
	"react-go-cms-content-service/internal/application/service/comment"
	"react-go-cms-content-service/internal/application/service/contenttype"
	"react-go-cms-content-service/internal/application/service/dashboard"
	"react-go-cms-content-service/internal/application/service/post"
	"react-go-cms-content-service/internal/application/service/settings"
	"react-go-cms-content-service/internal/application/service/tag"
	"react-go-cms-content-service/internal/application/service/uistring"
	"react-go-cms-content-service/internal/handler"
	"react-go-cms-content-service/internal/infrastructure/cache"
	"react-go-cms-content-service/internal/infrastructure/client"
	"react-go-cms-content-service/internal/infrastructure/configuration"
	"react-go-cms-content-service/internal/infrastructure/httpx"
	"react-go-cms-content-service/internal/infrastructure/repository/repo"
)

const (
	defaultPort         = "8082"
	databasePingTimeout = 10 * time.Second
	readHeaderTimeout   = 10 * time.Second
)

func main() {
	var cfg configuration.Config
	cfg = configuration.LoadConfig(defaultPort)
	var db *sql.DB
	var err error
	db, err = configuration.OpenDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), databasePingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("database: %v", err)
	}
	var popular *cache.Popular
	popular = cache.NewPopular()
	var postRepository *repo.PostRepository
	postRepository = repo.NewPostRepository(db)
	var categoryRepository *repo.CategoryRepository
	categoryRepository = repo.NewCategoryRepository(db, popular)
	var tagRepository *repo.TagRepository
	tagRepository = repo.NewTagRepository(db, popular)
	var commentRepository *repo.CommentRepository
	commentRepository = repo.NewCommentRepository(db)
	var contentTypeRepository *repo.ContentTypeRepository
	contentTypeRepository = repo.NewContentTypeRepository(db)
	var dashboardRepository *repo.DashboardRepository
	dashboardRepository = repo.NewDashboardRepository(db)
	var settingsRepository *repo.SettingsRepository
	settingsRepository = repo.NewSettingsRepository(db)
	var uiStringRepository *repo.UIStringRepository
	uiStringRepository = repo.NewUIStringRepository(db)
	var userClient *client.UserClient
	userClient = client.NewUserClient(cfg.AuthServiceURL)
	var categoryService *category.Service
	categoryService = category.New(categoryRepository)
	var tagService *tag.Service
	tagService = tag.New(tagRepository)
	var postService *post.Service
	postService = post.New(postRepository, categoryService, tagService)
	var commentService *comment.Service
	commentService = comment.New(commentRepository)
	var contentTypeService *contenttype.Service
	contentTypeService = contenttype.New(contentTypeRepository)
	var dashboardService *dashboard.Service
	dashboardService = dashboard.New(dashboardRepository, userClient)
	var settingsService *settings.Service
	settingsService = settings.New(settingsRepository)
	var uiStringService *uistring.Service
	uiStringService = uistring.New(uiStringRepository)
	var httpHandler *handler.Handler
	httpHandler = handler.New(postService, categoryService, tagService, commentService, contentTypeService, dashboardService, settingsService, uiStringService, cfg.JWTSecret, cfg.JWTIssuer)
	var mux *chi.Mux
	mux = chi.NewRouter()
	httpHandler.Register(mux)
	addr := ":" + cfg.Port
	log.Printf("react-go-cms-content-service listening on %s", addr)
	server := &http.Server{
		Addr:              addr,
		Handler:           httpx.CORS(cfg.CORSOrigins, mux),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	log.Fatal(server.ListenAndServe())
}

package api

import (
	"net/http"

	log "gophish/logger"
	
	mid "gophish/middleware"
	"gophish/middleware/ratelimit"
	"gophish/models"
	"gophish/worker"
	"github.com/gorilla/mux"
)

// ServerOption is an option to apply to the API server.
type ServerOption func(*Server)

// Server represents the routes and functionality of the Gophish API.
type Server struct {
	handler http.Handler
	worker  worker.Worker
	limiter *ratelimit.PostLimiter
}

// NewServer returns a new instance of the API handler with the provided options applied.
func NewServer(options ...ServerOption) *Server {
	defaultWorker, err := worker.New()
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}
	defaultLimiter := ratelimit.NewPostLimiter()
	as := &Server{
		worker:  defaultWorker,
		limiter: defaultLimiter,
	}
	for _, opt := range options {
		opt(as)
	}
	as.registerRoutes()
	return as
}

// WithWorker is an option that sets the background worker.
func WithWorker(w worker.Worker) ServerOption {
	return func(as *Server) {
		as.worker = w
	}
}

func WithLimiter(limiter *ratelimit.PostLimiter) ServerOption {
	return func(as *Server) {
		as.limiter = limiter
	}
}

func (as *Server) registerRoutes() {
	root := mux.NewRouter()
	root = root.StrictSlash(true)
	router := root.PathPrefix("/api/").Subrouter()
	router.Use(mid.RequireAPIKey)
	router.Use(mid.EnforceViewOnly)
	router.HandleFunc("/imap/", as.IMAPServer)
	router.HandleFunc("/imap/tenant/{tenant_id:[0-9]+}", as.ImapsByTenant)
	router.HandleFunc("/imap/validate", as.IMAPServerValidate)
	router.HandleFunc("/reset", as.Reset)
	router.HandleFunc("/campaigns/", as.Campaigns)
	router.HandleFunc("/campaigns/summary", as.CampaignsSummary)
	router.HandleFunc("/campaigns/{id:[0-9]+}", as.Campaign)
	router.HandleFunc("/campaigns/{id:[0-9]+}/results", as.CampaignResults)
	router.HandleFunc("/campaigns/{id:[0-9]+}/summary", as.CampaignSummary)
	router.HandleFunc("/campaigns/{id:[0-9]+}/complete", as.CampaignComplete)
	router.HandleFunc("/campaigns/tenant/{tenant_id:[0-9]+}", as.CampaignsByTenant)
	router.HandleFunc("/groups/", as.Groups)
	router.HandleFunc("/groups/summary", as.GroupsSummary)
	router.HandleFunc("/groups/{id:[0-9]+}", as.Group)
	router.HandleFunc("/groups/{id:[0-9]+}/summary", as.GroupSummary)
	router.HandleFunc("/groups/tenant/{tenant_id:[0-9]+}", as.GroupsByTenant)
	router.HandleFunc("/templates/", as.Templates)
	router.HandleFunc("/templates/{id:[0-9]+}", as.Template)
	router.HandleFunc("/templates/tenant/{tenant_id:[0-9]+}", as.TemplatesByTenant)
	router.HandleFunc("/pages/", as.Pages)
	router.HandleFunc("/pages/{id:[0-9]+}", as.Page)
	router.HandleFunc("/pages/tenant/{tenant_id:[0-9]+}", as.PagesByTenant)
	router.HandleFunc("/smtp/", as.SendingProfiles)
	router.HandleFunc("/smtp/{id:[0-9]+}", as.SendingProfile)
	router.HandleFunc("/smtp/tenant/{tenant_id:[0-9]+}", as.SmtpsByTenant)
	router.HandleFunc("/users/", mid.Use(as.Users, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/users/{id:[0-9]+}", mid.Use(as.User))
	router.HandleFunc("/users/tenant/{tenant_id:[0-9]+}", as.UsersByTenant)
	router.HandleFunc("/util/send_test_email", as.SendTestEmail)
	router.HandleFunc("/import/group", as.ImportGroup)
	router.HandleFunc("/import/email", as.ImportEmail)
	router.HandleFunc("/import/site", as.ImportSite)
	router.HandleFunc("/webhooks/", mid.Use(as.Webhooks, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/webhooks/{id:[0-9]+}/validate", mid.Use(as.ValidateWebhook, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/webhooks/{id:[0-9]+}", mid.Use(as.Webhook, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/webhooks/tenant/{tenant_id:[0-9]+}", mid.Use(as.WebhooksByTenant, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/tenants", as.Tenants).Methods("GET", "POST")
	router.HandleFunc("/tenants/{id:[0-9]+}", as.Tenant).Methods("GET", "PUT", "DELETE")
	as.handler = router
}

func (as *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	as.handler.ServeHTTP(w, r)
}

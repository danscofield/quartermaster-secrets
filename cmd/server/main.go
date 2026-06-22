package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	appconfig "github.com/qm-secrets/qm-secrets/internal/config"
	"github.com/qm-secrets/qm-secrets/internal/auth"
	"github.com/qm-secrets/qm-secrets/internal/docs"
	"github.com/qm-secrets/qm-secrets/internal/handler"
	"github.com/qm-secrets/qm-secrets/internal/service"
	"github.com/qm-secrets/qm-secrets/internal/store"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config file (default: QM_SECRETS_CONFIG, config.yaml, qm-secrets.yaml)")
	flag.Parse()

	cfg, err := appconfig.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.AWSRegion))
	if err != nil {
		log.Fatalf("aws config: %v", err)
	}

	dynamoClient := dynamodb.NewFromConfig(awsCfg)
	asmClient := secretsmanager.NewFromConfig(awsCfg)

	metaStore := store.NewDynamoDBStore(dynamoClient, cfg.DynamoDBTable)
	valueStore := store.NewSecretsManagerStore(asmClient, cfg.ASMSecretPrefix)
	secretSvc := service.NewSecretService(metaStore, valueStore)
	secretHandler := handler.NewSecretHandler(secretSvc)

	jwtValidator, err := auth.NewOIDCValidator(ctx, cfg.OIDCIssuer, cfg.OIDCAudience, auth.OIDCOptions{
		InsecureSkipTLSVerify: cfg.OIDCInsecureSkipTLSVerify,
	})
	if err != nil {
		log.Fatalf("oidc: %v", err)
	}
	if cfg.OIDCInsecureSkipTLSVerify {
		log.Println("WARNING: OIDC TLS verification disabled (OIDC_INSECURE_SKIP_TLS_VERIFY); do not use in production")
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	docs.Mount(r)

	r.Route("/secrets", func(r chi.Router) {
		r.Use(jwtValidator.Middleware)
		r.Mount("/", secretHandler.Routes())
	})

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if cfg.TLSEnabled() {
			log.Printf("listening on %s (TLS)", cfg.Addr)
			err = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			log.Printf("listening on %s", cfg.Addr)
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}

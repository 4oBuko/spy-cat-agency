package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	spycatagency "github.com/4oBuko/spy-cat-agency/internal"
	"github.com/4oBuko/spy-cat-agency/internal/repositories"
	"github.com/4oBuko/spy-cat-agency/internal/services"
	"github.com/4oBuko/spy-cat-agency/pkg/catapi"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "user:password@/spycatagency"
	catAPIUrl := "https://api.thecatapi.com/v1/breeds"
	db := initDBConnection(dsn)
	catRepo := repositories.NewMySQLCatRepository(db)
	catAPI := catapi.NewCatAPIClient(catAPIUrl, 1, time.Second)
	catService := services.NewDefaultCatService(catRepo, catAPI)
	missionRepo := repositories.NewMySQLMissionRepository(db)
	targetRepo := repositories.NewMySQLTargetRepository(db)
	missionService := services.NewDefaultMissionService(missionRepo, targetRepo, catRepo)
	server := spycatagency.NewServer(catService, catAPI, missionService)

	go func() {
		if err := server.Run(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func initDBConnection(dsn string) *sql.DB {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)
	return db
}

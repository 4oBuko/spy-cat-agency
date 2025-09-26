package main

import (
	"database/sql"
	"log"
	"time"

	spycatagency "github.com/4oBuko/spy-cat-agency/internal"
	"github.com/4oBuko/spy-cat-agency/internal/repositories"
	"github.com/4oBuko/spy-cat-agency/internal/services"
	"github.com/4oBuko/spy-cat-agency/pkg/catapi"
)

func main() {
	dsn := "user:password@/spycatagency"
	catAPIUrl := ""
	db := initDBConnection(dsn)
	catRepo := repositories.NewMySQLCatRepository(db)
	catAPI := catapi.NewCatAPIClient(catAPIUrl, 1, time.Second)
	catService := services.NewDefaultCatService(catRepo, catAPI)
	missionRepo := repositories.NewMySQLMissionRepository(db)
	targetRepo := repositories.NewMySQLTargetRepository(db)
	missionService := services.NewDefaultMissionService(missionRepo, targetRepo)
	server := spycatagency.NewServer(catService, catAPI, missionService)

	err := server.Run()
	if err != nil {
		log.Fatal(err)
	}
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

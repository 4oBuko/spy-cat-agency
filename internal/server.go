package spycatagency

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/4oBuko/spy-cat-agency/internal/models"
	"github.com/4oBuko/spy-cat-agency/internal/services"
	"github.com/4oBuko/spy-cat-agency/pkg/catapi"
	"github.com/gin-gonic/gin"
)

var Endpoints = struct {
	CatCreate     string
	CatGet        string
	CatGetAll     string
	CatUpdate     string
	CatDelete     string
	CatAssignTask string

	MissionCreate string
	MissionGet    string
	MissionGetAll string
	MissionUpdate string
	MissionDelete string

	TargetComplete string
	TargetUpdate   string
	TargetDelete   string
}{
	CatCreate: "/cats",
	CatGet:    "/cats/:id",
	CatUpdate: "/cats/:id",
	CatDelete: "/cats/:id",
	CatGetAll: "/cats",

	MissionCreate: "/missions",
	MissionGet:    "/missions/:id",
}

type Server struct {
	router         *gin.Engine
	catService     services.CatService
	catAPI         catapi.CatAPI
	missionService services.MissionService
}

func NewServer(catService services.CatService, catAPI catapi.CatAPI, missionService services.MissionService) *Server {
	router := gin.Default()

	server := &Server{
		router:         router,
		catService:     catService,
		catAPI:         catAPI,
		missionService: missionService,
	}
	router.POST(Endpoints.CatCreate, server.handleAddCat)
	router.GET(Endpoints.CatGet, server.handleGetCat)
	//! todo this endpoint must have pagination
	router.GET(Endpoints.CatGetAll, server.handleGetAllCats)
	router.PUT(Endpoints.CatUpdate, server.handleUpdateCat)
	router.DELETE(Endpoints.CatDelete, server.handleDeleteCat)

	router.POST(Endpoints.MissionCreate, server.handleAddMission)
	router.GET(Endpoints.MissionGet, server.handleGetMission)
	return server
}

func (s *Server) handleAddCat(ctx *gin.Context) {
	var cat models.Cat
	if err := ctx.BindJSON(&cat); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	newCat, err := s.catService.Add(ctx, cat)
	if err != nil {
		if myErr, ok := err.(*catapi.UnexistedBreedError); ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": myErr,
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusCreated, newCat)
}

func (s *Server) handleGetCat(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "cat not found. Use number as id!",
		})
		return
	}

	cat, err := s.catService.GetById(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, nil)
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to get cat by id: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, cat)
}

func (s *Server) handleUpdateCat(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "cat not found. Use number as id!",
		})
		return
	}
	var update models.CatUpdate
	if err := ctx.BindJSON(&update); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}
	updatedCat, err := s.catService.Update(ctx, int64(id), update)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "no rows affected during the update",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, updatedCat)
}
func (s *Server) handleDeleteCat(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "cat not found. Use number as id!",
		})
		return
	}

	err = s.catService.DeleteById(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"message": "attempt to delete unexisted entity!",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

func (s *Server) handleGetAllCats(ctx *gin.Context) {
	cats, err := s.catService.GetAll(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "error while attempting to fetch all cats:" + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, cats)
}

func (s *Server) handleAddMission(ctx *gin.Context) {
	var mission models.Mission
	if err := ctx.BindJSON(&mission); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid data. New mission should have at least one target!",
		})
		return
	}
	savedMission, err := s.missionService.Add(ctx, mission)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "attempt to add new mission failed: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusCreated, savedMission)
}

func (s *Server) handleGetMission(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "mission not found. Use number as id!",
		})
		return
	}

	mission, err := s.missionService.GetById(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, nil)
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to get cat by id: " + err.Error(),
		})
		return

	}
	ctx.JSON(http.StatusOK, mission)

}

// todo: add tests for assigning tasks to cat
// todo: test how cascade delete works (if it works at all)

func (s *Server) Run() error {
	return s.router.Run()
}

func (s *Server) Handler() http.Handler {
	return s.router
}

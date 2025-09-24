package spycatagency

import (
	"database/sql"
	"errors"
	"fmt"
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
}

type Server struct {
	router     *gin.Engine
	catService services.CatService
	catAPI     catapi.CatAPI
}

func NewServer(catService services.CatService, catAPI catapi.CatAPI) *Server {
	router := gin.Default()

	server := &Server{
		router:     router,
		catService: catService,
		catAPI:     catAPI,
	}
	router.POST(Endpoints.CatCreate, server.handleAddCat)
	router.GET(Endpoints.CatGet, server.handleGetCat)
	// ! this endpoint must have pagination
	router.GET(Endpoints.CatGetAll, server.handleGetAllCats)
	router.PUT(Endpoints.CatUpdate, server.handleUpdateCat)
	router.DELETE(Endpoints.CatDelete, server.handleDeleteCat)

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

	newCat, err := s.catService.AddNewCat(ctx, cat)
	if err != nil {
		if myErr, ok := err.(*catapi.UnexistedBreedError); ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": myErr,
			})
			return
		}
		fmt.Println("error", err)
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

	cat, err := s.catService.GetCatById(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, nil)
			return
		}
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
	updatedCat, err := s.catService.UpdateCat(ctx, int64(id), update)
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
	cats, err := s.catService.GetAllCats(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "error while attempting to fetch all cats:" + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, cats)
}

// todo: add tests for assigning tasks to cat
// todo: test how cascade delete works (if it works at all)

func (s *Server) Run() error {
	return s.router.Run()
}

func (s *Server) Handler() http.Handler {
	return s.router
}

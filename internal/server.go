package spycatagency

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/4oBuko/spy-cat-agency/internal/models"
	"github.com/4oBuko/spy-cat-agency/internal/myerrors"
	"github.com/4oBuko/spy-cat-agency/internal/services"
	"github.com/4oBuko/spy-cat-agency/pkg/catapi"
	"github.com/gin-gonic/gin"
)

var Endpoints = struct {
	CatCreate string
	CatGet    string
	CatGetAll string
	CatUpdate string
	CatDelete string

	MissionCreate   string
	MissionGet      string
	MissionGetAll   string
	MissionUpdate   string
	MissionDelete   string
	MissionAssign   string
	MissionComplete string

	TargetComplete string
	TargetUpdate   string
	TargetDelete   string
	TargetAdd      string
}{
	CatCreate: "/cats",
	CatGet:    "/cats/:id",
	CatUpdate: "/cats/:id",
	CatDelete: "/cats/:id",
	CatGetAll: "/cats",

	MissionCreate:   "/missions",
	MissionGet:      "/missions/:id",
	MissionGetAll:   "/missions",
	MissionUpdate:   "/missions/:id",
	MissionAssign:   "/missions/:id/assign/:catId",
	MissionComplete: "/missions/:id/complete",
	MissionDelete:   "/missions/:id",

	TargetComplete: "/missions/:id/targets/:targetId/complete",
	TargetUpdate:   "/missions/:id/targets/:targetId",
	TargetDelete:   "/missions/:id/targets/:targetId",
	TargetAdd:      "/missions/:id/targets",
}

type Server struct {
	router         *gin.Engine
	catService     services.CatService
	catAPI         catapi.CatAPI
	missionService services.MissionService
}

func NewServer(catService services.CatService, catAPI catapi.CatAPI, missionService services.MissionService) *Server {
	router := gin.Default()

	router.Use(SimpleLoggingMiddleware())
	router.Use(ErrorHandler())

	server := &Server{
		router:         router,
		catService:     catService,
		catAPI:         catAPI,
		missionService: missionService,
	}
	router.POST(Endpoints.CatCreate, server.handleAddCat)
	router.GET(Endpoints.CatGet, server.handleGetCat)
	router.GET(Endpoints.CatGetAll, server.handleGetAllCats)
	router.PUT(Endpoints.CatUpdate, server.handleUpdateCat)
	router.DELETE(Endpoints.CatDelete, server.handleDeleteCat)

	router.POST(Endpoints.MissionCreate, server.handleAddMission)
	router.GET(Endpoints.MissionGet, server.handleGetMission)
	router.GET(Endpoints.MissionGetAll, server.handleGetAllMissions)
	router.POST(Endpoints.MissionAssign, server.handleAssignMission)
	router.POST(Endpoints.MissionComplete, server.handleCompleteMission)
	router.DELETE(Endpoints.MissionDelete, server.handleDeleteMission)

	router.POST(Endpoints.TargetComplete, server.handleCompleteTarget)
	router.POST(Endpoints.TargetUpdate, server.handleUpdateTarget)
	router.DELETE(Endpoints.TargetDelete, server.handleDeleteTarget)
	router.POST(Endpoints.TargetAdd, server.handleAddTarget)

	return server
}

func (s *Server) handleAddCat(ctx *gin.Context) {
	var cat models.Cat
	if err := ctx.BindJSON(&cat); err != nil {
		ctx.Error(myerrors.NewBadRequestError(err.Error()))
		return
	}

	newCat, err := s.catService.Add(ctx, cat)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusCreated, newCat)
}

func (s *Server) handleGetCat(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Error(myerrors.NewNotFoundError("use number as id"))
		return
	}

	cat, err := s.catService.GetById(ctx, int64(id))
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, cat)
}

func (s *Server) handleUpdateCat(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Error(myerrors.NewNotFoundError("use number as id"))
		return
	}
	var update models.CatUpdate
	if err := ctx.BindJSON(&update); err != nil {
		ctx.Error(myerrors.NewBadRequestError(err.Error()))
		return
	}
	updatedCat, err := s.catService.Update(ctx, int64(id), update)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, updatedCat)
}
func (s *Server) handleDeleteCat(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Error(myerrors.NewNotFoundError("use number as id"))
		return
	}

	err = s.catService.DeleteById(ctx, int64(id))
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

func (s *Server) handleGetAllCats(ctx *gin.Context) {
	cats, err := s.catService.GetAll(ctx)
	if err != nil {
		ctx.Error(err)
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

func (s *Server) handleGetAllMissions(ctx *gin.Context) {
	missions, err := s.missionService.GetAll(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "attempt to get all missions failed: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, missions)
}

func (s *Server) handleAssignMission(ctx *gin.Context) {
	missionId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "mission not found. Use number as id!",
		})
		return
	}
	catId, err := strconv.Atoi(ctx.Param("catId"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "cat not found. Use number as id!",
		})
		return
	}
	err = s.missionService.Assign(ctx, int64(missionId), int64(catId))
	if err != nil {
		if err, ok := err.(*myerrors.AppError); ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "attempt to assign mission failed: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

func (s *Server) handleCompleteTarget(ctx *gin.Context) {
	missionId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "mission not found. Use number as id!",
		})
		return
	}
	targetId, err := strconv.Atoi(ctx.Param("targetId"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "target not found. Use number as id!",
		})
		return
	}

	err = s.missionService.CompleteTarget(ctx, int64(missionId), int64(targetId))
	if err != nil {
		if err, ok := err.(*myerrors.AppError); ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "attempt to complete target failed: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

func (s *Server) handleUpdateTarget(ctx *gin.Context) {
	missionId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "mission not found. Use number as id!",
		})
		return
	}
	targetId, err := strconv.Atoi(ctx.Param("targetId"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "target not found. Use number as id!",
		})
		return
	}
	var update models.TargetUpdate
	if err := ctx.BindJSON(&update); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}
	target, err := s.missionService.UpdateTarget(ctx, int64(missionId), int64(targetId), update)
	if err != nil {
		if err, ok := err.(*myerrors.AppError); ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "attempt to complete target failed: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, target)
}

func (s *Server) handleDeleteTarget(ctx *gin.Context) {
	missionId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "mission not found. Use number as id!",
		})
		return
	}
	targetId, err := strconv.Atoi(ctx.Param("targetId"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "target not found. Use number as id!",
		})
		return
	}
	err = s.missionService.DeleteTarget(ctx, int64(missionId), int64(targetId))
	if err != nil {
		if err, ok := err.(*myerrors.AppError); ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"message": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "attempt to complete target failed: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

func (s *Server) handleAddTarget(ctx *gin.Context) {
	missionId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "mission not found. Use number as id!",
		})
		return
	}
	var target models.Target
	if err := ctx.BindJSON(&target); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "incorrect target format: " + err.Error(),
		})
		return
	}

	updatedMission, err := s.missionService.AddTarget(ctx, int64(missionId), target)
	if err != nil {
		if err, ok := err.(*myerrors.AppError); ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "attempt to complete target failed: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, updatedMission)
}

func (s *Server) handleCompleteMission(ctx *gin.Context) {
	missionId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "mission not found. Use number as id!",
		})
		return
	}
	mission, err := s.missionService.Complete(ctx, int64(missionId))
	if err != nil {
		if err, ok := err.(*myerrors.AppError); ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "attempt to complete target failed: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, mission)
}

func (s *Server) handleDeleteMission(ctx *gin.Context) {
	missionId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": "mission not found. Use number as id!",
		})
		return
	}
	err = s.missionService.Delete(ctx, int64(missionId))
	if err != nil {
		if err, ok := err.(*myerrors.AppError); ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "attempt to complete target failed: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

func (s *Server) Run() error {
	return s.router.Run()
}

func (s *Server) Handler() http.Handler {
	return s.router
}

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			switch e := err.(type) {
			case *myerrors.AppError:
				c.JSON(e.StatusCode, gin.H{"error": e.Message})
			default:
				// Log unexpected errors
				log.Printf("Unexpected error: %v", err)
				c.JSON(500, gin.H{"error": "Internal server error"})
			}
		}
	}
}

func SimpleLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log Request
		logRequest(c)

		// Capture response body
		responseBody := &bytes.Buffer{}
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           responseBody,
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Log Response
		logResponse(c.Writer.Status(), responseBody.String())
	}
}

func logRequest(c *gin.Context) {
	method := c.Request.Method
	path := c.Request.URL.Path

	// Read request body if present
	var requestBody string
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err == nil && len(bodyBytes) > 0 {
			requestBody = string(bodyBytes)
			// Restore the request body for further processing
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	if requestBody != "" {
		prettyBody := prettifyJSON(requestBody)
		log.Printf("REQUEST: %s %s\nBody:\n%s", method, path, prettyBody)
	} else {
		log.Printf("REQUEST: %s %s", method, path)
	}
}

func logResponse(statusCode int, responseBody string) {
	if responseBody != "" {
		prettyBody := prettifyJSON(responseBody)
		log.Printf("RESPONSE: %d\nBody:\n%s", statusCode, prettyBody)
	} else {
		log.Printf("RESPONSE: %d", statusCode)
	}
}

// prettifyJSON formats JSON string with indentation if valid, otherwise returns original
func prettifyJSON(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}

	var jsonObj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonObj); err != nil {
		// If it's not valid JSON, return as is
		return jsonStr
	}

	prettyJSON, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		return jsonStr
	}

	return string(prettyJSON)
}

package spycatagency_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	spycatagency "github.com/4oBuko/spy-cat-agency/internal"
	"github.com/4oBuko/spy-cat-agency/internal/models"
	"github.com/4oBuko/spy-cat-agency/internal/repositories"
	"github.com/4oBuko/spy-cat-agency/internal/services"
	"github.com/4oBuko/spy-cat-agency/pkg/catapi"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

var server *spycatagency.Server

func TestMain(m *testing.M) {
	ctx := context.Background()
	pwd, _ := os.Getwd()
	initSQLPath := filepath.Join(pwd, "db", "init.sql")
	mysqlContainer, err := mysql.Run(ctx,
		"mysql:9.4.0",
		mysql.WithDatabase("spycatagency"),
		mysql.WithUsername("root"),
		mysql.WithPassword("password"),
		mysql.WithScripts(initSQLPath),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(mysqlContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()
	if err != nil {
		log.Printf("failed to start container: %s", err)
		return
	}

	connectionString, err := mysqlContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatal("failed to get connection string:%w", err)
	}

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)
	defer db.Close()

	catRepo := repositories.NewMySQLCatRepository(db)
	catAPI := NewFakeCatAPI()
	catService := services.NewDefaultCatService(catRepo, catAPI)
	missionRepo := repositories.NewMySQLMissionRepository(db)
	targetRepo := repositories.NewMySQLTargetRepository(db)
	missionService := services.NewDefaultMissionService(missionRepo, targetRepo)
	server = spycatagency.NewServer(catService, catAPI, missionService)
	code := m.Run()
	os.Exit(code)
}

// ? this test runs first when table cat is empty
// ? to have independent state from ther tests
func TestGetAllCats(t *testing.T) {
	cat1 := models.Cat{
		Name:              "Silky",
		Breed:             "abob",
		YearsOfExperience: 2,
		Salary:            500,
	}
	cat2 := models.Cat{
		Name:              "Milky",
		Breed:             "asho",
		YearsOfExperience: 4,
		Salary:            1500,
	}
	cat3 := models.Cat{
		Name:              "Morgana",
		Breed:             "acur",
		YearsOfExperience: 10,
		Salary:            5555,
	}
	var cats []models.Cat
	cats = append(cats, createNewCatSuccessfully(t, cat1))
	cats = append(cats, createNewCatSuccessfully(t, cat2))
	cats = append(cats, createNewCatSuccessfully(t, cat3))
	request, _ := http.NewRequest(http.MethodGet, spycatagency.Endpoints.CatGetAll, nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	allCats := unmarshal[[]models.Cat](t, response.Body.Bytes())
	require.Equal(t, 3, len(allCats))
	require.Equal(t, cats[0], allCats[0])
	require.Equal(t, cats[1], allCats[1])
	require.Equal(t, cats[2], allCats[2])

}

func TestAddNewCat(t *testing.T) {

	t.Run("add new cat successfully", func(t *testing.T) {
		newCat := models.Cat{
			Name:              "Tom",
			Breed:             "abys",
			YearsOfExperience: 1,
			Salary:            1000,
		}
		cat := createNewCatSuccessfully(t, newCat)
		newCat.Id = cat.Id
		assert.Equal(t, newCat, cat)
	})

	t.Run("attempt to add cat with unexisted breed", func(t *testing.T) {
		newCat := models.Cat{
			Name:              "Fraud",
			Breed:             "fraud",
			YearsOfExperience: 12348,
			Salary:            393911,
		}
		body := marshal(t, newCat)
		request, _ := http.NewRequest(http.MethodPost, spycatagency.Endpoints.CatCreate, bytes.NewBuffer(body))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

}

func TestGetCatById(t *testing.T) {
	t.Run("create new cat and try to get it", func(t *testing.T) {
		newCat := models.Cat{
			Name:              "Aboba",
			Breed:             "abob",
			YearsOfExperience: 1,
			Salary:            777,
		}
		cat := createNewCatSuccessfully(t, newCat)

		copycat := getCatByIDSuccessfully(t, int(cat.Id))
		assert.Equal(t, cat, copycat)
	})
	t.Run("try to get non existing cat", func(t *testing.T) {
		request := newGetCatByIdRequest(math.MaxInt64)
		doRequestAndExpect(t, request, http.StatusNotFound)
	})
}

func TestUpdateSalary(t *testing.T) {
	t.Run("create new cat and double their salary", func(t *testing.T) {
		newCat := models.Cat{
			Name:              "Bobby",
			Breed:             "asho",
			YearsOfExperience: 3,
			Salary:            900,
		}
		cat := createNewCatSuccessfully(t, newCat)

		url := strings.Replace(spycatagency.Endpoints.CatUpdate, ":id", strconv.Itoa(int(cat.Id)), 1)
		bodyStr := fmt.Sprintf(`{"salary":%d}`, cat.Salary*2)
		request, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(bodyStr))
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)

		require.Equal(t, http.StatusOK, response.Code)
		updatedCat := unmarshal[models.Cat](t, response.Body.Bytes())
		cat.Salary *= 2
		assert.Equal(t, cat, updatedCat)
	})
}

func TestDeleteCat(t *testing.T) {
	t.Run("delete cat and try to get by id", func(t *testing.T) {
		newCat := models.Cat{
			Name:              "Phantom Thief",
			Breed:             "acur",
			YearsOfExperience: 5,
			Salary:            555,
		}
		cat := createNewCatSuccessfully(t, newCat)

		request := newDeleteByIdRequest(int(cat.Id))
		doRequestAndExpect(t, request, http.StatusOK)

		request = newGetCatByIdRequest(int(cat.Id))
		doRequestAndExpect(t, request, http.StatusNotFound)

	})
	t.Run("delete non existing cat", func(t *testing.T) {
		request := newDeleteByIdRequest(math.MaxInt64)
		doRequestAndExpect(t, request, http.StatusNotFound)
	})
}

func TestAddNewMission(t *testing.T) {
	t.Run("add new mission successfully", func(t *testing.T) {
		newCat := models.Cat{
			Name:              "Ash",
			Breed:             "abys",
			YearsOfExperience: 6,
			Salary:            1200,
		}

		savedCat := createNewCatSuccessfully(t, newCat)
		newTarget := models.Target{
			Name:    "cucumber",
			Country: "USA",
			Notes:   "Never let it get behind your back",
		}
		newMisson := models.Mission{
			CatId: savedCat.Id,
			Targets: []models.Target{
				newTarget,
				{
					Name:    "Cristmas tree",
					Country: "Italy",
					Notes:   "Attacking it at night when it's not expecting you",
				},
			},
		}
		createNewMissionSuccessfully(t, savedCat, newMisson)
	})

	t.Run("new mission without tartest field in request", func(t *testing.T) {
		newCat := models.Cat{
			Name:              "Killer Queen",
			Breed:             "aege",
			YearsOfExperience: 4,
			Salary:            1972,
		}
		cat := createNewCatSuccessfully(t, newCat)

		newMission := models.Mission{
			Targets: nil,
		}

		createNewMissionSuccessfully(t, cat, newMission)
	})

	t.Run("new mission with empty targets array", func(t *testing.T) {
		newCat := models.Cat{
			Name:              "Rina",
			Breed:             "abob",
			YearsOfExperience: 3,
			Salary:            3301,
		}

		cat := createNewCatSuccessfully(t, newCat)
		newMission := models.Mission{
			Targets: make([]models.Target, 0, 1),
		}

		createNewMissionSuccessfully(t, cat, newMission)
	})
}

func getCatByIDSuccessfully(t *testing.T, id int) models.Cat {
	t.Helper()
	request := newGetCatByIdRequest(id)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	cat := unmarshal[models.Cat](t, response.Body.Bytes())
	return cat
}

func newDeleteByIdRequest(id int) *http.Request {
	url := strings.Replace(spycatagency.Endpoints.CatDelete, ":id", strconv.Itoa(id), 1)
	request, _ := http.NewRequest(http.MethodDelete, url, nil)
	return request
}

func newGetCatByIdRequest(id int) *http.Request {
	url := strings.Replace(spycatagency.Endpoints.CatGet, ":id", strconv.Itoa(id), 1)
	request, _ := http.NewRequest(http.MethodGet, url, nil)
	return request
}
func createNewMissionSuccessfully(t *testing.T, cat models.Cat, newMission models.Mission) models.Mission {
	t.Helper()
	newMission.CatId = cat.Id
	body := marshal(t, newMission)
	request, _ := http.NewRequest(http.MethodPost, spycatagency.Endpoints.MissionCreate, bytes.NewReader(body))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code)

	mission := unmarshal[models.Mission](t, response.Body.Bytes())
	require.Equal(t, len(newMission.Targets), len(mission.Targets))
	newMission.Id = mission.Id
	for i := range mission.Targets {
		newMission.Targets[i].Id = mission.Targets[i].Id
		// targets doesn't have mission id in response
		mission.Targets[i].MissionId = mission.Id
		newMission.Targets[i].MissionId = mission.Id
	}

	assert.Equal(t, newMission, mission)
	return mission
}

func createNewCatSuccessfully(t *testing.T, cat models.Cat) models.Cat {
	t.Helper()
	body := marshal(t, cat)
	request, _ := http.NewRequest(http.MethodPost, spycatagency.Endpoints.CatCreate, bytes.NewReader(body))
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code)

	persistedCat := unmarshal[models.Cat](t, response.Body.Bytes())
	cat.Id = persistedCat.Id
	require.Equal(t, cat, persistedCat)
	return persistedCat
}

func unmarshal[T any](t *testing.T, body []byte) T {
	t.Helper()
	var result T
	err := json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal(err)
	}
	return result

}
func marshal[T any](t *testing.T, value T) []byte {
	t.Helper()
	result, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func doRequestAndExpect(t *testing.T, request *http.Request, expected int) {
	t.Helper()
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	assert.Equal(t, expected, response.Code)
}

type FakeCatAPI struct {
	breeds []catapi.Breed
}

func NewFakeCatAPI() *FakeCatAPI {
	return &FakeCatAPI{
		[]catapi.Breed{
			{
				Id:   "abys",
				Name: "Abyssinian",
			},
			{
				Id:   "aege",
				Name: "Aegean",
			},
			{
				Id:   "abob",
				Name: "American Bobtail",
			},
			{
				Id:   "acur",
				Name: "American Curl",
			},
			{
				Id:   "asho",
				Name: "American Shorthair",
			},
		},
	}
}

func (n *FakeCatAPI) GetBreedById(ctx context.Context, id string) (catapi.Breed, error) {
	for _, breed := range n.breeds {
		if breed.Id == id {
			return breed, nil
		}
	}
	return catapi.Breed{}, &catapi.UnexistedBreedError{BreedId: id}
}

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

	catRepo := repositories.NewMySQLCatRepo(db)
	catAPI := NewFakeCatAPI()
	catService := services.NewDefaultCatService(catRepo, catAPI)
	server = spycatagency.NewServer(catService, catAPI)
	code := m.Run()
	os.Exit(code)
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
		body, err := json.Marshal(newCat)
		if err != nil {
			t.Fatal(err)
		}
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
		updatedCat := unmarshalCat(t, response.Body.Bytes())
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

// ? this only checks if the endpoint works
// ? for normal testing this test need to insert into empty table
// ? then add certain number of new entities assert results
func TestGetAllCats(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, spycatagency.Endpoints.CatGetAll, nil)
	doRequestAndExpect(t, request, http.StatusOK)
}

func getCatByIDSuccessfully(t *testing.T, id int) models.Cat {
	request := newGetCatByIdRequest(id)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	cat := unmarshalCat(t, response.Body.Bytes())
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

func createNewCatSuccessfully(t *testing.T, cat models.Cat) models.Cat {
	t.Helper()
	body, err := json.Marshal(cat)
	if err != nil {
		t.Fatal(err)
	}
	request, _ := http.NewRequest(http.MethodPost, spycatagency.Endpoints.CatCreate, bytes.NewReader(body))
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code)

	persistedCat := unmarshalCat(t, response.Body.Bytes())
	return persistedCat
}

func unmarshalCat(t *testing.T, body []byte) models.Cat {
	t.Helper()
	var persitedCat models.Cat
	err := json.Unmarshal(body, &persitedCat)
	if err != nil {
		t.Fatal(err)
	}
	return persitedCat

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

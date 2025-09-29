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
var cleaner *dbCleaner

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

	cleaner = &dbCleaner{db: db}
	catRepo := repositories.NewMySQLCatRepository(db)
	catAPI := NewFakeCatAPI()
	catService := services.NewDefaultCatService(catRepo, catAPI)
	missionRepo := repositories.NewMySQLMissionRepository(db)
	targetRepo := repositories.NewMySQLTargetRepository(db)
	missionService := services.NewDefaultMissionService(missionRepo, targetRepo, catRepo)
	server = spycatagency.NewServer(catService, catAPI, missionService)
	code := m.Run()
	os.Exit(code)
}

type dbCleaner struct {
	db *sql.DB
}

func (d *dbCleaner) cleanDB() error {
	deleteCats := "DELETE FROM cats"
	deleteTargets := "DELETE FROM targets"
	deleteMissions := "DELETE FROM missions"
	_, err := d.db.Exec(deleteCats)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(deleteTargets)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(deleteMissions)
	if err != nil {
		return err
	}
	return nil
}

func TestAddNewCat(t *testing.T) {

	t.Run("add new cat successfully", func(t *testing.T) {
		newCat := models.Cat{
			Name:              "Tom",
			Breed:             "abys",
			YearsOfExperience: 1,
			Salary:            1000,
		}
		cat := addNewCatSuccessfully(t, newCat)
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
		cat := addNewCatSuccessfully(t, newCat)

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
		cat := addNewCatSuccessfully(t, newCat)

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
		cat := addNewCatSuccessfully(t, newCat)

		request := newDeleteCatRequest(int(cat.Id))
		doRequestAndExpect(t, request, http.StatusOK)

		request = newGetCatByIdRequest(int(cat.Id))
		doRequestAndExpect(t, request, http.StatusNotFound)

	})
	t.Run("delete non existing cat", func(t *testing.T) {
		request := newDeleteCatRequest(math.MaxInt64)
		doRequestAndExpect(t, request, http.StatusNotFound)
	})
	t.Run("attempt to delete cat with assigned uncompleted mission", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Namus",
			Breed:             "abys",
			YearsOfExperience: 5,
			Salary:            2500,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Kebab",
					Country: "Turkey",
				},
			},
		}
		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		assignMissionSuccessfully(t, mission, cat)

		request := newDeleteCatRequest(int(cat.Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})
	t.Run("delete cat with assigned completed mission", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Namus",
			Breed:             "abys",
			YearsOfExperience: 5,
			Salary:            2500,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Kebab",
					Country: "Turkey",
				},
			},
		}
		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)
		mission = completeTargetSuccessfully(t, mission.Id, mission.Targets[0].Id)
		mission = completeMissionSuccessfully(t, mission)

		request := newDeleteCatRequest(int(cat.Id))
		doRequestAndExpect(t, request, http.StatusOK)

		request = newGetCatByIdRequest(int(cat.Id))
		doRequestAndExpect(t, request, http.StatusNotFound)

		request = newGetMissionByIdRequest(int(mission.Id))
		doRequestAndExpect(t, request, http.StatusNotFound)
	})
}
func TestGetAllCats(t *testing.T) {
	cleaner.cleanDB()
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
	cats = append(cats, addNewCatSuccessfully(t, cat1))
	cats = append(cats, addNewCatSuccessfully(t, cat2))
	cats = append(cats, addNewCatSuccessfully(t, cat3))
	request, _ := http.NewRequest(http.MethodGet, spycatagency.Endpoints.CatGetAll, nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	allCats := unmarshal[[]models.Cat](t, response.Body.Bytes())
	require.Equal(t, 3, len(allCats))
	assert.Equal(t, cats, allCats)
}
func TestAddNewMission(t *testing.T) {
	t.Run("add new mission successfully", func(t *testing.T) {
		newMisson := models.Mission{
			Targets: []models.Target{
				{
					Name:    "cucumber",
					Country: "USA",
					Notes:   "Never let it get behind your back",
				},
				{
					Name:    "Cristmas tree",
					Country: "Italy",
					Notes:   "Attacking it at night when it's not expecting you",
				},
			},
		}
		addNewMissionSuccessfully(t, newMisson)
	})

	t.Run("attempt to create mission without targets", func(t *testing.T) {

		newMission := models.Mission{}
		request := newAddMissionRequest(t, newMission)
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

	t.Run("attempt to create a mission with more than 3 targets", func(t *testing.T) {
		newMission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Lucinia Kushinada",
					Country: "Poland",
					Notes:   "Incredibly skilled netrunner",
				},
				{
					Name:    "Sasha Yakovleva",
					Country: "Russia",
					Notes:   "Very good net runner. Also her claws are bigger than mine",
				},
				{
					Name:    "Rebecca",
					Country: "USA",
					Notes:   "She is crazy good at shooting",
				},
				{
					Name:    "David Martinez",
					Country: "USA",
					Notes:   "Arasaka akademy student",
				},
			},
		}

		request := newAddMissionRequest(t, newMission)
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

}

func TestGetMissionById(t *testing.T) {
	t.Run("create new mission and get it by id", func(t *testing.T) {
		newMission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Cat Nip",
					Country: "Poland",
					Notes:   "It's mighty but has low stamina",
				},
			},
		}
		mission := addNewMissionSuccessfully(t, newMission)
		mById := getMissionByIdSuccessfully(t, int(mission.Id))
		require.Equal(t, len(mission.Targets), len(mById.Targets))

		setMissionIdForTargets(mission)
		setMissionIdForTargets(mById)

		assert.Equal(t, mission, mById)
	})
	t.Run("attempt to get unexisted mission", func(t *testing.T) {
		request := newGetMissionByIdRequest(math.MaxInt64)
		doRequestAndExpect(t, request, http.StatusNotFound)
	})
}

func TestGetAllMissions(t *testing.T) {
	t.Run("get all missions", func(t *testing.T) {
		cleaner.cleanDB()

		missions := []models.Mission{
			{Targets: []models.Target{{
				Name:    "Suguru Kamoshida",
				Country: "Japan",
				Notes:   "Abusive volleyball coach. His heart must be changed"}},
			},
			{Targets: []models.Target{{
				Name:    "Junya Kaneshiro",
				Country: "Japan",
				Notes:   "Shibuya scammer"}},
			},
			{Targets: []models.Target{{
				Name:    "Kunikazu Okumura",
				Country: "Japan",
				Notes:   "CEO of Okumura Foods, who runs Big Bang Burger"}},
			},
		}
		missions[0] = addNewMissionSuccessfully(t, missions[0])
		missions[1] = addNewMissionSuccessfully(t, missions[1])
		missions[2] = addNewMissionSuccessfully(t, missions[2])

		request, _ := http.NewRequest(http.MethodGet, spycatagency.Endpoints.MissionGetAll, nil)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code)

		allMissions := unmarshal[[]models.Mission](t, response.Body.Bytes())
		for i := range allMissions {
			setMissionIdForTargets(allMissions[i])
		}
		assert.Equal(t, missions, allMissions)
	})
}

func TestAssignMission(t *testing.T) {
	t.Run("create mission and assign it to a cat", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Morgana",
			Breed:             "abys",
			YearsOfExperience: 5,
			Salary:            5000,
		}
		cat = addNewCatSuccessfully(t, cat)
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Futaba Sakura",
					Country: "Japan",
				},
			},
		}
		mission = addNewMissionSuccessfully(t, mission)
		assignMissionSuccessfully(t, mission, cat)

	})
	t.Run("attempt to assign busy cat to a mission", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Felix",
			Breed:             "abys",
			YearsOfExperience: 3,
			Salary:            1300,
		}
		cat = addNewCatSuccessfully(t, cat)
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Thomas Shelby",
					Country: "UK",
					Notes:   "Mafia",
				},
			},
		}
		mission = addNewMissionSuccessfully(t, mission)

		assignMissionSuccessfully(t, mission, cat)

		newMission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Johny Jostar",
					Country: "USA",
					Notes:   "Likes to ride on horses",
				},
			},
		}
		newMission = addNewMissionSuccessfully(t, newMission)
		request := newAssingMissionRequest(int(newMission.Id), int(cat.Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)

	})
	t.Run("attempt to assign cat to a completed mission", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Tom",
			Breed:             "abob",
			YearsOfExperience: 5,
			Salary:            3200,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Jerry",
					Country: "USA",
				},
				{
					Name:    "Nibbles",
					Country: "USA",
				},
			},
		}
		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)
		mission = completeTargetSuccessfully(t, mission.Id, mission.Targets[0].Id)
		mission = completeTargetSuccessfully(t, mission.Id, mission.Targets[1].Id)
		mission = completeMissionSuccessfully(t, mission)

		newCat := models.Cat{
			Name:              "Chattini",
			Breed:             "abys",
			YearsOfExperience: 1,
			Salary:            1000,
		}
		newCat = addNewCatSuccessfully(t, newCat)
		request := newAssingMissionRequest(int(mission.Id), int(newCat.Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})
	t.Run("attempt to assign non existing mission", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Felix",
			Breed:             "abys",
			YearsOfExperience: 3,
			Salary:            1300,
		}

		cat = addNewCatSuccessfully(t, cat)
		request := newAssingMissionRequest(math.MaxInt64, int(cat.Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})
}

func TestCompleteMission(t *testing.T) {
	t.Run("complete mission with all targets completed", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Yoruichi Shihoin",
			Breed:             "abys",
			YearsOfExperience: 25,
			Salary:            20000,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Orihime Inoue",
					Country: "Japan",
				},
				{
					Name:    "Ichigo Kurosaki",
					Country: "USA",
				},
			},
		}

		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)
		mission = completeTargetSuccessfully(t, mission.Id, mission.Targets[0].Id)
		mission = completeTargetSuccessfully(t, mission.Id, mission.Targets[1].Id)

		completeMissionSuccessfully(t, mission)
	})
	t.Run("attempt to complete mission with uncompleted targets", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Tom",
			Breed:             "abob",
			YearsOfExperience: 5,
			Salary:            3200,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Jerry",
					Country: "USA",
				},
				{
					Name:    "Nibbles",
					Country: "USA",
				},
			},
		}

		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)
		completeTargetSuccessfully(t, mission.Id, mission.Targets[0].Id)

		request := newCompleteMissionRequest(int(mission.Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})
	t.Run("attempt to complete mission withihout assigned cat", func(t *testing.T) {
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Killer b",
					Country: "France",
				},
				{
					Name:    "Killer a",
					Country: "Poland",
				},
			},
		}
		mission = addNewMissionSuccessfully(t, mission)

		request := newCompleteMissionRequest(int(mission.Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

	t.Run("attempt to complete already completed mission", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Sphinx",
			Breed:             "abys",
			YearsOfExperience: 5,
			Salary:            4321,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Tired dev",
					Country: "Ukraine",
					Notes:   "He is tired writing tests",
				},
			},
		}

		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)
		mission = completeTargetSuccessfully(t, mission.Id, mission.Targets[0].Id)

		completeMissionSuccessfully(t, mission)

		request := newCompleteMissionRequest(int(mission.Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

}

func TestUpdateMissionTargets(t *testing.T) {
	t.Run("set target as completed", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Stormy",
			Breed:             "abys",
			YearsOfExperience: 3,
			Salary:            740,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Iggy",
					Country: "USA",
					Notes:   "Lowes coffee bubble gums",
				},
			},
		}

		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)

		uMission := completeTargetSuccessfully(t, mission.Id, mission.Targets[0].Id)
		target := uMission.Targets[0]
		mission.Targets[0].Completed = target.Completed
		assert.Equal(t, mission.Targets[0], target)
	})

	t.Run("attempt to complete already completed target", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Glossy",
			Breed:             "abys",
			YearsOfExperience: 1,
			Salary:            700,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Johny Silverhand",
					Country: "USA",
					Notes:   "Chipping in",
				},
			},
		}

		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)

		uMission := completeTargetSuccessfully(t, mission.Id, mission.Targets[0].Id)
		target := uMission.Targets[0]
		request := newCompleteTargetRequest(int(mission.Id), int(target.Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

	t.Run("attempt to complete target of unassigned mission", func(t *testing.T) {
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Freddie Mercury",
					Country: "England",
					Notes:   "He will rock you",
				},
			},
		}
		mission = addNewMissionSuccessfully(t, mission)

		request := newCompleteTargetRequest(int(mission.Id), int(mission.Targets[0].Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

	t.Run("update target's notes", func(t *testing.T) {
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Yagami Light",
					Country: "Japan",
					Notes:   "Hella smart, but sus",
				},
			},
		}
		mission = addNewMissionSuccessfully(t, mission)
		update := models.TargetUpdate{
			Notes: "Hella smart, but sus. He is super sus, L should be assigned to it",
		}
		reuqest := newUpdateTargetRequest(t, int(mission.Id), int(mission.Targets[0].Id), update)
		response := httptest.NewRecorder()

		server.Handler().ServeHTTP(response, reuqest)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("attempt to update notes of a completed target", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Saimon",
			Breed:             "abys",
			YearsOfExperience: 10,
			Salary:            9999,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Seong Gi-hun",
					Country: "South Korea",
					Notes:   "Gamble a lot of money on horse races",
				},
			},
		}
		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)
		uMission := completeTargetSuccessfully(t, mission.Id, mission.Targets[0].Id)
		target := uMission.Targets[0]

		update := models.TargetUpdate{
			Notes: "He has played this games before",
		}
		request := newUpdateTargetRequest(t, int(mission.Id), int(target.Id), update)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("attempt to update target unrelated to the mission", func(t *testing.T) {
		mission1 := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Rober Oppenheimer",
					Country: "USA",
					Notes:   "Spend time with communists often",
				},
			},
		}
		mission2 := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Joseph Joestar",
					Country: "USA",
					Notes:   "Very tall and fast guy",
				},
			},
		}
		mission1 = addNewMissionSuccessfully(t, mission1)
		mission2 = addNewMissionSuccessfully(t, mission2)

		update := models.TargetUpdate{
			Notes: `Very tall and fast guy. Often says "Your next line is"`,
		}

		request := newUpdateTargetRequest(t, int(mission1.Id), int(mission2.Targets[0].Id), update)
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

	t.Run("delete target from mission", func(t *testing.T) {
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Dio Brando",
					Country: "Egypt",
				},
				{
					Name:    "Noriaki Kakyoin",
					Country: "Japan",
				},
			},
		}

		mission = addNewMissionSuccessfully(t, mission)
		request := newDeleteTargetRequest(int(mission.Id), int(mission.Targets[1].Id))
		doRequestAndExpect(t, request, http.StatusOK)

		mission.Targets = mission.Targets[:1]
		updatedMission := getMissionByIdSuccessfully(t, int(mission.Id))
		setMissionIdForTargets(updatedMission)

		assert.Equal(t, mission, updatedMission)

	})

	t.Run("attempt to delete completed target", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Giorno Giovanna",
			Breed:             "abys",
			YearsOfExperience: 1,
			Salary:            1000,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Diavolo",
					Country: "Italy",
				},
				{
					Name:    "Polpo",
					Country: "Italy",
				},
			},
		}

		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)
		uMission := completeTargetSuccessfully(t, mission.Id, mission.Targets[1].Id)

		request := newDeleteTargetRequest(int(mission.Id), int(uMission.Targets[1].Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

	t.Run("attempt to delete target from a mission with only one target", func(t *testing.T) {
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Enrico Pucchi",
					Country: "USA",
				},
			},
		}

		mission = addNewMissionSuccessfully(t, mission)
		request := newDeleteTargetRequest(int(mission.Id), int(mission.Targets[0].Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

	t.Run("add target to a mission", func(t *testing.T) {
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Jane Doe",
					Country: "USA",
				},
			},
		}

		mission = addNewMissionSuccessfully(t, mission)
		newTarget := models.Target{
			Name:    "John Doe",
			Country: "USA",
		}
		request := newAddTargetRequest(t, int(mission.Id), newTarget)
		response := httptest.NewRecorder()

		server.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code)

		uMission := unmarshal[models.Mission](t, response.Body.Bytes())
		mission.Targets = append(mission.Targets, newTarget)
		require.Equal(t, len(mission.Targets), len(uMission.Targets))

		mission.Targets[1].Id = uMission.Targets[1].Id
		setMissionIdForTargets(mission)
		setMissionIdForTargets(uMission)
		assert.Equal(t, mission, uMission)
	})

	t.Run("attempt to add 4th target to a mission", func(t *testing.T) {
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Makoto Nijima",
					Country: "Japan",
				},
				{
					Name:    "Haru Okumura",
					Country: "Japan",
				},
				{
					Name:    "Ann Takamaki",
					Country: "Japan",
				},
			},
		}
		mission = addNewMissionSuccessfully(t, mission)

		newTarget := models.Target{
			Name:    "Sae Nijima",
			Country: "Japan",
		}
		request := newAddTargetRequest(t, int(mission.Id), newTarget)
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})

	t.Run("attempt to add target to a completed mission", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Giorno Giovanna",
			Breed:             "abys",
			YearsOfExperience: 1,
			Salary:            1000,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Jane Doe",
					Country: "USA",
				},
			},
		}

		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)
		mission = completeTargetSuccessfully(t, mission.Id, mission.Targets[0].Id)
		mission = completeMissionSuccessfully(t, mission)

		nTarget := models.Target{
			Name:    "Chika",
			Country: "USA",
		}
		request := newAddTargetRequest(t, int(mission.Id), nTarget)
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})
}

func TestMissionDelete(t *testing.T) {
	t.Run("delete mission", func(t *testing.T) {
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Jane Doe",
					Country: "Brazil",
				},
			},
		}
		mission = addNewMissionSuccessfully(t, mission)

		request := newDeleteMissionRequest(int(mission.Id))
		doRequestAndExpect(t, request, http.StatusOK)

		request = newGetMissionByIdRequest(int(mission.Id))
		doRequestAndExpect(t, request, http.StatusNotFound)
	})

	t.Run("attempt to delete assigned mission", func(t *testing.T) {
		cat := models.Cat{
			Name:              "Felixius",
			Breed:             "abys",
			YearsOfExperience: 7,
			Salary:            5500,
		}
		mission := models.Mission{
			Targets: []models.Target{
				{
					Name:    "Jane Doe",
					Country: "Portugal",
				},
			},
		}
		cat = addNewCatSuccessfully(t, cat)
		mission = addNewMissionSuccessfully(t, mission)
		mission = assignMissionSuccessfully(t, mission, cat)

		request := newDeleteMissionRequest(int(mission.Id))
		doRequestAndExpect(t, request, http.StatusBadRequest)
	})
}

func getMissionByIdSuccessfully(t *testing.T, id int) models.Mission {
	t.Helper()
	request := newGetMissionByIdRequest(id)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	mission := unmarshal[models.Mission](t, response.Body.Bytes())
	return mission
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

func addNewMissionSuccessfully(t *testing.T, newMission models.Mission) models.Mission {
	t.Helper()
	request := newAddMissionRequest(t, newMission)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code)

	mission := unmarshal[models.Mission](t, response.Body.Bytes())
	require.Equal(t, len(newMission.Targets), len(mission.Targets))
	newMission.Id = mission.Id
	for i := range mission.Targets {
		newMission.Targets[i].Id = mission.Targets[i].Id
	}
	setMissionIdForTargets(mission)
	setMissionIdForTargets(newMission)

	require.Equal(t, newMission, mission)
	return mission
}

func completeTargetSuccessfully(t *testing.T, missionId, targetid int64) models.Mission {
	t.Helper()
	request := newCompleteTargetRequest(int(missionId), int(targetid))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	mission := getMissionByIdSuccessfully(t, int(missionId))
	return mission
}

func addNewCatSuccessfully(t *testing.T, cat models.Cat) models.Cat {
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

func assignMissionSuccessfully(t *testing.T, mission models.Mission, cat models.Cat) models.Mission {
	t.Helper()
	request := newAssingMissionRequest(int(mission.Id), int(cat.Id))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	mission = getMissionByIdSuccessfully(t, int(mission.Id))
	require.Equal(t, mission.CatId, cat.Id)
	return mission
}

func completeMissionSuccessfully(t *testing.T, mission models.Mission) models.Mission {
	t.Helper()
	request := newCompleteMissionRequest(int(mission.Id))
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	cMission := unmarshal[models.Mission](t, response.Body.Bytes())
	require.Equal(t, true, cMission.Completed)
	mission.Completed = cMission.Completed
	assert.Equal(t, mission, cMission)
	return cMission
}

func newDeleteCatRequest(id int) *http.Request {
	url := strings.Replace(spycatagency.Endpoints.CatDelete, ":id", strconv.Itoa(id), 1)
	request, _ := http.NewRequest(http.MethodDelete, url, nil)
	return request
}

func newGetCatByIdRequest(id int) *http.Request {
	url := strings.Replace(spycatagency.Endpoints.CatGet, ":id", strconv.Itoa(id), 1)
	request, _ := http.NewRequest(http.MethodGet, url, nil)
	return request
}
func newGetMissionByIdRequest(id int) *http.Request {
	url := strings.Replace(spycatagency.Endpoints.MissionGet, ":id", strconv.Itoa(id), 1)
	request, _ := http.NewRequest(http.MethodGet, url, nil)
	return request
}

func newAssingMissionRequest(missionId, catId int) *http.Request {
	url := strings.Replace(spycatagency.Endpoints.MissionAssign, ":id", strconv.Itoa(missionId), 1)
	url = strings.Replace(url, ":catId", strconv.Itoa(catId), 1)
	request, _ := http.NewRequest(http.MethodPost, url, nil)
	return request
}

func newAddMissionRequest(t *testing.T, mission models.Mission) *http.Request {
	t.Helper()
	body := marshal(t, mission)
	request, _ := http.NewRequest(http.MethodPost, spycatagency.Endpoints.MissionCreate, bytes.NewReader(body))
	return request
}

func newCompleteTargetRequest(missionId, targetId int) *http.Request {
	url := strings.Replace(spycatagency.Endpoints.TargetComplete, ":id", strconv.Itoa(missionId), 1)
	url = strings.Replace(url, ":targetId", strconv.Itoa(targetId), 1)
	request, _ := http.NewRequest(http.MethodPost, url, nil)
	return request
}

func newUpdateTargetRequest(t *testing.T, missionId, targetId int, update models.TargetUpdate) *http.Request {
	t.Helper()
	url := strings.Replace(spycatagency.Endpoints.TargetUpdate, ":id", strconv.Itoa(missionId), 1)
	url = strings.Replace(url, ":targetId", strconv.Itoa(targetId), 1)
	body := marshal(t, update)
	request, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	return request
}

func newDeleteTargetRequest(missionId, targetId int) *http.Request {
	url := strings.Replace(spycatagency.Endpoints.TargetDelete, ":id", strconv.Itoa(missionId), 1)
	url = strings.Replace(url, ":targetId", strconv.Itoa(targetId), 1)
	request, _ := http.NewRequest(http.MethodDelete, url, nil)
	return request
}

func newAddTargetRequest(t *testing.T, missionId int, newTarget models.Target) *http.Request {
	t.Helper()
	url := strings.Replace(spycatagency.Endpoints.TargetAdd, ":id", strconv.Itoa(missionId), 1)
	body := marshal(t, newTarget)
	request, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	return request
}

func newCompleteMissionRequest(missionId int) *http.Request {
	url := strings.Replace(spycatagency.Endpoints.MissionComplete, ":id", strconv.Itoa(missionId), 1)
	request, _ := http.NewRequest(http.MethodPost, url, nil)
	return request
}

func newDeleteMissionRequest(missionId int) *http.Request {
	url := strings.Replace(spycatagency.Endpoints.MissionDelete, ":id", strconv.Itoa(missionId), 1)
	request, _ := http.NewRequest(http.MethodDelete, url, nil)
	return request
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

func setMissionIdForTargets(mission models.Mission) {
	for i := range mission.Targets {
		mission.Targets[i].MissionId = mission.Id
	}
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

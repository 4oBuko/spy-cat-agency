package spycatagency

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/4oBuko/spy-cat-agency/internal/models"
	"github.com/4oBuko/spy-cat-agency/pkg/catapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGracefulShutdown(t *testing.T) {
	newCat := models.Cat{
		Name:              "Fraud",
		Breed:             "fraud",
		YearsOfExperience: 12348,
		Salary:            393911,
	}
	body, err := json.Marshal(&newCat)
	if err != nil {
		t.Fatalf("failed to marshal cat:%v", err)
	}

	t.Run("test complete active requests", func(t *testing.T) {
		catService := &MockCatService{
			onRequestStart: make(chan bool, 2),
		}
		server := NewServer(catService, &MockCatAPI{}, &MockMissionService{})
		go func() {
			err := server.Run()

			if err != nil {
				fmt.Printf("failed to init server:%v", err)
			}
		}()
		for range 2 {
			go func() {
				http.Post("http://localhost:8080/cats", "application/json", bytes.NewReader(body))
			}()
		}
		<-catService.onRequestStart
		<-catService.onRequestStart

		err = server.Shutdown(context.Background())
		if err != nil {
			t.Fatalf("failed to shutdown:%v", err)
		}
		assert.Equal(t, 2, catService.addCounter)

	})

	t.Run("test reject new requests", func(t *testing.T) {
		catService := &MockCatService{
			onRequestStart: make(chan bool, 1),
		}
		catService.On("Add", body).Return(models.Cat{}, nil)
		server := NewServer(catService, &MockCatAPI{}, &MockMissionService{})
		go func() {
			err := server.Run()
			if err != nil {
				fmt.Printf("failed to init server:%v", err)
			}
		}()
		go func() {
			http.Post("http://localhost:8080/cats", "application/json", bytes.NewReader(body))
		}()
		<-catService.onRequestStart
		go func() {
			time.Sleep(120 * time.Millisecond)
			_, err = http.Post("http://localhost:8080/cats", "application/json", bytes.NewReader(body))
		}()
		server.Shutdown(context.Background())
		time.Sleep(100 * time.Millisecond)

		assert.NotNil(t, err)
	})
}

type MockCatService struct {
	mock.Mock
	addCounter     int
	onRequestStart chan bool
	mu             sync.Mutex
}

type MockMissionService struct {
}

type MockCatAPI struct {
}

func (m *MockCatAPI) GetBreedById(ctx context.Context, id string) (catapi.Breed, error) {
	return catapi.Breed{}, nil
}

func (m *MockCatService) Add(ctx context.Context, cat models.Cat) (models.Cat, error) {
	m.onRequestStart <- true
	time.Sleep(5 * time.Second)
	m.mu.Lock()
	m.addCounter++
	m.mu.Unlock()
	return models.Cat{}, nil
}

func (m *MockCatService) GetAddCounter() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addCounter
}

func (m *MockCatService) GetById(ctx context.Context, id int64) (models.Cat, error) {
	return models.Cat{}, nil
}

func (m *MockCatService) Update(ctx context.Context, id int64, update models.CatUpdate) (models.Cat, error) {
	return models.Cat{}, nil
}

func (m *MockCatService) DeleteById(ctx context.Context, id int64) error {
	return nil
}

func (m *MockCatService) GetAll(ctx context.Context, query models.PaginationQuery) (models.PaginatedCats, error) {
	return models.PaginatedCats{}, nil
}

func (m *MockMissionService) Add(ctx context.Context, mission models.Mission) (models.Mission, error) {
	return models.Mission{}, nil
}

func (m *MockMissionService) GetById(ctx context.Context, id int64) (models.Mission, error) {
	return models.Mission{}, nil
}

func (m *MockMissionService) GetAll(ctx context.Context, query models.PaginationQuery) (models.PaginatedMissions, error) {
	return models.PaginatedMissions{}, nil
}

func (m *MockMissionService) Assign(ctx context.Context, missionId, catId int64) error {
	return nil
}

func (m *MockMissionService) CompleteTarget(ctx context.Context, missionId, targetId int64) error {
	return nil
}

func (m *MockMissionService) UpdateTarget(ctx context.Context, missionId, targetId int64, update models.TargetUpdate) (models.Target, error) {
	return models.Target{}, nil
}

func (m *MockMissionService) DeleteTarget(ctx context.Context, missionId, targetId int64) error {
	return nil
}

func (m *MockMissionService) AddTarget(ctx context.Context, missionId int64, target models.Target) (models.Mission, error) {
	return models.Mission{}, nil
}

func (m *MockMissionService) Complete(ctx context.Context, missionId int64) (models.Mission, error) {
	return models.Mission{}, nil
}

func (m *MockMissionService) Delete(ctx context.Context, missionId int64) error {
	return nil
}

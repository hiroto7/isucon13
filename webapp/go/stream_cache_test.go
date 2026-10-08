package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestStreamCachePublicationAndInitializationFence(t *testing.T) {
	clearStreamMetadata()
	generation := currentStreamGeneration()
	input := streamMetadata{model: LivestreamModel{ID: 1, UserID: 2, Title: "committed"}, tags: []Tag{{ID: 3, Name: "tag"}}}
	publishStreamMetadata(map[int64]streamMetadata{1: input}, generation)
	input.tags[0].Name = "changed"
	got, ok := cachedStreamMetadata(1, generation)
	if !ok || got.tags[0].Name != "tag" {
		t.Fatal("caller mutated published tags")
	}
	got.tags[0].Name = "changed again"
	again, _ := cachedStreamMetadata(1, generation)
	if again.tags[0].Name != "tag" {
		t.Fatal("reader mutated cache")
	}
	clearStreamMetadata()
	publishStreamMetadata(map[int64]streamMetadata{1: input}, generation)
	if _, ok := cachedStreamMetadata(1, currentStreamGeneration()); ok {
		t.Fatal("old request repopulated ID after initialize")
	}
	if _, ok := cachedStreamMetadata(1, generation); ok {
		t.Fatal("old request read new generation")
	}
}
func TestStreamCacheOnlyPublishesAfterSuccessfulHandler(t *testing.T) {
	for _, fail := range []bool{true, false} {
		clearStreamMetadata()
		e := echo.New()
		c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
		err := responseCacheMiddleware(func(c echo.Context) error {
			state := responses(c.Request().Context())
			state.pendingStreams[10] = streamMetadata{model: LivestreamModel{ID: 10, Title: "reservation"}, tags: []Tag{}}
			if _, ok := cachedStreamMetadata(10, state.streamGeneration); ok {
				t.Fatal("published before handler commit/success")
			}
			if fail {
				return errors.New("rollback")
			}
			return c.NoContent(http.StatusCreated)
		})(c)
		_, present := cachedStreamMetadata(10, currentStreamGeneration())
		if present == fail || (err != nil) != fail {
			t.Fatal("failed request published or successful request lost")
		}
	}
}
func TestStreamCacheConcurrentReadersAndInitialization(t *testing.T) {
	clearStreamMetadata()
	var group sync.WaitGroup
	for n := 0; n < 8; n++ {
		group.Add(1)
		go func(n int) {
			defer group.Done()
			for j := 0; j < 200; j++ {
				generation := currentStreamGeneration()
				id := int64(n*200 + j)
				publishStreamMetadata(map[int64]streamMetadata{id: {model: LivestreamModel{ID: id, Title: fmt.Sprint(id)}, tags: []Tag{}}}, generation)
				if got, ok := cachedStreamMetadata(id, generation); ok && got.model.ID != id {
					t.Error("wrong cached ID")
				}
			}
		}(n)
	}
	for n := 0; n < 100; n++ {
		clearStreamMetadata()
	}
	group.Wait()
}

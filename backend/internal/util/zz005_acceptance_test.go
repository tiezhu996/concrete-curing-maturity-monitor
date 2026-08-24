package util

import (
	"errors"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestConcurrentWriteErrorNoRace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const workers = 16
	start := make(chan struct{})
	var done sync.WaitGroup
	done.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer done.Done()
			<-start
			for j := 0; j < 500; j++ {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = httptest.NewRequest("GET", "/", nil)
				WriteError(c, errors.New("boom"))
			}
		}()
	}
	close(start)
	done.Wait()
}

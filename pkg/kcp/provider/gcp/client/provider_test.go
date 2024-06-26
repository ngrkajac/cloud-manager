package client

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	"time"
)

type providerSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *providerSuite) SetupTest() {
	suite.ctx = log.IntoContext(context.Background(), logr.Discard())
}

func (suite *providerSuite) TestGetCachedGcpClient() {
	ctx := context.Background()
	saJsonKeyPath := os.Getenv("GCP_SA_JSON_KEY_PATH")
	err := os.Setenv("GCP_CLIENT_RENEW_DURATION", "500ms")
	assert.Nil(suite.T(), err)
	prevClient := &http.Client{}
	renewed := 0
	for i := 0; i < 33; i++ {
		client, err := GetCachedGcpClient(ctx, saJsonKeyPath)
		assert.Nil(suite.T(), err)
		if prevClient != client {
			renewed++
		}
		time.Sleep(100 * time.Millisecond)
		prevClient = client
	}
	assert.Equal(suite.T(), 7, renewed) //First loot iteration also adds to renewed count. So the result is 1 + totalTime/duration i.e. 1 + 33/5 which is 7
}

func TestProvider(t *testing.T) {
	t.SkipNow() // This test relies on the environment variable GCP_SA_JSON_KEY_PATH and also connection to gcp end point so skipping it for now. If needed can be commented out for manual testing
	suite.Run(t, new(providerSuite))
}

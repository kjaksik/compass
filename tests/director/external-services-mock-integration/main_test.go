package external_services_mock_integration

import (
	"log"
	"os"
	"testing"

	"github.com/pkg/errors"

	"github.com/vrischmann/envconfig"
)

type config struct {
	DefaultTenant               string
	DirectorURL                 string
	ExternalServicesMockBaseURL string
}

var testConfig config

func TestMain(m *testing.M) {
	err := envconfig.Init(&testConfig)
	if err != nil {
		log.Fatal(errors.Wrap(err, "while initializing envconfig"))
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

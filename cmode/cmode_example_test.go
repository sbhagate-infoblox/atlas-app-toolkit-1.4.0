package cmode

import (
	"net/http"

	"github.com/sbhagate-infoblox/atlas-app-toolkit-1.4.0/cmode/logger"
	"github.com/sirupsen/logrus"
)

func ExampleCMode() {
	appLogger := logrus.StandardLogger()

	cmLogger := logger.New(appLogger)
	cm := New(appLogger, cmLogger)

	http.Handle("/", cm.Handler())

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		appLogger.Fatalf("Server fatal error - %s", err)
	}
}

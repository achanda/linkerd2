package version_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/linkerd/linkerd2/controller/api/public"
	healthcheckPb "github.com/linkerd/linkerd2/controller/gen/common/healthcheck"
	pb "github.com/linkerd/linkerd2/controller/gen/public"
	"github.com/linkerd/linkerd2/pkg/version"
)

func TestVersionCheck(t *testing.T) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "{\"version\": \"v0.3.0\"}")
	})
	go http.ListenAndServe("localhost:23456", nil)

	// wait for HTTP server to initialize
	readyCh := make(chan struct{})
	go func() {
		for {
			_, err := http.Head("http://localhost:23456/")
			if err == nil {
				close(readyCh)
				break
			}
		}
	}()

	select {
	case <-readyCh:
	case <-time.After(5 * time.Second):
		t.Fatalf("Failed to initialize HTTP server")
	}

	t.Run("Passes when versions are latest", func(t *testing.T) {
		version.Version = "v0.3.0"
		mockPublicApi := createMockPublicApi("v0.3.0")

		versionStatusChecker := version.NewVersionStatusChecker("http://localhost:23456/", "", mockPublicApi)
		checks := versionStatusChecker.SelfCheck()

		expectedName := version.VersionSubsystemName
		if checks[0].SubsystemName != expectedName {
			t.Fatalf("Expecting check name to be [%s], got [%s]", expectedName, checks[0].SubsystemName)
		}
		if checks[1].SubsystemName != expectedName {
			t.Fatalf("Expecting check name to be [%s], got [%s]", expectedName, checks[0].SubsystemName)
		}

		expectedStatus := healthcheckPb.CheckStatus_OK
		if checks[0].Status != expectedStatus {
			t.Fatalf("Expecting cli check status to be [%d], got [%d]", expectedStatus, checks[0].Status)
		}
		if checks[1].Status != expectedStatus {
			t.Fatalf("Expecting control plane check status to be [%d], got [%d]", expectedStatus, checks[1].Status)
		}

		expectedDescription := version.CliCheckDescription
		if checks[0].CheckDescription != expectedDescription {
			t.Fatalf("Expecting check description to be [%s], got [%s]", expectedDescription, checks[0].CheckDescription)
		}
		expectedDescription = version.ControlPlaneCheckDescription
		if checks[1].CheckDescription != expectedDescription {
			t.Fatalf("Expecting check description to be [%s], got [%s]", expectedDescription, checks[0].CheckDescription)
		}
	})

	t.Run("Fails when cli version is not latest", func(t *testing.T) {
		version.Version = "v0.1.1"
		mockPublicApi := createMockPublicApi("v0.3.0")

		versionStatusChecker := version.NewVersionStatusChecker("http://localhost:23456/", "", mockPublicApi)
		checks := versionStatusChecker.SelfCheck()

		expectedStatus := healthcheckPb.CheckStatus_FAIL
		if checks[0].Status != expectedStatus {
			t.Fatalf("Expecting check status to be [%d], got [%d]", expectedStatus, checks[0].Status)
		}

		expectedMessage := "is running version v0.1.1 but the latest version is v0.3.0"
		if checks[0].FriendlyMessageToUser != expectedMessage {
			t.Fatalf("Expecting message to be [%s], got [%s]", expectedMessage, checks[0].FriendlyMessageToUser)
		}
	})

	t.Run("Fails when control plane version is not latest", func(t *testing.T) {
		version.Version = "v0.3.0"
		mockPublicApi := createMockPublicApi("v0.1.1")

		versionStatusChecker := version.NewVersionStatusChecker("http://localhost:23456/", "", mockPublicApi)
		checks := versionStatusChecker.SelfCheck()

		expectedStatus := healthcheckPb.CheckStatus_FAIL
		if checks[1].Status != expectedStatus {
			t.Fatalf("Expecting check status to be [%d], got [%d]", expectedStatus, checks[1].Status)
		}

		expectedMessage := "is running version v0.1.1 but the latest version is v0.3.0"
		if checks[1].FriendlyMessageToUser != expectedMessage {
			t.Fatalf("Expecting message to be [%s], got [%s]", expectedMessage, checks[1].FriendlyMessageToUser)
		}
	})

	t.Run("Supports overriding the expected version", func(t *testing.T) {
		version.Version = "customversion"
		mockPublicApi := createMockPublicApi("customversion")

		versionStatusChecker := version.NewVersionStatusChecker("http://localhost:23456/", "customversion", mockPublicApi)
		checks := versionStatusChecker.SelfCheck()

		for _, check := range checks {
			if check.Status != healthcheckPb.CheckStatus_OK {
				t.Errorf("Expecting check for [%s] to be [%s], got [%s]",
					check.CheckDescription, healthcheckPb.CheckStatus_OK, check.Status)
			}
		}
	})
}

func createMockPublicApi(version string) *public.MockApiClient {
	return &public.MockApiClient{
		VersionInfoToReturn: &pb.VersionInfo{
			ReleaseVersion: version,
		},
	}
}

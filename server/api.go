package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
)

const requestBodyMaxSizeBytes = 1024 * 1024 // 1MB

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	// All HTTP endpoints of this plugin require a logged-in user.
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	// All HTTP endpoints of this plugin require the user to be a System Admin
	if !p.Client.User.HasPermissionTo(userID, mattermostModel.PermissionManageSystem) {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
	}

	p.Client.Log.Info(r.URL.Path)

	router := mux.NewRouter()

	// Routes called by the plugin's webapp
	router.HandleFunc("/api/v1/legalholds", p.listLegalHolds).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/legalholds", p.createLegalHold).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/legalholds/{legalhold_id:[A-Za-z0-9]+}/release", p.releaseLegalHold).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/legalholds/{legalhold_id:[A-Za-z0-9]+}", p.updateLegalHold).Methods(http.MethodPut)
	router.HandleFunc("/api/v1/legalholds/{legalhold_id:[A-Za-z0-9]+}/download", p.downloadLegalHold).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/legalholds/{legalhold_id:[A-Za-z0-9]+}/run", p.runSingleLegalHold).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/legalholds/{legalhold_id:[A-Za-z0-9]+}/resetstatus", p.resetLegalHoldStatus).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/test_amazon_s3_connection", p.testAmazonS3Connection).Methods(http.MethodPost)

	// Other routes
	router.HandleFunc("/api/v1/legalhold/run", p.runJobFromAPI).Methods(http.MethodPost)

	p.router = router
	p.router.ServeHTTP(w, r)
}

// listLegalHolds serves a list of all LegalHold objects
func (p *Plugin) listLegalHolds(w http.ResponseWriter, _ *http.Request) {
	legalHolds, err := p.KVStore.GetAllLegalHolds()
	if err != nil {
		http.Error(w, "an error occurred fetching the legal holds", http.StatusInternalServerError)
		p.Client.Log.Error(err.Error())
		return
	}

	b, jsonErr := json.Marshal(legalHolds)
	if jsonErr != nil {
		http.Error(w, "Error encoding json", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		p.API.LogError("failed to write http response", err.Error())
		return
	}
}

// createLegalHold creates a new LegalHold
func (p *Plugin) createLegalHold(w http.ResponseWriter, r *http.Request) {
	var createLegalHold model.CreateLegalHold
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, requestBodyMaxSizeBytes)).Decode(&createLegalHold); err != nil {
		http.Error(w, "failed to parse request body", http.StatusBadRequest)
		p.Client.Log.Error(err.Error())
		return
	}

	legalHold := model.NewLegalHoldFromCreate(createLegalHold)

	config := p.API.GetConfig()
	if config == nil {
		http.Error(w, "failed to get config", http.StatusInternalServerError)
		return
	}

	if err := legalHold.IsValidForCreate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	savedLegalHold, err := p.KVStore.CreateLegalHold(legalHold)
	if err != nil {
		http.Error(w, "failed to save new legal hold", http.StatusInternalServerError)
		p.Client.Log.Error(err.Error())
		return
	}

	b, jsonErr := json.Marshal(savedLegalHold)
	if jsonErr != nil {
		http.Error(w, "Error encoding json", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		p.API.LogError("failed to write http response", err.Error())
		return
	}
}

// releaseLegalHold releases a LegalHold and removes all data associated with it
func (p *Plugin) releaseLegalHold(w http.ResponseWriter, r *http.Request) {
	legalholdID, err := RequireLegalHoldID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch the LegalHold
	lh, err := p.KVStore.GetLegalHoldByID(legalholdID)
	if err != nil {
		p.API.LogError("Failed to release legal hold - retrieve legal hold from kvstore", err.Error())
		http.Error(w, "failed to release legal hold", http.StatusInternalServerError)
		return
	}

	// Remove the LegalHold files.
	err = p.FileBackend.RemoveDirectory(lh.BasePath())
	if err != nil {
		p.API.LogError("Failed to release legal hold - failed to delete base directory", err.Error())
		http.Error(w, "failed to release legal hold", http.StatusInternalServerError)
		return
	}

	// Delete the LegalHold from the store.
	err = p.KVStore.DeleteLegalHold(legalholdID)
	if err != nil {
		p.API.LogError("Failed to release legal hold - deleting legal hold from kvstore", err.Error())
		http.Error(w, "failed to release legal hold", http.StatusInternalServerError)
		return
	}

	b, jsonErr := json.Marshal(make(map[string]interface{}))
	if jsonErr != nil {
		http.Error(w, "Error encoding json", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		p.API.LogError("failed to write http response", err.Error())
		return
	}
}

// updateLegalHold updates the properties of a LegalHold
func (p *Plugin) updateLegalHold(w http.ResponseWriter, r *http.Request) {
	var updateLegalHold model.UpdateLegalHold
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, requestBodyMaxSizeBytes)).Decode(&updateLegalHold); err != nil {
		http.Error(w, "failed to parse request body", http.StatusBadRequest)
		p.Client.Log.Error(err.Error())
		return
	}

	// Check that the LegalHold matches the ID in the URL parameter.
	legalholdID, err := RequireLegalHoldID(r)
	if err != nil {
		http.Error(w, "failed to parse LegalHold ID", http.StatusBadRequest)
		p.Client.Log.Error(err.Error())
		return
	}

	if legalholdID != updateLegalHold.ID {
		http.Error(w, "invalid LegalHold ID", http.StatusBadRequest)
		p.Client.Log.Error("legal hold ID specified in request parameters does not match legal hold ID")
		return
	}

	config := p.API.GetConfig()
	if config == nil {
		http.Error(w, "failed to get config", http.StatusInternalServerError)
		return
	}

	if err = updateLegalHold.IsValid(); err != nil {
		http.Error(w, "LegalHold update data is not valid", http.StatusBadRequest)
		p.Client.Log.Error(err.Error())
		return
	}

	// Retrieve the legal hold we are modifying
	originalLegalHold, err := p.KVStore.GetLegalHoldByID(legalholdID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		p.Client.Log.Error(err.Error())
		return
	}

	newLegalHold := originalLegalHold.DeepCopy()
	newLegalHold.ApplyUpdates(updateLegalHold)

	savedLegalHold, err := p.KVStore.UpdateLegalHold(newLegalHold, *originalLegalHold)
	if err != nil {
		http.Error(w, "failed to update legal hold", http.StatusInternalServerError)
		p.Client.Log.Error(err.Error())
		return
	}

	b, jsonErr := json.Marshal(savedLegalHold)
	if jsonErr != nil {
		http.Error(w, "Error encoding json", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		p.API.LogError("failed to write http response", err.Error())
		return
	}
}

func (p *Plugin) downloadLegalHold(w http.ResponseWriter, r *http.Request) {
	// Get the LegalHold.
	legalholdID, err := RequireLegalHoldID(r)
	if err != nil {
		http.Error(w, "failed to parse LegalHold ID", http.StatusBadRequest)
		p.Client.Log.Error(err.Error())
		return
	}

	legalHold, err := p.KVStore.GetLegalHoldByID(legalholdID)
	if err != nil {
		http.Error(w, "failed to download legal hold", http.StatusInternalServerError)
		p.Client.Log.Error(err.Error())
		return
	}

	// Get the list of files to include in the download.
	files, err := p.FileBackend.ListDirectoryRecursively(legalHold.BasePath())
	if err != nil {
		http.Error(w, "failed to download legal hold", http.StatusInternalServerError)
		p.Client.Log.Error(err.Error())
		return
	}

	// Write headers for the zip file.
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", "legalholddata.zip"))
	w.WriteHeader(http.StatusOK)

	// Write the files to the download on-the-fly.
	zipWriter := zip.NewWriter(w)
	for _, entry := range files {
		header := &zip.FileHeader{
			Name:     entry,
			Method:   zip.Deflate, // deflate also works, but at a cost
			Modified: time.Now(),
		}

		entryWriter, err := zipWriter.CreateHeader(header)
		if err != nil {
			http.Error(w, "failed to download legal hold", http.StatusInternalServerError)
			p.Client.Log.Error(err.Error())
			return
		}

		backendReader, err := p.FileBackend.Reader(entry)
		if err != nil {
			http.Error(w, "failed to download legal hold", http.StatusInternalServerError)
			p.Client.Log.Error(err.Error())
			return
		}

		fileReader := bufio.NewReader(backendReader)

		_, err = io.Copy(entryWriter, fileReader)
		if err != nil {
			http.Error(w, "failed to download legal hold", http.StatusInternalServerError)
			p.Client.Log.Error(err.Error())
			return
		}

		if err = zipWriter.Flush(); err != nil {
			http.Error(w, "failed to download legal hold", http.StatusInternalServerError)
			p.Client.Log.Error(err.Error())
			return
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	if err := zipWriter.Close(); err != nil {
		http.Error(w, "failed to download legal hold", http.StatusInternalServerError)
		p.Client.Log.Error(err.Error())
		return
	}
}

func (p *Plugin) runJobFromAPI(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte("Processing all Legal Holds. Please check the MM server logs for more details."))
	if err != nil {
		p.API.LogError("failed to write http response", err.Error())
	}

	go p.legalHoldJob.RunFromAPI()
}

func (p *Plugin) runSingleLegalHold(w http.ResponseWriter, r *http.Request) {
	legalholdID, err := RequireLegalHoldID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = p.legalHoldJob.RunSingleLegalHold(legalholdID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to run legal hold: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	response := struct {
		Message string `json:"message"`
	}{
		Message: fmt.Sprintf("Processing Legal Hold %s. Please check the MM server logs for more details.", legalholdID),
	}

	b, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error encoding json", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		p.API.LogError("failed to write http response", err.Error())
	}
}

// resetLegalHoldStatus resets the status of a LegalHold to Idle
func (p *Plugin) resetLegalHoldStatus(w http.ResponseWriter, r *http.Request) {
	legalholdID, err := RequireLegalHoldID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = p.KVStore.UpdateLegalHoldStatus(legalholdID, model.LegalHoldStatusIdle)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset legal hold status: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	response := struct {
		Message string `json:"message"`
	}{
		Message: fmt.Sprintf("Successfully reset status for Legal Hold %s", legalholdID),
	}

	b, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error encoding json", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		p.API.LogError("failed to write http response", err.Error())
	}
}

// testAmazonS3Connection tests the plugin's custom Amazon S3 connection
func (p *Plugin) testAmazonS3Connection(w http.ResponseWriter, _ *http.Request) {
	type messageResponse struct {
		Message string `json:"message"`
	}

	var err error

	conf := p.getConfiguration()
	if !conf.AmazonS3BucketSettings.Enable {
		http.Error(w, "Amazon S3 bucket settings are not enabled", http.StatusBadRequest)
		return
	}

	filesBackendSettings := FixedFileSettingsToFileBackendSettings(conf.AmazonS3BucketSettings.Settings)
	filesBackend, err := filestore.NewFileBackend(filesBackendSettings)
	if err != nil {
		err = errors.Wrap(err, "unable to initialize the file store")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		p.Client.Log.Error(err.Error())
		return
	}

	if err = filesBackend.TestConnection(); err != nil {
		err = errors.Wrap(err, "failed to connect to Amazon S3 bucket")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		p.Client.Log.Error(err.Error())
		return
	}

	response := messageResponse{
		Message: "Successfully connected to Amazon S3 bucket",
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		p.Client.Log.Error("failed to write http response", err.Error())
	}
}

func RequireLegalHoldID(r *http.Request) (string, error) {
	props := mux.Vars(r)

	legalholdID := props["legalhold_id"]

	if !mattermostModel.IsValidId(legalholdID) {
		return "", errors.New("a valid legal hold ID must be provided")
	}

	return legalholdID, nil
}

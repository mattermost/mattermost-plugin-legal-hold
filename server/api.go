package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/filebackend"
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
	router.HandleFunc("/api/v1/legalhold/list", p.listLegalHolds).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/legalhold/create", p.createLegalHold).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/legalhold/{legalhold_id:[A-Za-z0-9]+}/release", p.releaseLegalHold).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/legalhold/{legalhold_id:[A-Za-z0-9]+}/update", p.updateLegalHold).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/legalhold/{legalhold_id:[A-Za-z0-9]+}/download", p.downloadLegalHold).Methods(http.MethodGet)
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
	// TODO: Validate all the provided data here?

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

	if err = updateLegalHold.IsValid(); err != nil {
		http.Error(w, "LegalHold update data is not valid", http.StatusBadRequest)
		p.Client.Log.Error(err.Error())
		return
	}

	// Retrieve the legal hold we are modifying
	originalLegalHold, err := p.KVStore.GetLegalHoldByID(legalholdID)
	if err != nil {
		http.Error(w, "failed to update legal hold", http.StatusInternalServerError)
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

	// If the Amazon S3 bucket is enabled, generate the download on S3.
	if p.configuration.GenerateDownloadOnFilestore {
		mattermostUserID := r.Header.Get("Mattermost-User-Id")

		channel, appErr := p.API.GetDirectChannel(p.botUserID, mattermostUserID)
		if appErr != nil {
			http.Error(w, "failed to download legal hold", http.StatusInternalServerError)
			p.Client.Log.Error(appErr.Error())
			return
		}

		initialPost, appErr := p.API.CreatePost(&mattermostModel.Post{
			UserId:    p.botUserID,
			ChannelId: channel.Id,
			Message:   "Generating legal hold bundle in the background as per the request. You will be notified once the download is ready.",
		})
		if appErr != nil {
			http.Error(w, "failed to download legal hold", http.StatusInternalServerError)
			p.Client.Log.Error(appErr.Error())
			return
		}

		p.Client.Log.Info("Generating legal hold bundle on S3")
		go func() {
			err := p.KVStore.LockLegalHold(legalholdID)
			if err != nil {
				p.Client.Log.Error("failed to lock legal hold before download task", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			defer func() {
				if err := p.KVStore.UnlockLegalHold(legalholdID); err != nil {
					p.Client.Log.Error("failed to unlock legal hold after download task", err.Error())
				}
			}()

			// Use a custom writter to use the Write/Append functions while we generate the boject on
			// the fly and avoiding to store it on the local disk.
			// Also usign a buffer to avoid writing to the S3 object in small chunks, since the minimal
			// size for a source in the underneath `ComposeObject` call is 5MB, so using 5MB as buffer size.
			filename := filepath.Join(model.FilestoreBundlePath, fmt.Sprintf("%s_%d.zip", legalholdID, time.Now().Unix()))
			zipWriter := zip.NewWriter(
				bufio.NewWriterSize(
					filebackend.NewFileBackendWritter(p.FileBackend, filename),
					1024*1024*5, // 5MB
				))

			bytesWritten := int64(0)
			for _, entry := range files {
				header := &zip.FileHeader{
					Name:     entry,
					Method:   zip.Deflate, // deflate also works, but at a cost
					Modified: time.Now(),
				}

				entryWriter, err := zipWriter.CreateHeader(header)
				if err != nil {
					p.Client.Log.Error(err.Error())
					return
				}

				backendReader, err := p.FileBackend.Reader(entry)
				if err != nil {
					p.Client.Log.Error(err.Error())
					return
				}

				fileReader := bufio.NewReader(backendReader)

				loopBytesWritten, err := io.Copy(entryWriter, fileReader)
				if err != nil {
					p.Client.Log.Error(err.Error())
					return
				}
				bytesWritten += loopBytesWritten
			}

			if err := zipWriter.Close(); err != nil {
				p.Client.Log.Error(err.Error())
				return
			}

			_, appErr = p.API.CreatePost(&mattermostModel.Post{
				UserId:    p.botUserID,
				ChannelId: channel.Id,
				RootId:    initialPost.Id,
				Message:   fmt.Sprintf("Legal hold bundle is ready for download. You can find it under `%s` in your storage provider.", filename),
			})
			if appErr != nil {
				p.Client.Log.Error(appErr.Error())
				return
			}
		}()

		w.WriteHeader(http.StatusCreated)
		return
	}

	// Write headers for the zip file.
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", "legalholddata.zip"))
	w.WriteHeader(http.StatusOK)

	zipWriter := zip.NewWriter(w)
	// Write the files to the download on-the-fly.
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

	filesBackendSettings := FixedFileSettingsToFileBackendSettings(conf.AmazonS3BucketSettings.Settings, false, true)
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

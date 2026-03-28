package main

import (
	"encoding/json"
	"net/http"
	"docker-ui/auth"
	"docker-ui/docker"
	"docker-ui/ws"
	"github.com/docker/docker/errdefs"
	"github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/api/auth/status", AuthStatusHandler).Methods("GET")
	r.HandleFunc("/api/auth/setup", AuthSetupHandler).Methods("POST")
	r.HandleFunc("/api/auth/login", AuthLoginHandler).Methods("POST")

	api := r.PathPrefix("/api").Subrouter()
	api.Use(authService.Middleware)
	api.HandleFunc("/auth/logout", AuthLogoutHandler).Methods("POST")

	// Container routes
	api.HandleFunc("/containers", ListContainersHandler).Methods("GET")
	api.HandleFunc("/containers/{id}/start", StartContainerHandler).Methods("POST")
	api.HandleFunc("/containers/{id}/stop", StopContainerHandler).Methods("POST")
	api.HandleFunc("/containers/{id}/restart", RestartContainerHandler).Methods("POST")
	api.HandleFunc("/containers/{id}/remove", RemoveContainerHandler).Methods("DELETE")
	api.HandleFunc("/containers/{id}/inspect", InspectContainerHandler).Methods("GET")
	api.HandleFunc("/containers/prune", PruneContainersHandler).Methods("POST")

	// Image routes
	api.HandleFunc("/images", ListImagesHandler).Methods("GET")
	api.HandleFunc("/images/{id}", RemoveImageHandler).Methods("DELETE")
	api.HandleFunc("/images/prune", PruneImagesHandler).Methods("POST")

	// Volume routes
	api.HandleFunc("/volumes", ListVolumesHandler).Methods("GET")
	api.HandleFunc("/volumes/{id}", RemoveVolumeHandler).Methods("DELETE")
	api.HandleFunc("/volumes/prune", PruneVolumesHandler).Methods("POST")

	// Network routes
	api.HandleFunc("/networks", ListNetworksHandler).Methods("GET")
	api.HandleFunc("/networks/{id}", RemoveNetworkHandler).Methods("DELETE")
	api.HandleFunc("/networks/prune", PruneNetworksHandler).Methods("POST")

	// Stats routes
	api.HandleFunc("/info", SystemInfoHandler).Methods("GET")
	api.HandleFunc("/disk-usage", DiskUsageHandler).Methods("GET")
	api.HandleFunc("/dashboard/metrics", DashboardMetricsHandler).Methods("GET")

	// Compose routes
	api.HandleFunc("/compose/projects", ListComposeProjectsHandler).Methods("GET")
	api.HandleFunc("/compose/projects/{name}/start", StartComposeProjectHandler).Methods("POST")
	api.HandleFunc("/compose/projects/{name}/stop", StopComposeProjectHandler).Methods("POST")
	api.HandleFunc("/compose/projects/{name}/restart", RestartComposeProjectHandler).Methods("POST")
	api.HandleFunc("/compose/projects/{name}/down", DownComposeProjectHandler).Methods("DELETE")
	api.HandleFunc("/compose/projects/{name}/logs", ComposeProjectLogsHandler).Methods("GET")
	api.HandleFunc("/compose/projects/{name}/files", ComposeProjectFilesHandler).Methods("GET")
	api.HandleFunc("/compose/projects/{name}/files/validate", ValidateComposeProjectFileHandler).Methods("POST")
	api.HandleFunc("/compose/projects/{name}/files", UpdateComposeProjectFileHandler).Methods("PUT")

	// WebSocket routes
	r.HandleFunc("/ws/logs/{id}", ws.LogsHandler)
	r.HandleFunc("/ws/terminal/{id}", ws.TerminalHandler)

	return r
}

func ListContainersHandler(w http.ResponseWriter, r *http.Request) {
	containers, err := docker.ListContainers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(containers)
}

func StartContainerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := docker.StartContainer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func StopContainerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := docker.StopContainer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func RestartContainerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := docker.RestartContainer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func RemoveContainerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := docker.RemoveContainer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func InspectContainerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	info, err := docker.InspectContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(info)
}

func PruneContainersHandler(w http.ResponseWriter, r *http.Request) {
	report, err := docker.PruneContainers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(report)
}

func ListImagesHandler(w http.ResponseWriter, r *http.Request) {
	images, err := docker.ListImages()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(images)
}

func RemoveImageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := docker.RemoveImage(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func PruneImagesHandler(w http.ResponseWriter, r *http.Request) {
	report, err := docker.PruneImages()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(report)
}

func ListVolumesHandler(w http.ResponseWriter, r *http.Request) {
	volumes, err := docker.ListVolumes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(volumes)
}

func RemoveVolumeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := docker.RemoveVolume(id); err != nil {
		status := http.StatusInternalServerError
		if errdefs.IsConflict(err) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func PruneVolumesHandler(w http.ResponseWriter, r *http.Request) {
	report, err := docker.PruneVolumes()
	if err != nil {
		status := http.StatusInternalServerError
		if errdefs.IsConflict(err) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	json.NewEncoder(w).Encode(report)
}

func ListNetworksHandler(w http.ResponseWriter, r *http.Request) {
	networks, err := docker.ListNetworks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(networks)
}

func RemoveNetworkHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := docker.RemoveNetwork(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func PruneNetworksHandler(w http.ResponseWriter, r *http.Request) {
	report, err := docker.PruneNetworks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(report)
}

func SystemInfoHandler(w http.ResponseWriter, r *http.Request) {
	info, err := docker.GetSystemInfo()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(info)
}

func DiskUsageHandler(w http.ResponseWriter, r *http.Request) {
	usage, err := docker.GetDiskUsageSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(usage)
}

func DashboardMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := docker.GetDashboardMetrics(36)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(metrics)
}

func CurrentUser(r *http.Request) *auth.User {
	return auth.CurrentUser(r)
}

func ListComposeProjectsHandler(w http.ResponseWriter, r *http.Request) {
	projects, err := docker.ListComposeProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(projects)
}

func ValidateComposeProjectFileHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var payload struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := docker.ValidateComposeProjectFile(name, payload.Path, payload.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func StartComposeProjectHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := docker.StartComposeProject(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func StopComposeProjectHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := docker.StopComposeProject(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func RestartComposeProjectHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := docker.RestartComposeProject(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DownComposeProjectHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := docker.DownComposeProject(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func ComposeProjectLogsHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	tail := r.URL.Query().Get("tail")

	logs, err := docker.GetComposeProjectLogs(name, tail)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(logs))
}

func ComposeProjectFilesHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	files, err := docker.GetComposeProjectFiles(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(files)
}

func UpdateComposeProjectFileHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var payload struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}
	if payload.Path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	if err := docker.UpdateComposeProjectFile(name, payload.Path, payload.Content); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

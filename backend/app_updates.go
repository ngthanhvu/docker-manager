package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"docker-ui/auth"
	"docker-ui/docker"
	"docker-ui/ws"
	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gorilla/websocket"
)

type dockerHubTag struct {
	Name          string `json:"name"`
	TagLastPushed string `json:"tag_last_pushed"`
	LastUpdated   string `json:"last_updated"`
}

type dockerHubTagsResponse struct {
	Results []dockerHubTag `json:"results"`
}

type appUpdateCheckResponse struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	UpdateURL      string `json:"updateUrl"`
	CheckedAt      string `json:"checkedAt"`
	ReleaseDate    string `json:"releaseDate,omitempty"`
	Message        string `json:"message"`
	ImageName      string `json:"imageName"`
}

type appUpdateApplyRequest struct {
	Namespace     string `json:"namespace"`
	RepoPrefix    string `json:"repoPrefix"`
	TargetVersion string `json:"targetVersion"`
}

type appUpdateApplyResponse struct {
	Started       bool   `json:"started"`
	TargetVersion string `json:"targetVersion"`
	Message       string `json:"message"`
}

type appUpdateStatusResponse struct {
	InProgress    bool   `json:"inProgress"`
	TargetVersion string `json:"targetVersion,omitempty"`
	Message       string `json:"message"`
	StartedAt     string `json:"startedAt,omitempty"`
	FinishedAt    string `json:"finishedAt,omitempty"`
	Succeeded     bool   `json:"succeeded"`
}

var appUpdateHTTPClient = &http.Client{Timeout: 10 * time.Second}
var appUpdateApplyState struct {
	sync.Mutex
	inProgress bool
}
var appUpdateLogState = newUpdateLogState()

type updateLogState struct {
	sync.Mutex
	inProgress    bool
	targetVersion string
	startedAt     time.Time
	finishedAt    time.Time
	succeeded     bool
	message       string
	buffer        []string
	bufferBytes   int
	subscribers   map[int]chan string
	nextID        int
}

func newUpdateLogState() *updateLogState {
	return &updateLogState{
		subscribers: make(map[int]chan string),
	}
}

func (s *updateLogState) begin(targetVersion string) {
	s.Lock()
	defer s.Unlock()
	s.inProgress = true
	s.targetVersion = targetVersion
	s.startedAt = time.Now().UTC()
	s.finishedAt = time.Time{}
	s.succeeded = false
	s.message = fmt.Sprintf("Starting Docker Manager update to version %s.", targetVersion)
	s.buffer = nil
	s.bufferBytes = 0
}

func (s *updateLogState) append(text string) {
	if text == "" {
		return
	}

	s.Lock()
	s.buffer = append(s.buffer, text)
	s.bufferBytes += len(text)
	for s.bufferBytes > 256*1024 && len(s.buffer) > 1 {
		s.bufferBytes -= len(s.buffer[0])
		s.buffer = s.buffer[1:]
	}
	if lastLine := latestLogLine(text); lastLine != "" {
		s.message = lastLine
	}
	subscribers := make([]chan string, 0, len(s.subscribers))
	for _, ch := range s.subscribers {
		subscribers = append(subscribers, ch)
	}
	s.Unlock()

	for _, ch := range subscribers {
		select {
		case ch <- text:
		default:
		}
	}
}

func (s *updateLogState) finish(success bool, message string) {
	s.Lock()
	defer s.Unlock()
	s.inProgress = false
	s.finishedAt = time.Now().UTC()
	s.succeeded = success
	if strings.TrimSpace(message) != "" {
		s.message = strings.TrimSpace(message)
	}
}

func (s *updateLogState) subscribe() (int, chan string, string) {
	s.Lock()
	defer s.Unlock()
	id := s.nextID
	s.nextID++
	ch := make(chan string, 64)
	s.subscribers[id] = ch
	return id, ch, strings.Join(s.buffer, "")
}

func (s *updateLogState) unsubscribe(id int) {
	s.Lock()
	if _, ok := s.subscribers[id]; ok {
		delete(s.subscribers, id)
	}
	s.Unlock()
}

func (s *updateLogState) status() appUpdateStatusResponse {
	s.Lock()
	defer s.Unlock()

	resp := appUpdateStatusResponse{
		InProgress:    s.inProgress,
		TargetVersion: s.targetVersion,
		Message:       s.message,
		Succeeded:     s.succeeded,
	}
	if !s.startedAt.IsZero() {
		resp.StartedAt = s.startedAt.Format(time.RFC3339)
	}
	if !s.finishedAt.IsZero() {
		resp.FinishedAt = s.finishedAt.Format(time.RFC3339)
	}
	return resp
}

func latestLogLine(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return ""
}

type appUpdateLogWriter struct{}

func (w *appUpdateLogWriter) Write(p []byte) (int, error) {
	appUpdateLogState.append(string(p))
	return len(p), nil
}

func isIgnorableUpdateLogError(err error) bool {
	if err == nil {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "use of closed network connection") ||
		strings.Contains(message, "context canceled") ||
		strings.Contains(message, "response body closed") ||
		strings.Contains(message, "file already closed") ||
		message == "eof"
}

func CheckAppUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	currentVersion := strings.TrimSpace(r.URL.Query().Get("currentVersion"))
	if currentVersion == "" {
		currentVersion = "0.0.0"
	}

	namespace := strings.TrimSpace(r.URL.Query().Get("namespace"))
	if namespace == "" {
		namespace = "ngthanhvu"
	}

	repoPrefix := strings.TrimSpace(r.URL.Query().Get("repoPrefix"))
	if repoPrefix == "" {
		repoPrefix = "docker-manager"
	}

	result, err := checkDockerHubFrontendUpdate(currentVersion, namespace, repoPrefix)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func GetAppUpdateStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appUpdateLogState.status())
}

func ApplyAppUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var payload appUpdateApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	namespace := strings.TrimSpace(payload.Namespace)
	if namespace == "" {
		namespace = "ngthanhvu"
	}

	repoPrefix := strings.TrimSpace(payload.RepoPrefix)
	if repoPrefix == "" {
		repoPrefix = "docker-manager"
	}

	targetVersion := normalizeVersion(payload.TargetVersion)
	if targetVersion == "" {
		result, err := checkDockerHubFrontendUpdate("0.0.0", namespace, repoPrefix)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		targetVersion = normalizeVersion(result.LatestVersion)
	}

	if targetVersion == "" {
		http.Error(w, "unable to determine target version", http.StatusBadRequest)
		return
	}

	appUpdateApplyState.Lock()
	if appUpdateApplyState.inProgress {
		appUpdateApplyState.Unlock()
		http.Error(w, "an update is already in progress", http.StatusConflict)
		return
	}
	appUpdateApplyState.inProgress = true
	appUpdateApplyState.Unlock()
	appUpdateLogState.begin(targetVersion)
	appUpdateLogState.append(fmt.Sprintf("[update] preparing version %s\n", targetVersion))

	go func() {
		defer func() {
			appUpdateApplyState.Lock()
			appUpdateApplyState.inProgress = false
			appUpdateApplyState.Unlock()
		}()

		if err := applySelfUpdate(namespace, repoPrefix, targetVersion); err != nil {
			appUpdateLogState.append(fmt.Sprintf("[error] %v\n", err))
			appUpdateLogState.finish(false, err.Error())
			log.Printf("App update failed: %v", err)
			return
		}

		appUpdateLogState.append(fmt.Sprintf("[done] update commands finished for version %s\n", targetVersion))
		appUpdateLogState.finish(true, fmt.Sprintf("Update commands finished for version %s.", targetVersion))
		log.Printf("App update started successfully for version %s", targetVersion)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appUpdateApplyResponse{
		Started:       true,
		TargetVersion: targetVersion,
		Message:       fmt.Sprintf("Started updating Docker Manager to version %s. The UI may reconnect while containers are being recreated.", targetVersion),
	})
}

func AppUpdateLogsWSHandler(w http.ResponseWriter, r *http.Request) {
	if ws.RequestAuthorizer != nil {
		if err := ws.RequestAuthorizer(r); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, auth.ErrUnauthorized) {
				status = http.StatusUnauthorized
			}
			http.Error(w, err.Error(), status)
			return
		}
	}

	conn, err := ws.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade update log connection: %v", err)
		return
	}
	defer conn.Close()

	subID, ch, snapshot := appUpdateLogState.subscribe()
	defer appUpdateLogState.unsubscribe(subID)

	if snapshot != "" {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(snapshot)); err != nil {
			return
		}
	}

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-readDone:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				return
			}
		}
	}
}

func checkDockerHubFrontendUpdate(currentVersion string, namespace string, repoPrefix string) (*appUpdateCheckResponse, error) {
	imageName := fmt.Sprintf("%s/%s-frontend", namespace, repoPrefix)
	updateURL := fmt.Sprintf("https://hub.docker.com/r/%s/%s-frontend/tags", url.PathEscape(namespace), url.PathEscape(repoPrefix))
	endpoint := fmt.Sprintf(
		"https://hub.docker.com/v2/namespaces/%s/repositories/%s-frontend/tags?page_size=100",
		url.PathEscape(namespace),
		url.PathEscape(repoPrefix),
	)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "docker-manager-update-checker")

	resp, err := appUpdateHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("docker hub responded with status %d", resp.StatusCode)
	}

	var payload dockerHubTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	latest := pickLatestVersionTag(payload.Results)
	if latest == nil {
		return nil, fmt.Errorf("no version tags found for %s", imageName)
	}

	latestVersion := normalizeVersion(latest.Name)
	hasUpdate := compareVersions(latestVersion, currentVersion) > 0

	message := fmt.Sprintf("You are running the latest published frontend image.")
	if hasUpdate {
		message = fmt.Sprintf("Version %s is available for %s.", latestVersion, imageName)
	}

	return &appUpdateCheckResponse{
		CurrentVersion: normalizeVersion(currentVersion),
		LatestVersion:  latestVersion,
		HasUpdate:      hasUpdate,
		UpdateURL:      updateURL,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
		ReleaseDate:    firstNonEmpty(latest.TagLastPushed, latest.LastUpdated),
		Message:        message,
		ImageName:      imageName,
	}, nil
}

func pickLatestVersionTag(tags []dockerHubTag) *dockerHubTag {
	candidates := make([]dockerHubTag, 0, len(tags))
	for _, tag := range tags {
		if isVersionTag(tag.Name) {
			candidates = append(candidates, tag)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return compareVersions(candidates[i].Name, candidates[j].Name) > 0
	})

	return &candidates[0]
}

func isVersionTag(raw string) bool {
	value := normalizeVersion(raw)
	if value == "" {
		return false
	}

	for _, part := range strings.FieldsFunc(strings.SplitN(value, "-", 2)[0], func(r rune) bool {
		return r == '.'
	}) {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}

	return true
}

func normalizeVersion(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(strings.ToLower(raw)), "v")
}

func compareVersions(left string, right string) int {
	a := versionParts(left)
	b := versionParts(right)
	limit := len(a)
	if len(b) > limit {
		limit = len(b)
	}

	for i := 0; i < limit; i++ {
		var av, bv int
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av != bv {
			return av - bv
		}
	}

	return strings.Compare(normalizeVersion(left), normalizeVersion(right))
}

func versionParts(raw string) []int {
	base := strings.SplitN(strings.SplitN(normalizeVersion(raw), "+", 2)[0], "-", 2)[0]
	if base == "" {
		return []int{0}
	}

	items := strings.Split(base, ".")
	parts := make([]int, 0, len(items))
	for _, item := range items {
		n, err := strconv.Atoi(item)
		if err != nil {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, n)
	}

	return parts
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func applySelfUpdate(namespace string, repoPrefix string, targetVersion string) error {
	appUpdateLogState.append("[update] locating current Docker Manager container\n")
	self, err := findSelfContainer()
	if err != nil {
		return err
	}

	labels := self.Config.Labels
	workingDir := strings.TrimSpace(labels["com.docker.compose.project.working_dir"])
	configFiles := parseUpdateConfigFiles(labels["com.docker.compose.project.config_files"])
	if len(configFiles) == 0 && workingDir != "" {
		configFiles = []string{filepath.Join(workingDir, "docker-compose.yml")}
	}
	if workingDir == "" {
		return fmt.Errorf("current Docker Manager instance is not running from a compose working directory")
	}
	if len(configFiles) == 0 {
		return fmt.Errorf("no compose files found for the current Docker Manager instance")
	}

	resolvedFiles := make([]string, 0, len(configFiles))
	for _, path := range configFiles {
		resolved := resolveUpdateComposeFilePath(workingDir, path)
		if resolved == "" {
			continue
		}
		resolvedFiles = append(resolvedFiles, resolved)
	}
	if len(resolvedFiles) == 0 {
		return fmt.Errorf("no usable compose files found for the current Docker Manager instance")
	}
	appUpdateLogState.append(fmt.Sprintf("[update] compose working directory: %s\n", workingDir))
	appUpdateLogState.append(fmt.Sprintf("[update] compose files:\n- %s\n", strings.Join(resolvedFiles, "\n- ")))

	backendImage := fmt.Sprintf("%s/%s-backend:%s", namespace, repoPrefix, targetVersion)
	frontendImage := fmt.Sprintf("%s/%s-frontend:%s", namespace, repoPrefix, targetVersion)
	appUpdateLogState.append(fmt.Sprintf("[update] backend image -> %s\n", backendImage))
	appUpdateLogState.append(fmt.Sprintf("[update] frontend image -> %s\n", frontendImage))

	if err := runComposeHelper(
		workingDir,
		resolvedFiles,
		buildSelfUpdateScript(resolvedFiles, backendImage, frontendImage),
	); err != nil {
		return err
	}

	return nil
}

func findSelfContainer() (dockertypes.ContainerJSON, error) {
	hostname, _ := os.Hostname()
	if strings.TrimSpace(hostname) != "" {
		if inspected, err := docker.Cli.ContainerInspect(docker.Ctx(), hostname); err == nil && inspected.Config != nil {
			if inspected.Config.Labels["com.docker.compose.project"] != "" {
				return inspected, nil
			}
		}
	}

	containers, err := docker.ListContainers()
	if err != nil {
		return dockertypes.ContainerJSON{}, err
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if strings.TrimPrefix(name, "/") == "docker-manager-backend" {
				return docker.Cli.ContainerInspect(docker.Ctx(), c.ID)
			}
		}
	}

	return dockertypes.ContainerJSON{}, fmt.Errorf("could not locate the running docker-manager-backend container")
}

func parseUpdateConfigFiles(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func resolveUpdateComposeFilePath(workingDir string, composePath string) string {
	composePath = strings.TrimSpace(composePath)
	if composePath == "" {
		return ""
	}
	if filepath.IsAbs(composePath) {
		return filepath.Clean(composePath)
	}
	if workingDir == "" {
		return filepath.Clean(composePath)
	}
	return filepath.Clean(filepath.Join(workingDir, composePath))
}

func buildComposeFileArgs(configFiles []string) string {
	parts := make([]string, 0, len(configFiles)*2)
	for _, path := range configFiles {
		parts = append(parts, "-f", shellQuote(path))
	}
	return strings.Join(parts, " ")
}

func buildSelfUpdateScript(configFiles []string, backendImage string, frontendImage string) string {
	var script strings.Builder
	script.WriteString("set -eu\n")
	script.WriteString("set -x\n")
	script.WriteString("changed=0\n")
	script.WriteString(fmt.Sprintf("backend_image=%s\n", shellQuote(backendImage)))
	script.WriteString(fmt.Sprintf("frontend_image=%s\n", shellQuote(frontendImage)))
	script.WriteString("rewrite_file() {\n")
	script.WriteString("  file=\"$1\"\n")
	script.WriteString("  tmp=\"$file.tmp.$$\"\n")
	script.WriteString("  awk -v backend_image=\"$backend_image\" -v frontend_image=\"$frontend_image\" '\n")
	script.WriteString("    function flush_service() {\n")
	script.WriteString("      if (!in_service) { return }\n")
	script.WriteString("      if (!service_had_image && target_image != \"\") {\n")
	script.WriteString("        print service_indent \"  image: \" target_image\n")
	script.WriteString("        changed = 1\n")
	script.WriteString("      }\n")
	script.WriteString("      in_service = 0\n")
	script.WriteString("      service_name = \"\"\n")
	script.WriteString("      target_image = \"\"\n")
	script.WriteString("      service_indent = \"\"\n")
	script.WriteString("      service_had_image = 0\n")
	script.WriteString("    }\n")
	script.WriteString("    {\n")
	script.WriteString("      raw = $0\n")
	script.WriteString("      trimmed = raw\n")
	script.WriteString("      sub(/^[[:space:]]+/, \"\", trimmed)\n")
	script.WriteString("      indent_len = match(raw, /[^ ]/)\n")
	script.WriteString("      if (indent_len == 0) {\n")
	script.WriteString("        indent_len = length(raw)\n")
	script.WriteString("      } else {\n")
	script.WriteString("        indent_len -= 1\n")
	script.WriteString("      }\n")
	script.WriteString("      indent = substr(raw, 1, indent_len)\n")
	script.WriteString("      if (trimmed == \"services:\") {\n")
	script.WriteString("        flush_service()\n")
	script.WriteString("        in_services = 1\n")
	script.WriteString("        print raw\n")
	script.WriteString("        next\n")
	script.WriteString("      }\n")
	script.WriteString("      if (in_services && indent_len == 0 && trimmed != \"\") {\n")
	script.WriteString("        flush_service()\n")
	script.WriteString("        in_services = 0\n")
	script.WriteString("      }\n")
	script.WriteString("      if (in_services && indent_len == 2 && trimmed ~ /:$/ && trimmed !~ /^-/) {\n")
	script.WriteString("        flush_service()\n")
	script.WriteString("        service_name = trimmed\n")
	script.WriteString("        sub(/:$/, \"\", service_name)\n")
	script.WriteString("        in_service = 1\n")
	script.WriteString("        service_indent = indent\n")
	script.WriteString("        if (service_name == \"backend\") target_image = backend_image\n")
	script.WriteString("        else if (service_name == \"frontend\") target_image = frontend_image\n")
	script.WriteString("        else target_image = \"\"\n")
	script.WriteString("        service_had_image = 0\n")
	script.WriteString("        print raw\n")
	script.WriteString("        next\n")
	script.WriteString("      }\n")
	script.WriteString("      if (in_service && target_image != \"\" && indent_len > 2 && trimmed ~ /^image:[[:space:]]*/) {\n")
	script.WriteString("        print indent \"image: \" target_image\n")
	script.WriteString("        service_had_image = 1\n")
	script.WriteString("        changed = 1\n")
	script.WriteString("        next\n")
	script.WriteString("      }\n")
	script.WriteString("      print raw\n")
	script.WriteString("    }\n")
	script.WriteString("    END {\n")
	script.WriteString("      flush_service()\n")
	script.WriteString("      if (changed == 0) {\n")
	script.WriteString("        exit 3\n")
	script.WriteString("      }\n")
	script.WriteString("    }\n")
	script.WriteString("  ' \"$file\" > \"$tmp\" && mv \"$tmp\" \"$file\"\n")
	script.WriteString("}\n")
	script.WriteString("for file in")
	for _, path := range configFiles {
		script.WriteString(" ")
		script.WriteString(shellQuote(path))
	}
	script.WriteString("; do\n")
	script.WriteString("  if rewrite_file \"$file\"; then\n")
	script.WriteString("    changed=1\n")
	script.WriteString("  else\n")
	script.WriteString("    status=$?\n")
	script.WriteString("    rm -f \"$file.tmp.$$\"\n")
	script.WriteString("    if [ \"$status\" -ne 3 ]; then\n")
	script.WriteString("      exit \"$status\"\n")
	script.WriteString("    fi\n")
	script.WriteString("  fi\n")
	script.WriteString("done\n")
	script.WriteString("if [ \"$changed\" -eq 0 ]; then\n")
	script.WriteString("  echo \"could not find backend/frontend image definitions to update in compose files\" >&2\n")
	script.WriteString("  exit 4\n")
	script.WriteString("fi\n")
	script.WriteString(fmt.Sprintf("docker compose %s pull backend frontend\n", buildComposeFileArgs(configFiles)))
	script.WriteString(fmt.Sprintf("docker compose %s up -d backend frontend\n", buildComposeFileArgs(configFiles)))
	return script.String()
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func runComposeHelper(workingDir string, configFiles []string, script string) error {
	if strings.TrimSpace(workingDir) == "" {
		return fmt.Errorf("missing compose working directory")
	}

	helperImage := "docker:cli"
	appUpdateLogState.append(fmt.Sprintf("[update] pulling helper image %s\n", helperImage))
	pullResp, err := docker.Cli.ImagePull(docker.Ctx(), helperImage, dockerimage.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull helper image %s: %w", helperImage, err)
	}
	_, _ = io.Copy(io.Discard, pullResp)
	_ = pullResp.Close()
	appUpdateLogState.append("[update] helper image ready\n")

	helper, err := docker.Cli.ContainerCreate(
		docker.Ctx(),
		&dockercontainer.Config{
			Image:      helperImage,
			Entrypoint: []string{"sh", "-lc"},
			Cmd:        []string{script},
		},
		&dockercontainer.HostConfig{
			Binds: []string{
				"/var/run/docker.sock:/var/run/docker.sock",
				fmt.Sprintf("%s:%s", workingDir, workingDir),
			},
			AutoRemove: false,
		},
		nil,
		nil,
		"",
	)
	if err != nil {
		return fmt.Errorf("create helper container: %w", err)
	}
	defer func() {
		_ = docker.Cli.ContainerRemove(docker.Ctx(), helper.ID, dockercontainer.RemoveOptions{Force: true})
	}()
	appUpdateLogState.append(fmt.Sprintf("[update] helper container created: %s\n", helper.ID[:12]))

	if err := docker.Cli.ContainerStart(docker.Ctx(), helper.ID, dockercontainer.StartOptions{}); err != nil {
		return fmt.Errorf("start helper container: %w", err)
	}
	appUpdateLogState.append("[update] helper container started\n")

	logReader, err := docker.Cli.ContainerLogs(docker.Ctx(), helper.ID, dockercontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		appUpdateLogState.append(fmt.Sprintf("[warn] unable to attach live helper logs: %v\n", err))
	}
	logDone := make(chan struct{})
	if err == nil {
		go func() {
			defer close(logDone)
			defer logReader.Close()
			writer := &appUpdateLogWriter{}
			if _, copyErr := stdcopy.StdCopy(writer, writer, logReader); !isIgnorableUpdateLogError(copyErr) {
				appUpdateLogState.append(fmt.Sprintf("[warn] update log stream closed: %v\n", copyErr))
			}
		}()
	} else {
		close(logDone)
	}

	waitCh, errCh := docker.Cli.ContainerWait(docker.Ctx(), helper.ID, dockercontainer.WaitConditionNotRunning)
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			if logReader != nil {
				logReader.Close()
			}
			<-logDone
			return fmt.Errorf("wait helper container: %w", waitErr)
		}
	case result := <-waitCh:
		if logReader != nil {
			logReader.Close()
		}
		<-logDone
		if result.StatusCode != 0 {
			logs, _ := readHelperLogs(helper.ID)
			return fmt.Errorf("compose update helper exited with status %d: %s", result.StatusCode, strings.TrimSpace(logs))
		}
	}

	appUpdateLogState.append("[done] docker compose pull/up completed\n")

	return nil
}

func readHelperLogs(containerID string) (string, error) {
	reader, err := docker.Cli.ContainerLogs(docker.Ctx(), containerID, dockercontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var output bytes.Buffer
	if _, err := stdcopy.StdCopy(&output, &output, reader); err != nil {
		return "", err
	}
	return output.String(), nil
}

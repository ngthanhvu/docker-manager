package main

import (
	"encoding/json"
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

	"docker-ui/docker"
	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"
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

var appUpdateHTTPClient = &http.Client{Timeout: 10 * time.Second}
var appUpdateApplyState struct {
	sync.Mutex
	inProgress bool
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

	go func() {
		defer func() {
			appUpdateApplyState.Lock()
			appUpdateApplyState.inProgress = false
			appUpdateApplyState.Unlock()
		}()

		if err := applySelfUpdate(namespace, repoPrefix, targetVersion); err != nil {
			log.Printf("App update failed: %v", err)
			return
		}

		log.Printf("App update started successfully for version %s", targetVersion)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appUpdateApplyResponse{
		Started:       true,
		TargetVersion: targetVersion,
		Message:       fmt.Sprintf("Started updating Docker Manager to version %s. The UI may reconnect while containers are being recreated.", targetVersion),
	})
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

	backendImage := fmt.Sprintf("%s/%s-backend:%s", namespace, repoPrefix, targetVersion)
	frontendImage := fmt.Sprintf("%s/%s-frontend:%s", namespace, repoPrefix, targetVersion)

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
	pullResp, err := docker.Cli.ImagePull(docker.Ctx(), helperImage, dockerimage.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull helper image %s: %w", helperImage, err)
	}
	_, _ = io.Copy(io.Discard, pullResp)
	_ = pullResp.Close()

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

	if err := docker.Cli.ContainerStart(docker.Ctx(), helper.ID, dockercontainer.StartOptions{}); err != nil {
		return fmt.Errorf("start helper container: %w", err)
	}

	waitCh, errCh := docker.Cli.ContainerWait(docker.Ctx(), helper.ID, dockercontainer.WaitConditionNotRunning)
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return fmt.Errorf("wait helper container: %w", waitErr)
		}
	case result := <-waitCh:
		if result.StatusCode != 0 {
			logs, _ := readHelperLogs(helper.ID)
			return fmt.Errorf("compose update helper exited with status %d: %s", result.StatusCode, strings.TrimSpace(logs))
		}
	}

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

	raw, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

package docker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
	"gopkg.in/yaml.v3"
)

type ComposeService struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	State   string `json:"state"`
	Status  string `json:"status"`
	Created int64  `json:"created"`
}

type ComposeProject struct {
	Name          string           `json:"name"`
	Status        string           `json:"status"`
	Running       int              `json:"running"`
	Total         int              `json:"total"`
	Services      []ComposeService `json:"services"`
	WorkingDir    string           `json:"workingDir,omitempty"`
	ConfigFiles   string           `json:"configFiles,omitempty"`
	ProjectConfig string           `json:"projectConfig,omitempty"`
}

type ComposeProjectFile struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ComposeValidationResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

func ListComposeProjects() ([]ComposeProject, error) {
	all, err := Cli.ContainerList(Ctx(), container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	grouped := make(map[string]*ComposeProject)

	for _, c := range all {
		project := c.Labels["com.docker.compose.project"]
		if project == "" {
			continue
		}

		if _, ok := grouped[project]; !ok {
			grouped[project] = &ComposeProject{
				Name:          project,
				WorkingDir:    c.Labels["com.docker.compose.project.working_dir"],
				ConfigFiles:   c.Labels["com.docker.compose.project.config_files"],
				ProjectConfig: c.Labels["com.docker.compose.project.environment_file"],
				Services:      []ComposeService{},
			}
		}

		name := strings.TrimPrefix(c.Labels["com.docker.compose.service"], "/")
		if name == "" && len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		state := c.State
		if state == "" {
			state = "unknown"
		}

		grouped[project].Services = append(grouped[project].Services, ComposeService{
			ID:      c.ID,
			Name:    name,
			Image:   c.Image,
			State:   state,
			Status:  c.Status,
			Created: c.Created,
		})
	}

	projects := make([]ComposeProject, 0, len(grouped))
	for _, p := range grouped {
		sort.Slice(p.Services, func(i, j int) bool {
			return p.Services[i].Name < p.Services[j].Name
		})

		running := 0
		for _, s := range p.Services {
			if s.State == "running" {
				running++
			}
		}
		p.Running = running
		p.Total = len(p.Services)

		switch {
		case p.Total == 0:
			p.Status = "empty"
		case p.Running == p.Total:
			p.Status = "running"
		case p.Running == 0:
			p.Status = "stopped"
		default:
			p.Status = "partial"
		}

		projects = append(projects, *p)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

func StartComposeProject(project string) error {
	return applyComposeAction(project, func(id string) error {
		return StartContainer(id)
	})
}

func StopComposeProject(project string) error {
	return applyComposeAction(project, func(id string) error {
		return StopContainer(id)
	})
}

func RestartComposeProject(project string) error {
	return applyComposeAction(project, func(id string) error {
		if err := StopContainer(id); err != nil {
			return err
		}
		return StartContainer(id)
	})
}

func DownComposeProject(project string) error {
	return applyComposeAction(project, func(id string) error {
		return RemoveContainer(id)
	})
}

func GetComposeProjectLogs(project string, tail string) (string, error) {
	containers, err := listProjectContainers(project)
	if err != nil {
		return "", err
	}

	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Created < containers[j].Created
	})

	if tail == "" {
		tail = "200"
	}

	var out strings.Builder
	for _, c := range containers {
		service := c.Labels["com.docker.compose.service"]
		if service == "" {
			if len(c.Names) > 0 {
				service = strings.TrimPrefix(c.Names[0], "/")
			} else {
				service = "unknown-service"
			}
		}
		shortID := c.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		out.WriteString(fmt.Sprintf("===== %s (%s) =====\n", service, shortID))

		stream, err := Cli.ContainerLogs(Ctx(), c.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: true,
			Tail:       tail,
		})
		if err != nil {
			out.WriteString(fmt.Sprintf("Failed to fetch logs: %v\n\n", err))
			continue
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		_, _ = stdcopy.StdCopy(&stdout, &stderr, stream)
		_ = stream.Close()

		if stdout.Len() > 0 {
			out.WriteString(stdout.String())
		}
		if stderr.Len() > 0 {
			out.WriteString(stderr.String())
		}
		out.WriteString("\n")
	}

	return out.String(), nil
}

func GetComposeProjectFiles(project string) ([]ComposeProjectFile, error) {
	containers, err := listProjectContainers(project)
	if err != nil {
		return nil, err
	}

	descriptors := listProjectFiles(containers)
	files := make([]ComposeProjectFile, 0, len(descriptors))
	for _, descriptor := range descriptors {
		f := ComposeProjectFile{
			Path: descriptor.Path,
		}
		content, readErr := os.ReadFile(descriptor.Path)
		if readErr != nil {
			f.Error = readErr.Error()
		} else {
			f.Content = string(content)
		}
		files = append(files, f)
	}

	return files, nil
}

func UpdateComposeProjectFile(project string, path string, content string) error {
	containers, err := listProjectContainers(project)
	if err != nil {
		return err
	}

	allowed := map[string]struct{}{}
	for _, descriptor := range listProjectFiles(containers) {
		allowed[descriptor.Path] = struct{}{}
	}

	if _, ok := allowed[path]; !ok {
		return fmt.Errorf("file %q is not part of compose project %q", path, project)
	}

	if writeErr := os.WriteFile(path, []byte(content), 0o644); writeErr != nil {
		return writeErr
	}

	return nil
}

func ValidateComposeProjectFile(project string, path string, content string) (ComposeValidationResult, error) {
	containers, err := listProjectContainers(project)
	if err != nil {
		return ComposeValidationResult{}, err
	}

	allowed := map[string]struct{}{}
	for _, descriptor := range listProjectFiles(containers) {
		allowed[descriptor.Path] = struct{}{}
	}

	if _, ok := allowed[path]; !ok {
		return ComposeValidationResult{}, fmt.Errorf("file %q is not part of compose project %q", path, project)
	}

	for _, descriptor := range listProjectFiles(containers) {
		source := descriptor.Path
		payload := content
		if descriptor.Path != path {
			raw, readErr := os.ReadFile(descriptor.Path)
			if readErr != nil {
				return ComposeValidationResult{Valid: false, Error: fmt.Sprintf("%s: %v", filepath.Base(descriptor.Path), readErr)}, nil
			}
			payload = string(raw)
			source = descriptor.Path
		}

		if err := validateComposeYAML(payload); err != nil {
			return ComposeValidationResult{
				Valid: false,
				Error: fmt.Sprintf("%s: %v", filepath.Base(source), err),
			}, nil
		}
	}

	return ComposeValidationResult{Valid: true}, nil
}

func applyComposeAction(project string, action func(containerID string) error) error {
	containers, err := listProjectContainers(project)
	if err != nil {
		return err
	}

	var errs []string
	for _, c := range containers {
		if err := action(c.ID); err != nil {
			shortID := c.ID
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}
			errs = append(errs, fmt.Sprintf("%s: %v", shortID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("compose action failed: %s", strings.Join(errs, "; "))
	}

	return nil
}

func listProjectContainers(project string) ([]types.Container, error) {
	args := filters.NewArgs()
	args.Add("label", fmt.Sprintf("com.docker.compose.project=%s", project))

	containers, err := Cli.ContainerList(Ctx(), container.ListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("compose project %q not found", project)
	}
	return containers, nil
}

func parseConfigFiles(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

type composeFileDescriptor struct {
	Path string
}

func listProjectFiles(containers []types.Container) []composeFileDescriptor {
	if len(containers) == 0 {
		return nil
	}

	labels := containers[0].Labels
	composePaths := parseConfigFiles(labels["com.docker.compose.project.config_files"])

	out := make([]composeFileDescriptor, 0, len(composePaths))
	seen := map[string]struct{}{}

	for _, path := range composePaths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, composeFileDescriptor{Path: path})
	}

	return out
}

func validateComposeYAML(content string) error {
	decoder := yaml.NewDecoder(strings.NewReader(content))
	decoder.KnownFields(false)

	decodedDocuments := 0
	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == nil {
			decodedDocuments++
			if node.Kind != 0 && node.Kind != yaml.DocumentNode {
				return fmt.Errorf("invalid YAML document structure")
			}
			if len(node.Content) > 0 {
				root := node.Content[0]
				if root.Kind != yaml.MappingNode && root.Kind != yaml.AliasNode {
					return fmt.Errorf("compose root must be a mapping")
				}
			}
			continue
		}
		if err == io.EOF {
			break
		}
		return err
	}

	if decodedDocuments == 0 && strings.TrimSpace(content) != "" {
		return fmt.Errorf("unable to parse YAML content")
	}

	return nil
}

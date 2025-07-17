package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"gitlab-group-migrator/internal/config"
)

// Group represents a GitLab group (namespace), including visibility and full path metadata.
type Group struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	FullPath   string `json:"full_path"`
	Visibility string `json:"visibility"`
}

// Project represents a GitLab project, including its namespace path and visibility.
type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	Visibility        string `json:"visibility"`
	Description       string `json:"description"`
}

// FetchGroup retrieves group details from the GitLab API by its full path.
func FetchGroup(baseURL, token, groupPath string) (*Group, error) {
	endpoint := fmt.Sprintf("%s/api/v4/groups/%s", baseURL, url.PathEscape(groupPath))
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for group %s: %w", groupPath, err)
	}

	request.Header.Set("PRIVATE-TOKEN", token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("requesting group %s: %w", groupPath, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("API returned status %d fetching group %s: %s", response.StatusCode, groupPath, string(body))
	}

	var groupInfo Group
	if err = json.NewDecoder(response.Body).Decode(&groupInfo); err != nil {
		return nil, fmt.Errorf("decoding group %s response: %w", groupPath, err)
	}

	return &groupInfo, nil
}

// FetchProject retrieves a single project by its full path.
func FetchProject(baseURL, token, projectPath string) (*Project, error) {
	endpoint := fmt.Sprintf("%s/api/v4/projects/%s", baseURL, url.PathEscape(projectPath))
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for project %s: %w", projectPath, err)
	}

	request.Header.Set("PRIVATE-TOKEN", token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("requesting project %s: %w", projectPath, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("API returned status %d fetching project %s: %s", response.StatusCode, projectPath, string(body))
	}

	var projectInfo Project
	if err = json.NewDecoder(response.Body).Decode(&projectInfo); err != nil {
		return nil, fmt.Errorf("decoding project %s response: %w", projectPath, err)
	}

	return &projectInfo, nil
}

// MigrateNamespace recursively moves all subgroups and projects from the source group
// into the target group, preserving hierarchy.
func MigrateNamespace(cfg *config.Config, logger *log.Logger, sourceGroupID, targetGroupID int) {
	subgroupList, err := listSubgroups(cfg.SourceGitlabURL, cfg.SourceAccessToken, sourceGroupID)
	if err != nil {
		logger.Printf("Error listing subgroups of %d: %v", sourceGroupID, err)
	}

	for _, subgroup := range subgroupList {
		logger.Printf("  Subgroup: %s (ID %d)", subgroup.FullPath, subgroup.ID)

		var newGroup *Group
		targetFullPath := fmt.Sprintf("%s/%s", cfg.TargetGroup, subgroup.Path)
		existingGroup, err := FetchGroup(cfg.TargetGitlabURL, cfg.TargetAccessToken, targetFullPath)
		if err == nil {
			newGroup = existingGroup
			logger.Printf("  Reusing existing subgroup %s (ID %d)", newGroup.FullPath, newGroup.ID)
		} else {
			newGroup, err = createSubgroup(cfg, subgroup, targetGroupID)
			if err != nil {
				logger.Printf("Skipping subgroup %s: %v", subgroup.FullPath, err)
				continue
			}
			logger.Printf("  Created subgroup %s (ID %d)", newGroup.FullPath, newGroup.ID)
		}

		MigrateNamespace(cfg, logger, subgroup.ID, newGroup.ID)
	}

	projectList, err := listProjects(cfg.SourceGitlabURL, cfg.SourceAccessToken, sourceGroupID)
	if err != nil {
		logger.Printf("Error listing projects of %d: %v", sourceGroupID, err)
	}

	for _, project := range projectList {
		logger.Printf("  Project: %s", project.PathWithNamespace)
		if err = importProject(cfg, project, targetGroupID); err != nil {
			if strings.Contains(err.Error(), "status 409") {
				logger.Printf("    Already exists, skipping %s", project.PathWithNamespace)
			} else {
				logger.Printf("    Skipping project %s: %v", project.PathWithNamespace, err)
			}

			continue
		}

		logger.Printf("    Imported %s successfully", project.PathWithNamespace)
	}

	logger.Printf("Finished namespace %d => %d", sourceGroupID, targetGroupID)
}

// MigrateSpecificProjects moves a limited list of projects from the source group to the target group
func MigrateSpecificProjects(cfg *config.Config, logger *log.Logger, targetGroupID int) {
	for _, projectPath := range cfg.SpecificProjects {
		fullProjectPath := cfg.SourceGroup + "/" + projectPath

		project, err := FetchProject(cfg.SourceGitlabURL, cfg.SourceAccessToken, fullProjectPath)
		if err != nil {
			logger.Printf("Skipping project %s: %v", fullProjectPath, err)
			continue
		}

		logger.Printf("Importing selected project %s", project.PathWithNamespace)
		if err = importProject(cfg, *project, targetGroupID); err != nil {
			if strings.Contains(err.Error(), "status 409") {
				logger.Printf("Project %s already exists, skipping", project.PathWithNamespace)
			} else {
				logger.Printf("Error importing project %s: %v", project.PathWithNamespace, err)
			}
		}
	}
}

// listSubgroups pages through all subgroups of a given group ID and returns them.
func listSubgroups(baseURL, token string, groupID int) ([]Group, error) {
	var resultList []Group

	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/api/v4/groups/%d/subgroups?per_page=100&page=%d", baseURL, groupID, page)
		request, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request listSubgroups page %d: %w", page, err)
		}

		request.Header.Set("PRIVATE-TOKEN", token)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("requesting subgroups page %d: %w", page, err)
		}

		if response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			return nil, fmt.Errorf("API status %d on listSubgroups page %d: %s", response.StatusCode, page, string(body))
		}

		var pageGroups []Group
		if err = json.NewDecoder(response.Body).Decode(&pageGroups); err != nil {
			response.Body.Close()
			return nil, fmt.Errorf("decoding subgroups page %d: %w", page, err)
		}

		response.Body.Close()
		if len(pageGroups) == 0 {
			break
		}

		resultList = append(resultList, pageGroups...)
	}

	return resultList, nil
}

// listProjects pages through all projects of a given group ID and returns them.
func listProjects(baseURL, token string, groupID int) ([]Project, error) {
	var resultList []Project

	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/api/v4/groups/%d/projects?per_page=100&page=%d", baseURL, groupID, page)
		request, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request listProjects page %d: %w", page, err)
		}

		request.Header.Set("PRIVATE-TOKEN", token)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("requesting projects page %d: %w", page, err)
		}

		if response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			return nil, fmt.Errorf("API status %d on listProjects page %d: %s", response.StatusCode, page, string(body))
		}

		var pageProjects []Project
		if err = json.NewDecoder(response.Body).Decode(&pageProjects); err != nil {
			response.Body.Close()
			return nil, fmt.Errorf("decoding projects page %d: %w", page, err)
		}
		response.Body.Close()

		if len(pageProjects) == 0 {
			break
		}

		resultList = append(resultList, pageProjects...)
	}

	return resultList, nil
}

// createSubgroup sends a request to the GitLab API to create a new subgroup under a parent namespace.
func createSubgroup(cfg *config.Config, sourceGroup Group, parentID int) (*Group, error) {
	endpoint := fmt.Sprintf("%s/api/v4/groups", cfg.TargetGitlabURL)

	payload, err := json.Marshal(map[string]interface{}{
		"name":       sourceGroup.Name,
		"path":       sourceGroup.Path,
		"parent_id":  parentID,
		"visibility": sourceGroup.Visibility,
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling subgroup payload: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("creating request createSubgroup: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("PRIVATE-TOKEN", cfg.TargetAccessToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("requesting createSubgroup: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("API status %d on createSubgroup: %s", response.StatusCode, string(body))
	}

	var newGroup Group
	if err = json.NewDecoder(response.Body).Decode(&newGroup); err != nil {
		return nil, fmt.Errorf("decoding createSubgroup response: %w", err)
	}

	return &newGroup, nil
}

// importProject triggers a server-side import of a project repository into a target namespace.
func importProject(cfg *config.Config, sourceProject Project, parentID int) error {
	repoURL := fmt.Sprintf("%s/%s.git", cfg.SourceGitlabURL, sourceProject.PathWithNamespace)
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("parsing repo URL %s: %w", repoURL, err)
	}

	parsedURL.User = url.UserPassword("oauth2", cfg.SourceAccessToken)

	endpoint := fmt.Sprintf("%s/api/v4/projects", cfg.TargetGitlabURL)
	payload, err := json.Marshal(map[string]interface{}{
		"name":         sourceProject.Name,
		"path":         sourceProject.Path,
		"namespace_id": parentID,
		"import_url":   parsedURL.String(),
		"description":  sourceProject.Description,
		"visibility":   sourceProject.Visibility,
	})
	if err != nil {
		return fmt.Errorf("marshaling importProject payload: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("creating request importProject: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("PRIVATE-TOKEN", cfg.TargetAccessToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("requesting importProject: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("API status %d on importProject: %s", response.StatusCode, string(body))
	}

	return nil
}

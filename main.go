package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config содержит параметры подключения и пути групп для миграции
type Config struct {
	SourceGitlabURL string `yaml:"source_gitlab_url"`
	TargetGitlabURL string `yaml:"target_gitlab_url"`
	SourceToken     string `yaml:"source_token"`
	TargetToken     string `yaml:"target_token"`
	SourceGroupPath string `yaml:"source_group"`
	TargetGroupPath string `yaml:"target_group"`
}

// Group описывает GitLab группу
type Group struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	FullPath   string `json:"full_path"`
	Visibility string `json:"visibility"`
}

// Project описывает GitLab проект
type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	Visibility        string `json:"visibility"`
	Description       string `json:"description"`
}

func main() {
	// Путь к YAML-конфигу
	configFile := flag.String("config", "config.yaml", "путь к файлу конфигурации")
	flag.Parse()

	// Чтение конфигурации
	configData, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Не удалось прочитать конфиг: %v", err)
	}
	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		log.Fatalf("Не удалось разобрать конфиг: %v", err)
	}

	// Нормализация URL
	config.SourceGitlabURL = strings.TrimSuffix(config.SourceGitlabURL, "/")
	if config.TargetGitlabURL == "" {
		config.TargetGitlabURL = config.SourceGitlabURL
	}
	config.TargetGitlabURL = strings.TrimSuffix(config.TargetGitlabURL, "/")

	// Токен назначения по умолчанию равен токену источника
	if config.TargetToken == "" {
		config.TargetToken = config.SourceToken
	}

	// Получение исходной и целевой групп
	sourceGroup := fetchGroup(config.SourceGitlabURL, config.SourceToken, config.SourceGroupPath)
	targetGroup := fetchGroup(config.TargetGitlabURL, config.TargetToken, config.TargetGroupPath)

	// Запуск миграции
	if err := migrateNamespace(config, sourceGroup.ID, targetGroup.ID); err != nil {
		log.Fatalf("Миграция завершилась ошибкой: %v", err)
	}
	fmt.Println("Миграция успешно завершена.")
}

// fetchGroup запрашивает информацию о группе по полному пути
func fetchGroup(baseURL, token, groupPath string) Group {
	endpoint := fmt.Sprintf("%s/api/v4/groups/%s", baseURL, url.PathEscape(groupPath))
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		log.Fatalf("Ошибка создания запроса: %v", err)
	}
	request.Header.Set("PRIVATE-TOKEN", token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatalf("Ошибка запроса группы: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		log.Fatalf("Не удалось получить группу: %s", string(body))
	}

	var groupInfo Group
	if err := json.NewDecoder(response.Body).Decode(&groupInfo); err != nil {
		log.Fatalf("Ошибка разбора ответа группы: %v", err)
	}
	return groupInfo
}

// migrateNamespace рекурсивно переносит подгруппы и проекты
func migrateNamespace(config Config, sourceGroupID, targetGroupID int) error {
	// Миграция подгрупп
	subgroupList := listSubgroups(config.SourceGitlabURL, config.SourceToken, sourceGroupID)
	for _, subgroup := range subgroupList {
		newGroup, err := createSubgroup(config.TargetGitlabURL, config.TargetToken, subgroup, targetGroupID)
		if err != nil {
			return fmt.Errorf("ошибка создания подгруппы %s: %w", subgroup.FullPath, err)
		}
		if err := migrateNamespace(config, subgroup.ID, newGroup.ID); err != nil {
			return err
		}
	}

	// Миграция проектов
	projectList := listProjects(config.SourceGitlabURL, config.SourceToken, sourceGroupID)
	for _, project := range projectList {
		if err := importProject(config, project, targetGroupID); err != nil {
			return fmt.Errorf("ошибка импорта проекта %s: %w", project.PathWithNamespace, err)
		}
	}
	return nil
}

// listSubgroups возвращает список подгрупп с постраничной навигацией
func listSubgroups(baseURL, token string, groupID int) []Group {
	var result []Group
	for pageNumber := 1; ; pageNumber++ {
		endpoint := fmt.Sprintf("%s/api/v4/groups/%d/subgroups?per_page=100&page=%d", baseURL, groupID, pageNumber)
		request, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			log.Fatalf("Ошибка создания запроса подгрупп: %v", err)
		}
		request.Header.Set("PRIVATE-TOKEN", token)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			log.Fatalf("Ошибка списка подгрупп: %v", err)
		}

		if response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			log.Fatalf("Не удалось получить подгруппы: %s", string(body))
		}

		var pageGroups []Group
		if err := json.NewDecoder(response.Body).Decode(&pageGroups); err != nil {
			response.Body.Close()
			log.Fatalf("Ошибка разбора подгрупп: %v", err)
		}
		response.Body.Close()

		if len(pageGroups) == 0 {
			break
		}
		result = append(result, pageGroups...)
	}
	return result
}

// listProjects возвращает список проектов с постраничной навигацией
func listProjects(baseURL, token string, groupID int) []Project {
	var result []Project
	for pageNumber := 1; ; pageNumber++ {
		endpoint := fmt.Sprintf("%s/api/v4/groups/%d/projects?per_page=100&page=%d", baseURL, groupID, pageNumber)
		request, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			log.Fatalf("Ошибка создания запроса проектов: %v", err)
		}
		request.Header.Set("PRIVATE-TOKEN", token)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			log.Fatalf("Ошибка списка проектов: %v", err)
		}

		if response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			log.Fatalf("Не удалось получить проекты: %s", string(body))
		}

		var pageProjects []Project
		if err := json.NewDecoder(response.Body).Decode(&pageProjects); err != nil {
			response.Body.Close()
			log.Fatalf("Ошибка разбора проектов: %v", err)
		}
		response.Body.Close()

		if len(pageProjects) == 0 {
			break
		}
		result = append(result, pageProjects...)
	}
	return result
}

// createSubgroup создаёт новую подгруппу в целевом пространстве имён
func createSubgroup(baseURL, token string, sourceGroup Group, parentID int) (Group, error) {
	endpoint := fmt.Sprintf("%s/api/v4/groups", baseURL)
	payload, err := json.Marshal(map[string]interface{}{
		"name":       sourceGroup.Name,
		"path":       sourceGroup.Path,
		"parent_id":  parentID,
		"visibility": sourceGroup.Visibility,
	})
	if err != nil {
		return Group{}, err
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return Group{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("PRIVATE-TOKEN", token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return Group{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		return Group{}, fmt.Errorf("не удалось создать подгруппу: %s", string(body))
	}

	var newGroup Group
	if err := json.NewDecoder(response.Body).Decode(&newGroup); err != nil {
		return Group{}, err
	}
	return newGroup, nil
}

// importProject запускает импорт проекта из исходного репозитория
func importProject(config Config, project Project, targetID int) error {
	repoURL := fmt.Sprintf("%s/%s.git", config.SourceGitlabURL, project.PathWithNamespace)
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return err
	}
	parsedURL.User = url.UserPassword("oauth2", config.SourceToken)

	endpoint := fmt.Sprintf("%s/api/v4/projects", config.TargetGitlabURL)
	payload, err := json.Marshal(map[string]interface{}{
		"name":         project.Name,
		"path":         project.Path,
		"namespace_id": targetID,
		"import_url":   parsedURL.String(),
		"description":  project.Description,
		"visibility":   project.Visibility,
	})
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("PRIVATE-TOKEN", config.TargetToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("не удалось импортировать проект %s: %s", project.PathWithNamespace, string(body))
	}
	return nil
}

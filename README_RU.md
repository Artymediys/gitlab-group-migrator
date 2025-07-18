# GGM – GitLab Group Migrator
Приложение для переноса групповых проектов между GitLab-инстансами.  
Также поддерживается перенос в рамках одного инстанса.

## Пререквизиты
-  В `admin area` открыть сетевой доступ к **source-gitlab**
   - В `settings` => `network` => `outbound requests` поставить галочку на `Allow requests to the local network from webhooks and integrations` и вписать адрес **source-gitlab**

-  В `admin area` выдать разрешение на **import/export** для репозиториев
   - в `settings` => `general` => `import and export settings` поставить галочку на `Repository by URL`

## Конфигурация
Для корректной работы приложения требуется создать конфиг-файл в формате `yaml`:
```yaml
# URL исходного GitLab (откуда переносим)
source_gitlab_url: "https://source.gitlab.example.com"

# URL целевого GitLab (куда переносим)
# Если пусто — будет использован source_gitlab_url
target_gitlab_url: "https://target.gitlab.example.com"

# API-токен для исходного GitLab
source_access_token: "YOUR_SOURCE_PRIVATE_TOKEN"

# API-токен для целевого GitLab.
# Если пусто — будет использован source_access_token
target_access_token: "YOUR_TARGET_PRIVATE_TOKEN"

# Полный путь исходной группы (source namespace)
# Указывается в формате URL-path группы
source_group: "main-group-name/subgroup-name"

# Полный путь целевой группы (target namespace)
# Указывается в формате URL-path группы
target_group: "main-group-name/subgroup-name"

# Список специальных проектов для миграции
# Проекты указываются в URL-path формате
# Если задать список, то будут перенесены только эти проекты
specific_projects:
   - "subgroup-name/project-a"
   - "project-b"
```

## Сборка и запуск
В директории `./bin` лежат сборки приложения под стандартный набор ОС.

### Запуск собранного приложения
Приложение может быть запущено двумя способами:
1) **С флагом** `./app_name -config=./path/to/config.yaml` – можно указать расположение конфигурационного файла
2) **Без флага** `./app_name` – если файл конфигурации расположен в той же директории, где и исполняемый файл приложения

### Самостоятельная сборка
Для самостоятельной сборки и последующего запуска, требуется установленный **Go** версии `>=1.24.4`.

В корне репозитория находится bash-скрипт `build.sh` для сборки приложения под необходимую ОС и архитектуру.

#### Сборка под стандартный набор ОС
Скрипт можно запустить вместе с аргументами:
1) Сборка приложения под предустановленные ОС и архитектуры, а именно –> **windows amd64**; **linux amd64**;
   **macos amd64** и **arm64**
```shell
./build.sh all
```

2) Сборка приложения под конкретную ОС. Из доступных вариантов -> **windows**; **linux**, **macos**
```shell
./build.sh <ОС>
```

#### Сборка под текущую ОС
Также скрипт можно запустить без аргументов. Тогда он соберёт приложение под текущую ОС и архитектуру,
на которой был запущен скрипт:
```shell
./build.sh
```

#### Сборка под специфичную ОС
Если требуется собрать приложение под ОС или архитектуру, которой нет в стандартном наборе скрипта `build.sh`,
тогда можно посмотреть список доступных для сборки ОС и архитектуры командой:
```shell
go tool dist list
```

После чего выполнить следующую команду для сборки:

`Linux/MacOS`
```shell
env GOOS=<ОС> GOARCH=<архитектура> go build -o <название итогового файла> main.go
```

`Windows/Powershell`
```shell
$env:GOOS="<ОС>"; $env:GOARCH="<архитектура>"; go build -o <название итогового файла> main.go
```

`Windows/cmd`
```shell
set GOOS=<ОС> && set GOARCH=<архитектура> && go build -o <название итогового файла> main.go
```
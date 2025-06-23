# GGM – GitLab Group Migrator
An application for migrating groups between GitLab instances.  
Migration within a single instance is also supported.

## Configuration
To operate correctly, the application requires a YAML configuration file:
```yaml
# URL of the source GitLab (from which to migrate)
source_gitlab_url: "https://source.gitlab.example.com"

# URL of the target GitLab (to which to migrate).
# If empty — source_gitlab_url will be used
target_gitlab_url: "https://target.gitlab.example.com"

# API token for the source GitLab
source_access_token: "YOUR_SOURCE_PRIVATE_TOKEN"

# API token for the target GitLab.
# If empty — source_access_token will be used
target_access_token: "YOUR_TARGET_PRIVATE_TOKEN"

# Full path of the source group (source namespace)
# Specified in the group's URL-path format
source_group: "main-group-name/subgroup-name"

# Full path of the target group (target namespace)
# Specified in the group's URL-path format
target_group: "main-group-name/subgroup-name"
```

## Building and Running
The `./bin` directory contains application builds for a standard set of operating systems.

### Running a Built Application
The application can be run in two ways:
1) **With the flag**  
   `./app_name -config=./path/to/config.yaml` – allows specifying the configuration file location
2) **Without the flag**  
   `./app_name` – if the configuration file is located in the same directory as the executable

### Building from Source
To build and run from source, you need **Go** version `>=1.24.4` installed.

At the root of the repository, there is a bash script `build.sh` for building the application for the required OS and architecture.

#### Building for Standard OS Set
Run the script with arguments:
1) Build the application for the preset OSes and architectures — **windows amd64**; **linux amd64**; **macos amd64**; **arm64**
```shell
./build.sh all
```

2) Build the application for a specific OS. Available options — **windows**; **linux**; **macos**
```shell
./build.sh <OS>
```

#### Building for the Current OS
Run the script without arguments to build for the current OS and architecture:
```shell
./build.sh
```

#### Building for a Specific OS
If you need to build for an OS or architecture not included in the standard set, list available targets with:
```shell
go tool dist list
```
Then build with:

`Linux/macOS`
```shell
env GOOS=<OS> GOARCH=<architecture> go build -o <output_filename> main.go
```

`Windows/PowerShell`
```shell
$env:GOOS="<OS>"; $env:GOARCH="<architecture>"; go build -o <output_filename> main.go
```

`Windows/cmd`
```shell
set GOOS=<OS> && set GOARCH=<architecture> && go build -o <output_filename> main.go
```

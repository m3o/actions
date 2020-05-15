package changedetector

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v30/github"
	"github.com/spf13/afero"
	"golang.org/x/oauth2"
)

var (
	appFS = afero.NewOsFs() // use this for all OS manipulation so we can mock more easily
)

var (
	errGoModNotFound = errors.New("Parent go.mod file not found")
)

type fileToStatus struct {
	fileName string
	status   githubFileChangeStatus
}

type githubFileChangeStatus string

// a list of github file status changes.
// not documented in the github API
var (
	githubFileChangeStatusCreated  githubFileChangeStatus = "added"
	githubFileChangeStatusChanged  githubFileChangeStatus = "changed"
	githubFileChangeStatusModified githubFileChangeStatus = "modified"
	githubFileChangeStatusRemoved  githubFileChangeStatus = "removed"
)

// Status is the status of the service
type Status string

var (
	StatusCreated Status = "created"
	StatusUpdated Status = "updated"
	StatusDeleted Status = "deleted"
)

func New(ghToken, ghRepo, ghOwner, ghSHA string) *ChangeDetector {
	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	))

	return &ChangeDetector{
		client:      github.NewClient(tc),
		githubRepo:  ghRepo,
		githubOwner: ghOwner,
		githubSHA:   ghSHA,
	}
}

type ChangeDetector struct {
	client      *github.Client
	githubRepo  string
	githubOwner string
	githubSHA   string
}

// List uses the github sha to determine all the changes
// It returns a list of folders which we think represent services that have changed.
// For example, if serviceA/main.go and serviceB/handler/handler.go have changed but serviceC/ hasn't it should return serviceA and serviceB
func (cd *ChangeDetector) List() (map[string]Status, error) {
	repo := strings.TrimPrefix(cd.githubRepo, cd.githubOwner+"/")
	commit, _, err := cd.client.Repositories.GetCommit(context.Background(), cd.githubOwner, repo, cd.githubSHA)
	if err != nil {
		return nil, err
	}

	filesToStatuses := []fileToStatus{}
	for _, v := range commit.Files {
		// special case. If m3o.yaml is *added* we should build all the things
		if v.GetFilename() == ".github/workflows/m3o.yaml" && githubFileChangeStatus(v.GetStatus()) == githubFileChangeStatusCreated {
			return findAllGoModDirs(".")
		}

		// skip files starting with . e.g. ".github"
		if strings.HasPrefix(v.GetFilename(), ".") {
			continue
		}

		filesToStatuses = append(filesToStatuses, fileToStatus{
			fileName: v.GetFilename(),
			status:   githubFileChangeStatus(v.GetStatus()),
		})
	}

	return directoryStatuses(filesToStatuses)
}

// findAllGoModDirs records directory of every go.mod file and returns with StatusCreated to force a rebuild of all the things
func findAllGoModDirs(dirPath string) (map[string]Status, error) {
	// search for all go.mod files and record their directories
	ret := map[string]Status{}
	listing, err := afero.ReadDir(appFS, dirPath)
	if err != nil {
		return nil, err
	}
	for _, f := range listing {
		if f.IsDir() {
			statuses, err := findAllGoModDirs(dirPath + "/" + f.Name())
			if err != nil {
				return nil, err
			}
			for k, v := range statuses {
				ret[k] = v
			}
			continue
		}
		if f.Name() == "go.mod" {
			ret[filepath.Clean(dirPath)] = StatusCreated
		}
	}
	return ret, nil
}

// maps github file change statuses to directories (or actually services) and their deployment status
// ie. "asim/scheduler/main.go" "removed" => "asim/scheduler" "deleted"
// "serviceA/handler/handler.go" "modified" => "serviceA" "updafed"
func directoryStatuses(statuses []fileToStatus) (map[string]Status, error) {
	dirs := map[string]Status{}

	// Logic. Assume that go.mod is the root of the service or thing to be built. Note not using main.go because sometimes people like to use something else
	// For every changed file traverse up to find the direct parent go.mod file and record the dir
	// Dedupe so you only build a dir once.
	// If go.mod is created or deleted we record that as the service being created or deleted. Everything else is assumed an update

	for _, status := range statuses {
		fname := status.fileName
		status := status.status
		_, fileName := filepath.Split(fname)

		dir, err := findParentGoModDir(fname)
		if err != nil && err == errGoModNotFound {
			// assume that go mod was deleted, skip
			continue
		}
		if err != nil {
			return nil, err
		}
		if fileName == "go.mod" {
			if status == githubFileChangeStatusCreated {
				dirs[dir] = StatusCreated
			} else if status == githubFileChangeStatusRemoved {
				dirs[dir] = StatusDeleted
			}
		}
		if _, ok := dirs[dir]; !ok {
			dirs[dir] = StatusUpdated
		}

	}
	return dirs, nil
}

func findParentGoModDir(fileName string) (string, error) {
	dir, file := filepath.Split(fileName)
	dir = filepath.Clean(dir)
	if file == "go.mod" {
		return dir, nil
	}
	for {
		listing, _ := afero.ReadDir(appFS, dir) // ignore error because it would indicate that dir is missing which is OK
		for _, f := range listing {
			if f.Name() == "go.mod" {
				return dir, nil
			}
		}
		dir = filepath.Dir(dir)
		if dir == "." {
			break
		}
	}
	return "", errGoModNotFound

}

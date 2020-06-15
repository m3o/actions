package changes

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
	errGoMainNotFound = errors.New("Parent main.go file not found")
)

type fileToStatus struct {
	fileName string
	status   githubFileChangeStatus
}

type githubFileChangeStatus string

// a list of github file status changes.
// Sort of documented here https://developer.github.com/v3/repos/commits/#compare-two-commits
// The response also includes details on the files that were changed between the two commits.
// This includes the status of the change (for example, if a file was added, removed, modified, or renamed), and details of the change itself.
// For example, files with a renamed status have a previous_filename field showing the previous filename of the file, and files with a modified status have a patch field showing the changes made to the file.
var (
	githubFileChangeStatusCreated  githubFileChangeStatus = "added"
	githubFileChangeStatusModified githubFileChangeStatus = "modified"
	githubFileChangeStatusRemoved  githubFileChangeStatus = "removed"
	githubFileChangeStatusRenamed  githubFileChangeStatus = "renamed"
)

// Status is the status of the service
type Status string

var (
	StatusCreated Status = "created"
	StatusUpdated Status = "updated"
	StatusDeleted Status = "deleted"
)

func New(ghToken, ghRepo, ghOwner, ghSHA string) *Changes {
	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	))

	return &Changes{
		client:      github.NewClient(tc),
		githubRepo:  ghRepo,
		githubOwner: ghOwner,
		githubSHA:   ghSHA,
	}
}

type Changes struct {
	client      *github.Client
	githubRepo  string
	githubOwner string
	githubSHA   string
}

// List uses the github sha to determine all the changes
// It returns a list of folders which we think represent services that have changed.
// For example, if serviceA/main.go and serviceB/handler/handler.go have changed but serviceC/ hasn't it should return serviceA and serviceB
func (cd *Changes) List() (map[string]Status, error) {
	repo := strings.TrimPrefix(cd.githubRepo, cd.githubOwner+"/")
	commit, _, err := cd.client.Repositories.GetCommit(context.Background(), cd.githubOwner, repo, cd.githubSHA)
	if err != nil {
		return nil, err
	}
	filesToStatuses := []fileToStatus{}
	for _, v := range commit.Files {
		// special case. If m3o.yaml is *added* we should build all the things
		if v.GetFilename() == ".github/workflows/m3o.yaml" && githubFileChangeStatus(v.GetStatus()) == githubFileChangeStatusCreated {
			return findAllGoMainDirs(".")
		}

		// skip files starting with . e.g. ".github"
		if strings.HasPrefix(v.GetFilename(), ".") {
			continue
		}
		status := githubFileChangeStatus(v.GetStatus())
		// hack. If file is renamed, treat it as a delete and create so we correctly delete and recreate the services
		if githubFileChangeStatus(v.GetStatus()) == githubFileChangeStatusRenamed {
			filesToStatuses = append(filesToStatuses, fileToStatus{
				fileName: v.GetPreviousFilename(),
				status:   githubFileChangeStatusRemoved,
			})
			status = githubFileChangeStatusCreated
		}
		filesToStatuses = append(filesToStatuses, fileToStatus{
			fileName: v.GetFilename(),
			status:   status,
		})
	}

	return directoryStatuses(filesToStatuses)
}

// findAllGoMainDirs records directory of every main.go file and returns with StatusCreated to force a rebuild of all the things
func findAllGoMainDirs(dirPath string) (map[string]Status, error) {
	// search for all main.go files and record their directories
	ret := map[string]Status{}
	listing, err := afero.ReadDir(appFS, dirPath)
	if err != nil {
		return nil, err
	}
	for _, f := range listing {
		if f.IsDir() {
			statuses, err := findAllGoMainDirs(dirPath + "/" + f.Name())
			if err != nil {
				return nil, err
			}
			for k, v := range statuses {
				ret[k] = v
			}
			continue
		}
		if f.Name() == "main.go" {
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

	// Logic. Assume that main.go is the root of the service or thing to be built.
	// For every changed file traverse up to find the direct parent main.go file and record the dir
	// Dedupe so you only build a dir once.
	// If main.go is created or deleted we record that as the service being created or deleted. Everything else is assumed an update

	for _, status := range statuses {
		fname := status.fileName
		status := status.status
		_, fileName := filepath.Split(fname)

		dir, err := findParentGoMainDir(fname)
		if err != nil && err == errGoMainNotFound {
			// assume that go main was deleted, skip
			continue
		}
		if err != nil {
			return nil, err
		}
		if fileName == "main.go" {
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

func findParentGoMainDir(fileName string) (string, error) {
	dir, file := filepath.Split(fileName)
	dir = filepath.Clean(dir)
	if file == "main.go" {
		return dir, nil
	}
	for {
		listing, _ := afero.ReadDir(appFS, dir) // ignore error because it would indicate that dir is missing which is OK
		for _, f := range listing {
			if f.Name() == "main.go" {
				return dir, nil
			}
		}
		dir = filepath.Dir(dir)
		if dir == "." {
			break
		}
	}
	return "", errGoMainNotFound

}

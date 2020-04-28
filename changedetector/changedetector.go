package changedetector

import (
	"context"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v30/github"
	"golang.org/x/oauth2"
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
func (cd *ChangeDetector) List() (map[string]Status, error) {
	repo := strings.TrimPrefix(cd.githubRepo, cd.githubOwner+"/")
	commit, _, err := cd.client.Repositories.GetCommit(context.Background(), cd.githubOwner, repo, cd.githubSHA)
	if err != nil {
		return nil, err
	}

	filesToStatuses := []fileToStatus{}
	for _, v := range commit.Files {
		// skip files starting with . e.g. ".github"
		if strings.HasPrefix(v.GetFilename(), ".") {
			continue
		}

		filesToStatuses = append(filesToStatuses, fileToStatus{
			fileName: v.GetFilename(),
			status:   githubFileChangeStatus(v.GetStatus()),
		})
	}

	return folderStatuses(filesToStatuses), nil
}

// maps github file change statuses to folders and their deployment status
// ie. "asim/scheduler/main.go" "removed" will become "asim/scheduler" "deleted"
func folderStatuses(statuses []fileToStatus) map[string]Status {
	folders := map[string]Status{}
	// Prioritize main.go creates and deletes
	for _, status := range statuses {
		fname := status.fileName
		status := status.status
		if !strings.HasSuffix(fname, "main.go") {
			continue
		}
		fold := path.Dir(fname)

		_, exists := folders[fold]
		if exists {
			continue
		}
		if status == "added" {
			folders[fold] = StatusCreated
		} else if status == "removed" {
			folders[fold] = StatusDeleted
		}

	}
	// continue with normal file changes for service updates
	for _, status := range statuses {
		fname := status.fileName
		// All service files are inside folders,
		// so any file in the top folder can be safely ignored.
		if !strings.Contains(fname, "/") {
			continue
		}
		folds := topFolders(fname)
		for _, fold := range folds {
			_, exists := folders[fold]
			if exists {
				continue
			}
			folders[fold] = StatusUpdated
		}
	}
	return folders
}

// from path returns the top level dirs to be deployed
// ie.
func topFolders(path string) []string {
	parts := strings.Split(path, "/")
	ret := []string{parts[0]}
	if len(parts) > 2 {
		ret = append(ret, filepath.Join(parts[0], parts[1]))
	}
	return ret
}

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

type Status string

var (
	StatusCreated Status = "created"
	StatusUpdated Status = "updated"
	StatusDeleted Status = "deleted"
)

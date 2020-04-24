package main

import (
	"context"
	"path"
	"path/filepath"
	"strings"
)

// ListChangedDirectories uses the commitHash to determine all the changes
func (a *Action) ListChangedDirectories(commitHash string) (map[string]ServiceStatus, error) {
	repo := strings.TrimPrefix(a.githubRepo, a.githubOwner+"/")
	commit, _, err := a.client.Repositories.GetCommit(context.Background(), a.githubOwner, repo, commitHash)
	if err != nil {
		return nil, err
	}

	filesToStatuses := []fileToStatus{}
	for _, v := range commit.Files {
		filesToStatuses = append(filesToStatuses, fileToStatus{
			fileName: v.GetFilename(),
			status:   GithubFileChangeStatus(v.GetStatus()),
		})
	}

	return folderStatuses(filesToStatuses), nil
}

// maps github file change statuses to folders and their deployment status
// ie. "asim/scheduler/main.go" "removed" will become "asim/scheduler" "deleted"
func folderStatuses(statuses []fileToStatus) map[string]ServiceStatus {
	folders := map[string]ServiceStatus{}
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
			folders[fold] = ServiceStatusCreated
		} else if status == "removed" {
			folders[fold] = ServiceStatusDeleted
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
			folders[fold] = ServiceStatusUpdated
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
	status   GithubFileChangeStatus
}

type GithubFileChangeStatus string

// a list of github file status changes.
// not documented in the github API
var (
	GithubFileChangeStatusCreated  GithubFileChangeStatus = "added"
	GithubFileChangeStatusChanged  GithubFileChangeStatus = "changed"
	GithubFileChangeStatusModified GithubFileChangeStatus = "modified"
	GithubFileChangeStatusRemoved  GithubFileChangeStatus = "removed"
)

type ServiceStatus string

var (
	ServiceStatusCreated ServiceStatus = "created"
	ServiceStatusUpdated ServiceStatus = "updated"
	ServiceStatusDeleted ServiceStatus = "deleted"
)

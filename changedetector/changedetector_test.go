package changedetector

import (
	"path"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestFolderStatuses(t *testing.T) {

	tcs := []struct {
		input      []fileToStatus
		goModFiles []string // where do the go.mod files live
		expected   map[string]Status
	}{
		{
			input: []fileToStatus{
				{
					fileName: "serviceA/main.go", status: githubFileChangeStatusModified,
				},
			},
			goModFiles: []string{"serviceA/go.mod"},
			expected:   map[string]Status{"serviceA": StatusUpdated},
		},
		{
			input: []fileToStatus{
				{
					fileName: "serviceA/handler/handler.go", status: githubFileChangeStatusModified,
				},
			},
			goModFiles: []string{"serviceA/go.mod"},
			expected:   map[string]Status{"serviceA": StatusUpdated},
		},
		{
			input: []fileToStatus{
				{
					fileName: "serviceA/proto/serviceA/serviceA.pb.go", status: githubFileChangeStatusModified,
				},
			},
			goModFiles: []string{"serviceA/go.mod"},
			expected:   map[string]Status{"serviceA": StatusUpdated},
		},
		{
			input: []fileToStatus{
				{
					fileName: "serviceA/proto/serviceA/serviceA.pb.go", status: githubFileChangeStatusModified,
				},
				{
					fileName: "serviceB/main.go", status: githubFileChangeStatusModified,
				},
				{
					fileName: "serviceB/dao/dao.go", status: githubFileChangeStatusModified,
				},
			},
			goModFiles: []string{"serviceA/go.mod", "serviceB/go.mod"},
			expected: map[string]Status{
				"serviceA": StatusUpdated,
				"serviceB": StatusUpdated,
			},
		},
		{
			input: []fileToStatus{
				{
					fileName: "foo/nestedServiceA/main.go", status: githubFileChangeStatusModified,
				},
			},
			goModFiles: []string{"foo/nestedServiceA/go.mod"},
			expected: map[string]Status{
				"foo/nestedServiceA": StatusUpdated,
			},
		},
		{
			input: []fileToStatus{
				{
					fileName: "serviceA/go.mod", status: githubFileChangeStatusRemoved,
				},
			},
			expected: map[string]Status{
				"serviceA": StatusDeleted,
			},
		},
		{
			input: []fileToStatus{
				{
					fileName: "serviceA/go.mod", status: githubFileChangeStatusRemoved,
				},
				{
					fileName: "serviceA/main.go", status: githubFileChangeStatusRemoved,
				},
			},
			expected: map[string]Status{
				"serviceA": StatusDeleted,
			},
		},
		{
			input: []fileToStatus{
				{
					fileName: "serviceA/main.go", status: githubFileChangeStatusRemoved,
				},
			},
			goModFiles: []string{"serviceA/go.mod"},
			expected: map[string]Status{
				"serviceA": StatusUpdated, // Updated not deleted because main method might not live in main.go
			},
		},
	}
	for i, tc := range tcs {
		// set up the file system
		appFS = afero.NewMemMapFs()
		for _, fs := range tc.input {
			if fs.status == githubFileChangeStatusRemoved {
				continue
			}
			assert.NoError(t, appFS.MkdirAll(path.Dir(fs.fileName), 0755), "Error setting up file system for test %d", i)
			assert.NoError(t, afero.WriteFile(appFS, fs.fileName, []byte("foobar"), 0755), "Error setting up file system for test %d", i)
		}
		for _, gomod := range tc.goModFiles {
			assert.NoError(t, afero.WriteFile(appFS, gomod, []byte("foobar"), 0755), "Error setting up file system for test %d", i)
		}

		out, err := directoryStatuses(tc.input)
		assert.NoError(t, err, "Error processing directory statuses for test %d", i)
		assert.Equal(t, tc.expected, out, "Failed test case %d", i)
	}

}

package changedetector

import (
	"path"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestFolderStatuses(t *testing.T) {

	tcs := []struct {
		input       []fileToStatus
		goMainFiles []string // where do the main.go files live
		expected    map[string]Status
	}{
		{
			input: []fileToStatus{
				{
					fileName: "serviceA/main.go", status: githubFileChangeStatusModified,
				},
			},
			goMainFiles: []string{"serviceA/main.go"},
			expected:    map[string]Status{"serviceA": StatusUpdated},
		},
		{
			input: []fileToStatus{
				{
					fileName: "serviceA/handler/handler.go", status: githubFileChangeStatusModified,
				},
			},
			goMainFiles: []string{"serviceA/main.go"},
			expected:    map[string]Status{"serviceA": StatusUpdated},
		},
		{
			input: []fileToStatus{
				{
					fileName: "serviceA/proto/serviceA/serviceA.pb.go", status: githubFileChangeStatusModified,
				},
			},
			goMainFiles: []string{"serviceA/main.go"},
			expected:    map[string]Status{"serviceA": StatusUpdated},
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
			goMainFiles: []string{"serviceA/main.go", "serviceB/main.go"},
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
			goMainFiles: []string{"foo/nestedServiceA/main.go"},
			expected: map[string]Status{
				"foo/nestedServiceA": StatusUpdated,
			},
		},
		{
			input: []fileToStatus{
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
				{
					fileName: "serviceA/hander/handler.go", status: githubFileChangeStatusRemoved,
				},
			},
			expected: map[string]Status{
				"serviceA": StatusDeleted,
			},
		},
		{ // this is what a rename of the whole service should look like
			input: []fileToStatus{
				{
					fileName: "serviceA/main.go", status: githubFileChangeStatusRemoved,
				},
				{
					fileName: "serviceARenamed/main.go", status: githubFileChangeStatusCreated,
				},
			},
			expected: map[string]Status{
				"serviceA":        StatusDeleted,
				"serviceARenamed": StatusCreated,
			},
		},
		{ // this is what a rename of a single file, not including the main.go, should look like
			input: []fileToStatus{
				{
					fileName: "serviceA/types/types.go", status: githubFileChangeStatusRemoved,
				},
				{
					fileName: "serviceB/types/types.go", status: githubFileChangeStatusCreated,
				},
			},
			goMainFiles: []string{"serviceA/main.go", "serviceB/main.go"},
			expected: map[string]Status{
				"serviceA": StatusUpdated,
				"serviceB": StatusUpdated,
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
		for _, gomod := range tc.goMainFiles {
			assert.NoError(t, afero.WriteFile(appFS, gomod, []byte("foobar"), 0755), "Error setting up file system for test %d", i)
		}

		out, err := directoryStatuses(tc.input)
		assert.NoError(t, err, "Error processing directory statuses for test %d", i)
		assert.Equal(t, tc.expected, out, "Failed test case %d", i)
	}

}

func TestFindGoMain(t *testing.T) {
	tcs := []struct {
		files    []string
		expected []string
	}{
		{
			files:    []string{"serviceA/main.go", "serviceA/handler/handler.go", "serviceB/main.go"},
			expected: []string{"serviceA", "serviceB"},
		},
		{
			files:    []string{"nested/serviceA/main.go", "nested/serviceA/handler/handler.go", "serviceB/main.go", "nested/nested/serviceC/main.go", "nested/serviceC/some/other/dir/foo.go", "nested/go.mod"},
			expected: []string{"nested/serviceA", "serviceB", "nested/nested/serviceC"},
		},
	}
	for i, tc := range tcs {
		appFS = afero.NewMemMapFs()
		for _, f := range tc.files {
			afero.WriteFile(appFS, f, []byte("foobar"), 0755)
		}
		out, err := findAllGoMainDirs(".")
		assert.NoError(t, err, "Unexpected error finding go main for test %d", i)
		expected := map[string]Status{}
		for _, v := range tc.expected {
			expected[v] = StatusCreated
		}
		assert.Equal(t, expected, out, "Failed test case %d", i)
	}
}

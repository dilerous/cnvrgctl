package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetPods(t *testing.T) {
	testCases := []struct {
		name               string
		pods               []runtime.Object
		targetNamespace    string
		targetPod          string
		targetLabelKey     string
		expectedLabelValue string
		expectSuccess      bool
	}{
		{
			name: "existing_pod_found",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1-existing",
						Namespace: "namespace1",
						Labels: map[string]string{
							"label1": "value1",
						},
					},
				},
			},
			targetNamespace:    "namespace1",
			targetPod:          "pod1",
			targetLabelKey:     "label1",
			expectedLabelValue: "VALUE1",
			expectSuccess:      true,
		},
		{
			name:               "no_pods_existing",
			pods:               []runtime.Object{},
			targetNamespace:    "namespace1",
			targetPod:          "pod1",
			targetLabelKey:     "label1",
			expectedLabelValue: "VALUE1",
			expectSuccess:      false,
		},
		{
			name: "existing_pod_missing_label",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-no-label",
						Namespace: "namespace1",
					},
				},
			},
			targetNamespace:    "namespace1",
			targetPod:          "pod1",
			targetLabelKey:     "label1",
			expectedLabelValue: "VALUE1",
			expectSuccess:      true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			fakeClientset := fake.NewSimpleClientset(test.pods...)

			pods, err := getPods(
				test.targetNamespace,
				fakeClientset,
			)

			for _, pod := range pods {
				if pod.ObjectMeta.Name == "" && test.expectSuccess {
					t.Fatalf("no pod found: %v", err)
				}
			}

			err = getLogs(pods, 10, fakeClientset)
			if err != nil {
				t.Fatalf("logs not gathered: %v", err)
			}

		})
	}
}

func TestCreateTar(t *testing.T) {
	// Call the function to be tested
	err := createTar()
	if err != nil {
		t.Fatalf("Error creating tar: %v", err)
	}

	//Check if the tar file was created
	if _, err := os.Stat("logs.tar.gz"); os.IsNotExist(err) {
		t.Fatalf("Expected tar file logs.tar.gz does not exist")
	}

	// Optionally, you can also check the contents of the tar file if needed
	// You may want to extract and inspect the contents of the tar file in your test
	// to ensure that it contains the expected files or directories.
}

func TestCreateTarError(t *testing.T) {
	// Create a directory with invalid permissions to force an error
	err := os.Mkdir("testDir", 0000)
	if err != nil {
		t.Fatalf("Error creating test directory: %v", err)
	}
	defer os.RemoveAll("testDir")

	// Call the function to be tested
	err = createTar()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	actualErrorMessage := err.Error()
	expectedError := "error creating the tar file, logs.tar.gz. error walking the file path. open testDir: permission denied"

	// check the specific error is returned
	if actualErrorMessage != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, actualErrorMessage)
	}
}

func TestCreateTarGzError(t *testing.T) {

	badSource := "./missing-folder"
	badTarget := "/-asdf.tar.gz"

	// cause an error by trying to create a file with an invalid name
	err := createTarGz(badSource, badTarget)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	actualErrorMessage := err.Error()
	expectedError := fmt.Sprintf("file creation failed, the name of the file is %v. open %v: read-only file system", badTarget, badTarget)

	// check the specific error is returned
	if actualErrorMessage != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, actualErrorMessage)
	}

	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a test .txt file
	testFilePath := filepath.Join(tmpDir, "test.txt")
	testFile, err := os.Create(testFilePath)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer testFile.Close()

	_, err = fmt.Fprintf(testFile, "Hello, world!")
	if err != nil {
		t.Fatalf("failed to write content to test file: %v", err)
	}

	expectedErr := errors.New("error walking the file path. lstat /nonexistent/directory: no such file or directory")
	err = createTarGz("/nonexistent/directory", "test.tar.gz")
	if err == nil || err.Error() != expectedErr.Error() {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}

	// Simulate an error during header writing
	testFile, _ = os.Create(testFilePath)
	testFile.Close() // Close the file to simulate a failure to open it for writing
	err = createTarGz(tmpDir, testFilePath)
	if err == nil || !strings.Contains(err.Error(), "error copying the file contents to the tar archive. archive/tar: write too long") {
		t.Errorf("Expected error containing 'error writing the header to the tar archive', got '%v'", err)
	}

}

func Test_ExecuteAnyCommand(t *testing.T) {

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"logs", "logs"})
	rootCmd.Execute()

	expected := "logs called"

	assert.Equal(t, actual.String(), expected, "actual is not expected")
}

func TestRunCleanup(t *testing.T) {
	// Specify the file path
	filesToDelete := []string{"../cmd/test.tar.gz", "../cmd/logs.tar.gz"}

	// Cleanup the testing files
	for _, file := range filesToDelete {
		err := os.Remove(file)
		if err != nil {
			fmt.Printf("error deleting file. %v", err)
		}
		fmt.Printf("file %v deleted.", file)
	}
}

/*

func TestGetLogs(t *testing.T) {
	testCases := []struct {
		name               string
		pods               []runtime.Object
		targetNamespace    string
		targetPod          string
		targetLabelKey     string
		expectedLabelValue string
		expectSuccess      bool
	}{
		{
			name: "existing_pod_found",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1-existing",
						Namespace: "namespace1",
						Labels: map[string]string{
							"label1": "value1",
						},
					},
				},
			},
			targetNamespace:    "namespace1",
			targetPod:          "pod1",
			targetLabelKey:     "label1",
			expectedLabelValue: "VALUE1",
			expectSuccess:      true,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			fakeClientset := fake.NewSimpleClientset(test.pods...)
			pods, err := getPods()
			logs, err := getLogs(
				test.targetNamespace,
				fakeClientset,
			)

			for _, pod := range pods {
				if pod.ObjectMeta.Name == "" && test.expectSuccess {
					t.Fatalf("no pod found: %v", err)
				}
			}
		})
	}
}
*/

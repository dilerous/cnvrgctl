package logs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			logDir := "./logs"

			pods, err := getPods(
				test.targetNamespace,
				fakeClientset,
			)

			for _, pod := range pods {
				if pod.ObjectMeta.Name == "" && test.expectSuccess {
					t.Fatalf("no pod found: %v", err)
				}
			}

			err = getLogs(pods, 10, logDir, fakeClientset)
			if err != nil {
				t.Fatalf("logs not gathered: %v", err)
			}

		})
	}
}

/*
func TestGetLogsError(t *testing.T) {

	fakeClientsetWithErr := fake.NewSimpleClientset()
	fakeClientsetWithErr.AddReactor("list", "pods", func(action reactor.Action) (handled bool, ret runtime.Object, err error) {

		// Inspect the action being performed
		fmt.Println("grabbing the output of AddReactor")
		fmt.Printf("Action: %s, Resource: %s\n", action.GetVerb(), action.GetResource().GroupResource())

		// Return an error to simulate a failed operation
		return true, nil, errors.New("fake list error")
	})

}
*/

func TestCreateTar(t *testing.T) {

	t.Run("when days is just 30", func(t *testing.T) {
		// Call the function to be tested
		err := createTar()
		if err != nil {
			t.Fatalf("Error creating tar: %v", err)
		}
		//Check if the tar file was created
		if _, err := os.Stat("logs.tar.gz"); os.IsNotExist(err) {
			t.Fatalf("Expected tar file logs.tar.gz does not exist")
		}
	})
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

/*
func Test_ExecuteAnyCommand(t *testing.T) {

	// Test calling connectToK8s()
	//client, err := connectToK8s()

	//TODO: added in fake

	// Testing the root command
	actual := new(bytes.Buffer)
	root.RootCmd.SetOut(actual)
	root.RootCmd.SetErr(actual)
	root.RootCmd.SetArgs([]string{""})
	root.RootCmd.Execute()

	// expected buffer length when calling the cnvrgctl command
	expected := 1030

	// grabing the actual output len as an int and comparing to expected.
	assert.Equal(t, actual.Len(), expected, "the expected length doesn't match")

	// testing the logs command
	logsActual := new(bytes.Buffer)
	root.RootCmd.SetOut(logsActual)
	root.RootCmd.SetErr(logsActual)
	root.RootCmd.SetArgs([]string{"logs"})
	root.RootCmd.Execute()

	fmt.Println(logsActual.Len())

	// expected buffer length when calling the cnvrgctl logs command
	expected = 0

	// grabing the actual output len as an int and comparing to expected length.
	assert.Equal(t, logsActual.Len(), expected, "the expected length doesn't match")

}
*/

func TestRunCleanup(t *testing.T) {
	// Specify the file path
	goPath := "/Users/bsoper/Documents/code/go_code/cnvrgctl/cmd/logs"
	filesToDelete := []string{goPath + "test.tar.gz", goPath + "logs.tar.gz"}

	// Cleanup the testing files
	for _, file := range filesToDelete {
		err := os.Remove(file)
		if err != nil {
			fmt.Printf("error deleting file, %v", err)
		}
		fmt.Printf("file %v deleted.", file)
	}
}

/*

	fakeKubeconfig := []byte(`{
        "apiVersion": "v1",
        "kind": "Config",
        "clusters": [
            {
                "name": "fake-cluster",
                "cluster": {
                    "server": "https://fake-cluster-server",
                    "insecure-skip-tls-verify": true
                }
            }
        ],
        "contexts": [
            {
                "name": "fake-context",
                "context": {
                    "cluster": "fake-cluster",
                    "user": "fake-user"
                }
            }
        ],
        "current-context": "fake-context",
        "users": [
            {
                "name": "fake-user",
                "user": {}
            }
        ]
    }`)

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

/*
	assert.Equal(t, pods[0].Namespace, testNamespace)
	assert.Equal(t, nil, actualError)
	assert.Equal(t, pods[0].Namespace, badNamespace)
*/

// Assert that the returned pods match the expected pods
/*
	if len(pods) != len(expectedPods) {
		t.Fatalf("Unexpected number of pods. Expected: %d, Got: %d", len(expectedPods), len(pods))
	}

	for i := range pods {
		if pods[i].Name != expectedPods[i].Name {
			t.Errorf("Unexpected pod name at index %d. Expected: %s, Got: %s", i, expectedPods[i].Name, pods[i].Name)
		}
		if pods[i].Namespace != expectedPods[i].Namespace {
			t.Errorf("Unexpected pod namespace at index %d. Expected: %s, Got: %s", i, expectedPods[i].Namespace, pods[i].Namespace)
		}
	}
*/

/*
// test for setting the log path
	// Temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		path           string
		expectedError  error
		expectedFolder bool
		pods           []runtime.Object
	}{
		{
			name:           "CreateLogsFolder_Success",
			path:           tempDir + "/logs",
			expectedError:  nil,
			expectedFolder: true,
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "test-namespace",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClientset := fake.NewSimpleClientset(test.pods...)

			var pods []corev1.Pod
			for _, obj := range test.pods {
				pod, ok := obj.(*corev1.Pod)
				if !ok {
					fmt.Println("Error: Object is not of type Pod")
					continue
				}
				pods = append(pods, *pod)
			}

			err := getLogs(pods, 1, fakeClientset)
			if err != nil {
				fmt.Printf("Here is the error %v.", err)
			}

			// Check if the folder exists
			//_, err = os.Stat(test.path)
			//if test.expectedFolder && err != nil {
			//	t.Errorf("Expected folder %s does not exist.", test.path)
			//}
		})
	}
*/

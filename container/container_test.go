package container

import (
	"testing"

	"fmt"

	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

func testClient() *docker.Client {
	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)
	if err != nil {
		panic(err)
	}

	return client
}

func testContainer(c *docker.Client) *docker.Container {
	config := docker.Config{Image: "golang:1.6", AttachStdout: true, AttachStderr: true}
	container, err := c.CreateContainer(docker.CreateContainerOptions{Name: "testContainer", Config: &config})
	if err != nil {
		panic(err)
	}

	return container
}

// Clean up the mess
func cleanup(c *docker.Client, id string) {
	fmt.Printf("Cleaning up %s\n", id)
	c.StopContainer(id, 10)

	err := c.RemoveContainer(docker.RemoveContainerOptions{ID: id})
	if err != nil {
		panic(err)
	}
}

func TestGetContainerByName(t *testing.T) {

	// Test fixture setup
	testClient := testClient()
	testContainer := testContainer(testClient)
	// Start the container
	testClient.StartContainer(testContainer.ID, &docker.HostConfig{})

	//XXX make sure to cleanup these on the way out :) thank go for such concise syntax
	defer cleanup(testClient, testContainer.ID)

	actualContainer, err := GetContainerByName(testClient, "testContainer")

	if err != nil {
		t.Error(err.Error())
		return
	}

	if actualContainer == nil {
		t.Error("getting nil container")
		return
	}

	if actualContainer.ID != testContainer.ID {
		t.Error("Getting wrong container")
		return
	}
}

func TestGetOutputFromStopppedCOntainer(t *testing.T) {

	// Test fixture setup
	testClient := testClient()

	config := docker.Config{
		Image: "golang:1.6", AttachStdout: true, AttachStderr: true, Cmd: []string{"ls"}}
	testContainer, err := testClient.CreateContainer(docker.CreateContainerOptions{Name: "testContainer", Config: &config})
	if err != nil {
		panic(err)
	}
	//XXX make sure to cleanup these on the way out :) thank go for such concise syntax
	defer cleanup(testClient, testContainer.ID)

	// Start the container
	testClient.StartContainer(testContainer.ID, &docker.HostConfig{})

	_, err = testClient.WaitContainer(testContainer.ID)
	if err != nil {
		t.Error(err)
		return
	}

	output, err := GetOutputFromStoppedContainer(testClient, testContainer.ID)
	if err != nil {
		t.Error(err)
		return
	}

	if !strings.Contains(output, "bin") {
		t.Error("Output should contain bin, instead ", output)
		return
	}
}

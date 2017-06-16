package container

import (
	"testing"

	"fmt"

	"strings"

	"time"

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

func testContainer(c *docker.Client, repository string, tag string) *docker.Container {
	c.PullImage(docker.PullImageOptions{Repository: repository, Tag: tag}, docker.AuthConfiguration{})
	config := docker.Config{Image: repository + ":" + tag, AttachStdout: true, AttachStderr: true}
	container, err := c.CreateContainer(docker.CreateContainerOptions{Name: "testContainer", Config: &config})
	if err != nil {
		panic(err)
	}

	return container
}

// Clean up the mess
func cleanup(c *docker.Client, id string) {
	fmt.Printf("Cleaning up %s\n", id)
	c.RemoveContainer(docker.RemoveContainerOptions{ID: id, Force: true})
}

func TestGetContainerByName(t *testing.T) {

	// Test fixture setup
	testClient := testClient()
	testContainer := testContainer(testClient, "golang", "1.6")
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

func TestMySQLAvailability(t *testing.T) {
	// Test fixture setup
	testClient := testClient()

	config := docker.Config{
		Image:        "mysql:5.6",
		AttachStdout: true,
		AttachStderr: true,
		Env:          []string{"MYSQL_ROOT_PASSWORD=password"},
		Healthcheck:  &docker.HealthConfig{Interval: 2000 * time.Millisecond, Retries: 10, Test: []string{"CMD-SHELL", "mysqladmin -ppassword ping --silent"}}}

	mysqlContainer, err := testClient.CreateContainer(docker.CreateContainerOptions{Name: "testMySQLContainer", Config: &config})
	if err != nil {
		panic(err)
	}

	// Start the MySQL container
	testClient.StartContainer(mysqlContainer.ID, &docker.HostConfig{})
	defer cleanup(testClient, mysqlContainer.ID)

	err = MySQLContainerAvailable(testClient, mysqlContainer.ID, 20000*time.Millisecond)
	if err != nil {
		t.Error(err)
	}
}

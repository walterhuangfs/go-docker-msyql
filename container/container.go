package container

import (
	"fmt"
	"log"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

type containerError struct {
	reason string
}

func (e *containerError) Error() string {
	return fmt.Sprintf("Container service failed with %s", e.reason)
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s\n", name, elapsed)
}

//GetContainerByName returns the container handle if it exists
// other wise error indicating it's not found
func GetContainerByName(client *docker.Client, name string) (*docker.APIContainers, error) {
	defer timeTrack(time.Now(), "GetContainerByName")

	// Instead of going to the more expensive inspect container
	// list container should show the right container
	containers, err := client.ListContainers(docker.ListContainersOptions{Filters: map[string][]string{"name": {name}}})

	if err != nil {
		return nil, err
	}

	if len(containers) != 1 {
		return nil, &containerError{fmt.Sprintf("number of container is not 1 instead %d", len(containers))}
	}

	// I don't know why I'm always doing this stupid thing
	return &containers[0], nil
}

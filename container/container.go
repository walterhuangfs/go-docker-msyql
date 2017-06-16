package container

import (
	"fmt"
	"log"
	"sync"
	"time"

	"bytes"

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

//GetOutputFromStoppedContainer  return the output of a stopped container as string
func GetOutputFromStoppedContainer(c *docker.Client, id string) (string, error) {
	var buf bytes.Buffer
	logsOptions := docker.LogsOptions{
		Container:    id,
		OutputStream: &buf,
		Stdout:       true,
		Stderr:       true,
		Follow:       true,
	}

	err := c.Logs(logsOptions)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// MySQLContainerAvailable block and wait for mysql container
// to be available
func MySQLContainerAvailable(c *docker.Client, id string, timeout time.Duration) error {
	listener := make(chan *docker.APIEvents)
	err := c.AddEventListener(listener)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func(listener chan *docker.APIEvents, wg *sync.WaitGroup) {
		for {
			select {
			case event := <-listener:
				//XXX Why health_status: healthy ?? not parsable
				if event.ID == id && event.Type == "container" && event.Action == "health_status: healthy" {
					fmt.Println("MySQL container started completed.")
					wg.Done()
				}
			}
		}
	}(listener, &wg)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return &containerError{"Time out waiting for database to be available"}
	}
}

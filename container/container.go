package container

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
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
	// Use pipe to read logs
	r, w := io.Pipe()
	go func() {
		c.Logs(docker.LogsOptions{
			Container:    id,
			OutputStream: w,
			ErrorStream:  w,
			Follow:       true,
			Stdout:       true,
			Stderr:       true,
			Tail:         "0",
		})
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	go func(reader io.Reader, wg *sync.WaitGroup) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			if strings.Contains(line, "starting as process 1") {
				log.Println("Database started!")
				wg.Done()
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "There was an error with the scanner in logging container", err)
		}
	}(r, &wg)

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

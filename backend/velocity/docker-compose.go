package velocity

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	yaml "gopkg.in/yaml.v2"
)

type DockerCompose struct {
	BaseStep
	ComposeFile string `json:"composeFile" yaml:"composeFile"`
	Contents    dockerComposeYaml
}

func NewDockerCompose(y string) *DockerCompose {
	step := DockerCompose{
		BaseStep: BaseStep{
			Type: "compose",
		},
	}
	err := yaml.Unmarshal([]byte(y), &step)
	if err != nil {
		panic(err)
	}

	dir, _ := os.Getwd()
	dockerComposeYml, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, step.ComposeFile))
	err = yaml.Unmarshal(dockerComposeYml, &step.Contents)
	if err != nil {
		panic(err)
	}

	services := make([]string, len(step.Contents.Services))
	i := 0
	for k := range step.Contents.Services {
		services[i] = k
		i++
	}
	step.OutputStreams = services

	return &step
}

func (dC DockerCompose) GetDetails() string {
	return fmt.Sprintf("composeFile: %s", dC.ComposeFile)
}

func (dC *DockerCompose) Validate(params map[string]Parameter) error {
	return nil
}

func (dC *DockerCompose) SetParams(params map[string]Parameter) error {
	return nil
}

func (dC *DockerCompose) Execute(emitter Emitter, params map[string]Parameter) error {
	serviceOrder := getServiceOrder(dC.Contents.Services, []string{})

	services := []*serviceRunner{}
	var wg sync.WaitGroup
	cli, _ := client.NewEnvClient()
	ctx := context.Background()

	networkResp, err := cli.NetworkCreate(ctx, fmt.Sprintf("vci-%s", dC.GetRunID()), types.NetworkCreate{
		Labels: map[string]string{"owner": "velocity-ci"},
	})
	if err != nil {
		log.Println(err)
	}
	log.Println(networkResp.ID)

	writers := map[string]StreamWriter{}
	// Create writers
	for _, serviceName := range serviceOrder {
		writers[serviceName] = emitter.NewStreamWriter(serviceName)
	}

	for _, serviceName := range serviceOrder {
		writer := writers[serviceName]
		writer.SetStatus(StateRunning)
		writer.Write([]byte(fmt.Sprintf("Configured %s", serviceName)))

		s := dC.Contents.Services[serviceName]

		// generate containerConfig + hostConfig
		containerConfig, hostConfig := generateContainerAndHostConfig(s)

		// Create service runners
		sR := newServiceRunner(
			cli,
			ctx,
			writer,
			&wg,
			params,
			fmt.Sprintf("%s-%s", dC.GetRunID(), serviceName),
			s.Image,
			&s.Build,
			containerConfig,
			hostConfig,
			networkResp.ID,
		)

		services = append(services, sR)
	}

	// Pull/Build images
	for _, serviceRunner := range services {
		serviceRunner.PullOrBuild()
	}

	// Create services
	for _, serviceRunner := range services {
		serviceRunner.Create()
	}
	stopServicesChannel := make(chan string, 32)
	// Start services
	for _, serviceRunner := range services {
		wg.Add(1)
		go serviceRunner.Run(stopServicesChannel)
	}

	_ = <-stopServicesChannel
	for _, s := range services {
		s.Stop()
	}
	wg.Wait()
	err = cli.NetworkRemove(ctx, networkResp.ID)
	if err != nil {
		log.Printf("network %s remove err: %s", networkResp.ID, err)
	}
	success := true
	for _, serviceRunner := range services {
		if serviceRunner.exitCode != 0 {
			success = false

			break
		}
	}

	if !success {
		for _, serviceName := range serviceOrder {
			writers[serviceName].SetStatus(StateFailed)
			writers[serviceName].Write([]byte(fmt.Sprintf("%s\n### FAILED \x1b[0m", errorANSI)))
		}
	} else {
		for _, serviceName := range serviceOrder {
			writers[serviceName].SetStatus(StateSuccess)
			writers[serviceName].Write([]byte(fmt.Sprintf("%s\n### SUCCESS \x1b[0m", successANSI)))
		}
	}

	return nil
}

func (dC *DockerCompose) String() string {
	j, _ := json.Marshal(dC)
	return string(j)
}

func generateContainerAndHostConfig(s dockerComposeService) (*container.Config, *container.HostConfig) {
	containerConfig := &container.Config{}
	if len(s.Command) > 0 {
		// containerConfig.Cmd = s.Command
	}
	return containerConfig, &container.HostConfig{}
}

func getServiceOrder(services map[string]dockerComposeService, serviceOrder []string) []string {
	for serviceName, serviceDef := range services {
		if isIn(serviceName, serviceOrder) {
			break
		}
		for _, linkedService := range serviceDef.Links {
			serviceOrder = getLinkedServiceOrder(linkedService, services, serviceOrder)
		}
		serviceOrder = append(serviceOrder, serviceName)
	}

	for len(services) != len(serviceOrder) {
		serviceOrder = getServiceOrder(services, serviceOrder)
	}

	return serviceOrder
}

func getLinkedServiceOrder(serviceName string, services map[string]dockerComposeService, serviceOrder []string) []string {
	if isIn(serviceName, serviceOrder) {
		return serviceOrder
	}
	for _, linkedService := range services[serviceName].Links {
		serviceOrder = getLinkedServiceOrder(linkedService, services, serviceOrder)
	}
	return append(serviceOrder, serviceName)
}

func isIn(needle string, haystack []string) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}

type dockerComposeYaml struct {
	Services map[string]dockerComposeService `json:"services" yaml:"services"`
}

type dockerComposeService struct {
	Image       string                    `json:"image" yaml:"image"`
	Build       dockerComposeServiceBuild `json:"build" yaml:"build"`
	WorkingDir  string                    `json:"workingDir" yaml:"working_dir"`
	Command     []string                  `json:"command" yaml:"command"`
	Links       []string                  `json:"links" yaml:"links"`
	Environment map[string]string         `json:"environment" yaml:"environment"`
	Volumes     []string                  `json:"volumes" yaml:"volumes"`
	Expose      []string                  `json:"expose" yaml:"expose"`
}

// func (a *dockerComposeService) UnmarshalYAML(unmarshal func(interface{}) error) error {
// 	var multi []string
// 	err := unmarshal(&multi)
// 	if err != nil {
// 		var single string
// 		err := unmarshal(&single)
// 		if err != nil {
// 			return err
// 		}
// 		*a = []string{single}
// 	} else {
// 		*a = multi
// 	}
// 	return nil
// }

type dockerComposeServiceBuild struct {
	Context    string `json:"context" yaml:"context"`
	Dockerfile string `json:"dockerfile" yaml:"dockerfile"`
}

package velocity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerRun struct {
	BaseStep       `yaml:",inline"`
	Image          string            `json:"image" yaml:"image"`
	Command        []string          `json:"command" yaml:"command"`
	Environment    map[string]string `json:"environment" yaml:"environment"`
	WorkingDir     string            `json:"workingDir" yaml:"workingDir"`
	MountPoint     string            `json:"mountPoint" yaml:"mountPoint"`
	IgnoreExitCode bool              `json:"ignoreExitCode" yaml:"ignoreExit"`
}

func (s *DockerRun) UnmarshalYamlInterface(y map[interface{}]interface{}) error {
	switch x := y["image"].(type) {
	case interface{}:
		s.Image = x.(string)
		break
	}

	s.Command = []string{}
	switch x := y["command"].(type) {
	case []interface{}:
		for _, p := range x {
			s.Command = append(s.Command, p.(string))
		}
		break
	case interface{}:
		re := regexp.MustCompile(`(".+")|('.+')|(\S+)`)
		matches := re.FindAllString(x.(string), -1)
		s.Command = []string{}
		for _, m := range matches {
			s.Command = append(s.Command, strings.TrimFunc(m, func(r rune) bool {
				return string(r) == `"` || string(r) == `'`
			}))
		}
		break
	}

	s.Environment = map[string]string{}
	switch x := y["environment"].(type) {
	case []interface{}:
		for _, e := range x {
			parts := strings.Split(e.(string), "=")
			key := parts[0]
			val := parts[1]
			s.Environment[key] = val
		}
		break
	case map[interface{}]interface{}:
		for k, v := range x {
			if num, ok := v.(int); ok {
				v = strconv.Itoa(num)
			}
			s.Environment[k.(string)] = v.(string)
		}
		break
	}

	switch x := y["workingDir"].(type) {
	case interface{}:
		s.WorkingDir = x.(string)
		break
	}
	switch x := y["mountPoint"].(type) {
	case interface{}:
		s.MountPoint = x.(string)
		break
	}
	switch x := y["ignoreExit"].(type) {
	case interface{}:
		s.IgnoreExitCode = x.(bool)
		break
	}
	return nil
}

func NewDockerRun() *DockerRun {
	return &DockerRun{
		Image:          "",
		Command:        []string{},
		Environment:    map[string]string{},
		WorkingDir:     "",
		MountPoint:     "",
		IgnoreExitCode: false,
		BaseStep: BaseStep{
			Type:          "run",
			OutputStreams: []string{"run"},
		},
	}
}

func (dR DockerRun) GetDetails() string {
	return fmt.Sprintf("image: %s command: %s", dR.Image, dR.Command)
}

func (dR *DockerRun) Execute(emitter Emitter, t *Task) error {
	writer := emitter.GetStreamWriter("run")
	writer.SetStatus(StateRunning)
	writer.Write([]byte(fmt.Sprintf("%s## %s\x1b[0m", infoANSI, dR.Description)))

	if dR.MountPoint == "" {
		dR.MountPoint = "/velocity_ci"
	}
	env := []string{}
	for k, v := range dR.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cwd, _ := os.Getwd()

	// Only used for Docker-based CLI. Unsupported right now.
	// if os.Getenv("SIB_CWD") != "" {
	// 	cwd = os.Getenv("SIB_CWD")
	// }

	config := &container.Config{
		Image: dR.Image,
		Cmd:   dR.Command,
		Volumes: map[string]struct{}{
			dR.MountPoint: {},
		},
		WorkingDir: fmt.Sprintf("%s/%s", dR.MountPoint, dR.WorkingDir),
		Env:        env,
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", cwd, dR.MountPoint),
		},
	}

	var wg sync.WaitGroup
	cli, _ := client.NewEnvClient()
	ctx := context.Background()

	networkResp, err := cli.NetworkCreate(ctx, fmt.Sprintf("vci-%s", dR.GetRunID()), types.NetworkCreate{
		Labels: map[string]string{"owner": "velocity-ci"},
	})
	if err != nil {
		GetLogger().Error("could not create docker network", zap.Error(err))
	}

	sR := newServiceRunner(
		cli,
		ctx,
		writer,
		&wg,
		t.ResolvedParameters,
		fmt.Sprintf("%s-%s", dR.GetRunID(), "run"),
		dR.Image,
		nil,
		config,
		hostConfig,
		nil,
		networkResp.ID,
	)

	sR.PullOrBuild(t.Docker.Registries)
	sR.Create()
	stopServicesChannel := make(chan string, 32)
	wg.Add(1)
	go sR.Run(stopServicesChannel)
	_ = <-stopServicesChannel
	sR.Stop()
	wg.Wait()
	err = cli.NetworkRemove(ctx, networkResp.ID)
	if err != nil {
		GetLogger().Error("could not remove docker network", zap.String("networkID", networkResp.ID), zap.Error(err))
	}

	exitCode := sR.exitCode

	if err != nil {
		return err
	}

	if exitCode != 0 && !dR.IgnoreExitCode {
		writer.SetStatus(StateFailed)
		writer.Write([]byte(fmt.Sprintf("%s### FAILED (exited: %d)\x1b[0m", errorANSI, exitCode)))
		return fmt.Errorf("Non-zero exit code: %d", exitCode)
	}

	writer.SetStatus(StateSuccess)
	writer.Write([]byte(fmt.Sprintf("%s### SUCCESS (exited: %d)\x1b[0m", successANSI, exitCode)))
	return nil

}

func (dR DockerRun) Validate(params map[string]Parameter) error {
	re := regexp.MustCompile("\\$\\{(.+)\\}")

	requiredParams := re.FindAllStringSubmatch(dR.Image, -1)
	if !isAllInParams(requiredParams, params) {
		return fmt.Errorf("Parameter %v missing", requiredParams)
	}
	requiredParams = re.FindAllStringSubmatch(dR.WorkingDir, -1)
	if !isAllInParams(requiredParams, params) {
		return fmt.Errorf("Parameter %v missing", requiredParams)
	}
	for _, c := range dR.Command {
		requiredParams = re.FindAllStringSubmatch(c, -1)
		if !isAllInParams(requiredParams, params) {
			return fmt.Errorf("Parameter %v missing", requiredParams)
		}
	}

	for key, val := range dR.Environment {
		requiredParams = re.FindAllStringSubmatch(key, -1)
		if !isAllInParams(requiredParams, params) {
			return fmt.Errorf("Parameter %v missing", requiredParams)
		}
		requiredParams = re.FindAllStringSubmatch(val, -1)
		if !isAllInParams(requiredParams, params) {
			return fmt.Errorf("Parameter %v missing", requiredParams)
		}
	}
	return nil
}

func (dR *DockerRun) SetParams(params map[string]Parameter) error {
	for paramName, param := range params {
		dR.Image = strings.Replace(dR.Image, fmt.Sprintf("${%s}", paramName), param.Value, -1)
		dR.WorkingDir = strings.Replace(dR.WorkingDir, fmt.Sprintf("${%s}", paramName), param.Value, -1)

		cmd := []string{}
		for _, c := range dR.Command {
			correctedCmd := strings.Replace(c, fmt.Sprintf("${%s}", paramName), param.Value, -1)
			cmd = append(cmd, correctedCmd)
		}
		dR.Command = cmd

		env := map[string]string{}
		for key, val := range dR.Environment {
			correctedKey := strings.Replace(key, fmt.Sprintf("${%s}", paramName), param.Value, -1)
			correctedVal := strings.Replace(val, fmt.Sprintf("${%s}", paramName), param.Value, -1)
			env[correctedKey] = correctedVal
		}

		dR.Environment = env
	}
	return nil
}

func (dR *DockerRun) String() string {
	j, _ := json.Marshal(dR)
	return string(j)
}

func isAllInParams(matches [][]string, params map[string]Parameter) bool {
	for _, match := range matches {
		found := false
		for paramName := range params {
			if paramName == match[1] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

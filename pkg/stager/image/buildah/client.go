package buildah

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	zlog "github.com/rs/zerolog/log"
)

var (
	TimeoutError = errors.New("Timeout")
)

type Client struct {
	DriverName string
	ExecPath   string
	Args       []string
	Timeout    time.Duration
	GcTimeout  time.Duration
}

func (b *Client) runCmd(args []string) ([]byte, error) {
	return b.runCmdWithTimeout(args, b.Timeout)
}

func (b *Client) runCmdWithTimeout(args []string, runTimeout time.Duration) ([]byte, error) {
	execPath := b.ExecPath
	actualArgs := append(b.Args, args...)

	cmd := exec.Command(execPath, actualArgs...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Start a timer
	timeout := make(<-chan time.Time)
	if runTimeout > 0 {
		timeout = time.After(b.Timeout)
	}

	select {
	case <-timeout:
		// Timeout happened first, kill the process and print a message.
		_ = cmd.Process.Kill()
		return buf.Bytes(), errors.Errorf("command timeout after %v", runTimeout)
	case err := <-done:
		output := buf.Bytes()
		if err != nil {
			zlog.Error().
				Str("exec", execPath).
				Strs("args", actualArgs).
				Err(err).
				Bytes("output", output).
				Msg("command failed")
			return output, errors.Wrapf(err, "command %s failed: %s", cmd.String(), output)
		}
		zlog.Debug().
			Str("exec", execPath).
			Strs("args", actualArgs).
			Bytes("output", output).
			Msg("command succeeded")
		return output, nil
	}
}

func (b *Client) IsContainerExist(containerName string) (bool, error) {
	args := []string{
		"containers",
		"--format", "{{.ContainerName}}",
		"--noheading",
		"--filter", fmt.Sprintf("name=%s", containerName),
	}

	output, err := b.runCmd(args)

	if err != nil {
		return false, err
	}

	if !regexp.MustCompile(fmt.Sprintf("^%s", containerName)).Match(output) {
		return false, nil
	}

	return true, nil
}

func (b *Client) From(containerName, image, dockerConfigJson string, tlsVerify bool) error {
	args := []string{"from", "--name", containerName, "--pull-always"}
	if !tlsVerify {
		args = append(args, "--tls-verify=false")
	}
	if dockerConfigJson != "" {
		authFilePath, cleanupFunc, err := b.CreateDockerAuth(containerName, dockerConfigJson)
		if err != nil {
			return errors.Wrapf(err, "can't create authfile=%s", authFilePath)
		}
		defer cleanupFunc()
		args = append(args, "--authfile", authFilePath)
	}
	args = append(args, image)

	_, err := b.runCmd(args)
	if err != nil {
		return err
	}
	return nil
}

func (b *Client) Mount(containerName string) (string, error) {
	args := []string{"mount", containerName}
	output, err := b.runCmd(args)
	glog.V(4).Infof("Client %s: %s", strings.Join(args, " "), string(output))
	if err != nil {
		return "", errors.Wrapf(err, "'Client %s' failed", strings.Join(args, " "))
	}
	provisionedRoot := strings.TrimSpace(string(output[:]))
	glog.V(4).Infof("container(name=%s)'s mount point at %s\n", containerName, provisionedRoot)
	return provisionedRoot, nil
}

func (b *Client) Commit(containerName, image string, squash bool) error {
	args := []string{"commit", "--format", "docker"}
	if squash {
		args = append(args, "--squash")
	}
	args = append(args, containerName, image)

	_, err := b.runCmd(args)
	if err != nil {
		return err
	}
	return nil
}

func (b *Client) Umount(containerName string) error {
	args := []string{"umount", containerName}
	_, err := b.runCmd(args)
	if err != nil {
		return err
	}
	return nil
}

func (b *Client) Push(containerName, image, dockerConfigJson string, tlsVerify bool) error {
	args := []string{"push"}
	if !tlsVerify {
		args = append(args, "--tls-verify=false")
	}
	if dockerConfigJson != "" {
		authFilePath, cleanupFunc, err := b.CreateDockerAuth(containerName, dockerConfigJson)
		if err != nil {
			return errors.Wrapf(err, "can't create authfile=%s", authFilePath)
		}
		defer cleanupFunc()
		args = append(args, "--authfile", authFilePath)
	}
	args = append(args, image)
	_, err := b.runCmd(args)
	if err != nil {
		return err
	}
	return nil
}

func (b *Client) Delete(containerName string) error {
	args := []string{"delete", containerName}
	_, err := b.runCmd(args)
	if err != nil {
		return err
	}
	return err
}

func (b *Client) CreateDockerAuth(containerName, dockerConfigJson string) (string, func(), error) {
	file, err := ioutil.TempFile("", fmt.Sprintf("%s-%s-", b.DriverName, containerName))
	cleanUpAuthFile := func() {
		if err := os.Remove(file.Name()); err != nil {
			zlog.Error().Err(err).Str("file", file.Name()).Msg("can't delete a file")
		}
	}

	if err != nil {
		cleanUpAuthFile()
		return "", nil, err
	}
	if err = file.Chmod(0700); err != nil {
		cleanUpAuthFile()
		return "", nil, err
	}
	if _, err = file.Write(([]byte)(dockerConfigJson)); err != nil {
		cleanUpAuthFile()
		return "", nil, err
	}
	return file.Name(), cleanUpAuthFile, nil
}

func (b *Client) GarbageCollectOnce() {
	zlog.Info().Dur("timeout", b.GcTimeout).Msg("collecting builadh garbage")
	out, err := b.runCmdWithTimeout([]string{"rmi", "-p"}, b.GcTimeout)
	if err != nil {
		zlog.Error().Err(err).Str("output", string(out)).Msg("failed collecting buildah garbage")
		return
	}
	zlog.Info().Dur("timeout", b.GcTimeout).Msg("done collecting buildah garbage")
}

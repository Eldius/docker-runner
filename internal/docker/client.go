package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var (
	ClientBuildErr        = errors.New("failed to create Docker client")
	ImageBuildErr         = errors.New("failed to build image")
	BuildDockerAPIErr     = errors.New("docker api build error")
	DockerfileNotFoundErr = errors.New("dockerfile not found")
	ContextDirReadErr     = errors.New("failed to read folder content to build request")
	ContextFilesReadErr   = errors.New("failed to add file to build request")
)

type Client struct {
	d *client.Client
}

// NewClient builds the Docker Client
func NewClient() (*Client, error) {
	apiClient, err := client.NewClientWithOpts(client.WithHostFromEnv(), client.WithAPIVersionNegotiation())
	if err != nil {
		err := fmt.Errorf("%w: %w", ClientBuildErr, err)
		return nil, err
	}

	fmt.Println("Client API version:", apiClient.ClientVersion())

	return &Client{
		d: apiClient,
	}, nil
}

func (c Client) Build(ctx context.Context, src string) error {
	fmt.Sprintln("Building image...")

	dockerFileReader, err := buildRequestReaderWithAllFiles(src)
	if err != nil {
		err = fmt.Errorf("%w: %w", ImageBuildErr, err)
		return err
	}

	//dockerFileReader, err := buildRequestReaderWithDockerfile(src)
	//if err != nil {
	//	err = fmt.Errorf("%w: %w", ImageBuildErr, err)
	//	return err
	//}

	response, err := c.d.ImageBuild(
		ctx,
		dockerFileReader,
		types.ImageBuildOptions{
			Tags: []string{"eldius/test-image"},
		},
	)
	if err != nil {
		err = fmt.Errorf("%w: %w", BuildDockerAPIErr, err)
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func buildRequestReaderWithAllFiles(src string) (io.Reader, error) {
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		err = fmt.Errorf("%w: %w", DockerfileNotFoundErr, err)
		return nil, err
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer func() {
		_ = tw.Close()
	}()

	dir, err := os.ReadDir(srcAbs)
	if err != nil {
		err = fmt.Errorf("%w: %w", ContextDirReadErr, err)
		return nil, err
	}

	for _, d := range dir {
		if !d.IsDir() {
			b, err := readFile(src, d.Name())
			if err != nil {
				err = fmt.Errorf("%w (opening %s):%w", ContextFilesReadErr, d.Name(), err)
				return nil, err
			}

			i, _ := d.Info()
			tarHeader := &tar.Header{
				Name: d.Name(),
				Size: int64(len(b)),
				Mode: int64(i.Mode()),
			}
			err = tw.WriteHeader(tarHeader)
			if err != nil {
				err = fmt.Errorf("%w (writing header %s):%w", ContextFilesReadErr, d.Name(), err)
				return nil, err
			}
			_, err = tw.Write(b)
			if err != nil {
				err = fmt.Errorf("%w (writing content %s):%w", ContextFilesReadErr, d.Name(), err)
				return nil, err
			}
		}
	}
	if err := tw.Flush(); err != nil {
		err = fmt.Errorf("%w (flushing header):%w", ContextFilesReadErr, err)
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func buildRequestReaderWithDockerfile(src string) (io.Reader, error) {
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		err = fmt.Errorf("%w: %w", DockerfileNotFoundErr, err)
		return nil, err
	}

	b, err := readFile(srcAbs, "Dockerfile")
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer func() {
		_ = tw.Close()
	}()

	tarHeader := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(b)),
	}
	err = tw.WriteHeader(tarHeader)
	if err != nil {
		err = fmt.Errorf("%w (writing %s):%w", ContextFilesReadErr, "Dockerfile", err)
		return nil, err
	}
	if err := tw.Close(); err != nil {
		err = fmt.Errorf("%w (closing header):%w", ContextFilesReadErr, err)
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func readFile(srcFolder, fileName string) ([]byte, error) {
	f, err := os.Open(filepath.Join(srcFolder, fileName))
	if err != nil {
		err = fmt.Errorf("%w (opening %s):%w", ContextFilesReadErr, fileName, err)
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	b, err := io.ReadAll(f)
	if err != nil {
		err = fmt.Errorf("%w (reading %s):%w", ContextFilesReadErr, fileName, err)
		return nil, err
	}

	slog.With("file_content", string(b), "file_name", fileName).Info("FileContent")
	return b, nil
}

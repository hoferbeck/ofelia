package core

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/client"
)

func BuildTestImage(client *client.Client, name string) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tw.WriteHeader(&tar.Header{Name: "Dockerfile"})
	tw.Write([]byte("FROM alpine\n"))
	tw.Close()

	resp, err := client.ImageBuild(context.Background(), &buf, build.ImageBuildOptions{
		Tags: []string{name},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(os.Stdout, resp.Body)
	return nil
}

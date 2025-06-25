## DMS glusterfs controller 

In order to test DMS as the glusterfs controller we need to deploy and run DMS in the same container as the gluster fs server.

We could do the following: 

build the binary and copy

```
docker cp dms gluster-container:/dms
```

then run the binary insdie the container:
```
docker exec gluster-container /path/in/container/dms
```

or with go:

```
func runGoBinaryInContainer() error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	ctx := context.Background()

	execConfig := container.ExecOptions{
		Cmd:          []string{"/path/in/container/myapp"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execIDResp, err := cli.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}

	if err := cli.ContainerExecStart(ctx, execIDResp.ID, container.ExecStartOptions{}); err != nil {
		return fmt.Errorf("failed to start exec command: %w", err)
	}

	return nil
}

```
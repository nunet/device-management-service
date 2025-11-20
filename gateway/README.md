
### Provider Interface

The `provider` package defines a **common contract** for all infrastructure backends.

```go
type Provider interface {
    Name() string
    ListPlans(ctx context.Context) ([]Plan, error)
    ProvisionServer(ctx context.Context, plan Plan, name, image, orchestratorDID string) (*Server, error)
    DeleteServer(ctx context.Context, serverID string) error
    RestartServer(ctx context.Context, serverID string) error
    GetServerStatus(ctx context.Context, serverID string) (*Server, error)
    SelectMatchingPlan(plans []Plan, target types.Resources) (*Plan, error)
}
```

### Local Incus Provider

`LocalIncusProvider` in the provider directory, is a concrete implementation of Provider that manages local virtual machines using Incus. This way we can locally run and test the on demand provisioning/promise bid logic


#### Run the logic

Prepare incus first, and download a vm image:

first run a vm to download an image:

```
incus launch images:ubuntu/24.04/cloud v2 --vm
```

then list

```
incus image list
```

then create an alias: (replace  2658fb357103 with what you see)
```
 incus image alias create ubuntu-22.04-vm 2658fb357103
```

run incus web ui and stop and delete any running vm:

```
incus webui
```


Run dms as a gateway by the following config:

```
general:
  compute_gateway: true
  providers:
    - name: local-incus
      type: local-incus
      config: {}
```

do a normal deployment


#### Run e2e tests

Just run `make e2e-DeployWithOnDemandProvisioner` and check the incus webui.

Once the online shell is available you can go and check the logs inside `/root`

You will be able to see the deployment being done
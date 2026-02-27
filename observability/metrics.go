package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// metrics

var (
	NodeOnboarded     metric.Int64UpDownCounter
	NodeOnboardedCPU  metric.Float64Gauge
	NodeOnboardedRAM  metric.Int64Gauge
	NodeOnboardedDisk metric.Int64Gauge
	NodeOnboardedGPU  metric.Int64Gauge
	NodeLocation      metric.Int64Gauge

	BidReceived metric.Int64Counter
	BidAccepted metric.Int64Counter

	DeploymentSuccess             metric.Int64Counter
	DeploySuccessAllocations      metric.Int64Gauge
	DeploySuccessCPUCoresAssigned metric.Float64Gauge
	DeploySuccessRAMGBAssigned    metric.Int64Gauge
	DeploySuccessDiskMBAssigned   metric.Float64Gauge
	DeploySuccessGPUCountAssigned metric.Int64Gauge
	DeploymentStatus              metric.Int64Counter

	AllocationHeartbeat metric.Int64Counter
	AllocationStatus    metric.Int64Gauge
	AllocCPUUsage       metric.Float64Gauge
	AllocMemUsed        metric.Int64Gauge
	AllocMemLimit       metric.Int64Gauge
	AllocNetRx          metric.Int64Gauge
	AllocNetTx          metric.Int64Gauge

	TxPaidAmount     metric.Float64Counter
	TxPaidFeesAmount metric.Float64Counter

	TxCreatedAmount    metric.Float64Counter
	TxCreatedUSDAmount metric.Float64Counter
)

func initMetrics(ctx context.Context) error {
	if !ObservabilityCfg.OTel.Enabled {
		log.Info("otel_metrics_disabled", "msg", "OTel metrics export disabled in config")
		return nil
	}
	if ObservabilityCfg.OTel.Endpoint == "" {
		log.Warn("otel_metrics_skipped", "msg", "OTel endpoint not configured")
		return nil
	}

	// Build exporter options from config
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(ObservabilityCfg.OTel.Endpoint),
	}
	if ObservabilityCfg.OTel.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return err
	}

	res, _ := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("dms"),
		),
	)

	// This sends metrics to the collector every 3 seconds
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(3*time.Second))),
	)

	otel.SetMeterProvider(mp)
	if err := systemMetrics(ctx); err != nil {
		return err
	}
	if err := nodeMetrics(ctx); err != nil {
		return err
	}
	if err := deploymentMetrics(ctx); err != nil {
		return err
	}
	if err := transactionMetrics(ctx); err != nil {
		return err
	}

	return nil
}

func systemMetrics(_ context.Context) error {
	meter := otel.Meter("system")

	// define types
	cpu, err := meter.Float64ObservableGauge(
		"dms.sys.cpu.total.norm",
		metric.WithDescription("CPU usage as a percentage (0.0 to 1.0)"),
		metric.WithUnit("%"),
		// did: string
	)
	if err != nil {
		return err
	}

	ramUsed, err := meter.Int64ObservableGauge(
		"dms.sys.memory.actual.used",
		metric.WithDescription("RAM usage in bytes"),
		metric.WithUnit("By"),
		// did: string
	)
	if err != nil {
		return err
	}

	ramTotal, err := meter.Int64ObservableGauge(
		"dms.sys.memory.total",
		metric.WithDescription("Total RAM in bytes"),
		metric.WithUnit("By"),
		// did: string
	)
	if err != nil {
		return err
	}

	diskUsed, err := meter.Int64ObservableGauge(
		"dms.sys.filesystem.used",
		metric.WithDescription("Disk usage in bytes"),
		metric.WithUnit("By"),
		// did: string
	)
	if err != nil {
		return err
	}

	diskTotal, err := meter.Int64ObservableGauge(
		"dms.sys.filesystem.total",
		metric.WithDescription("Total disk space in bytes"),
		metric.WithUnit("By"),
		// did: string
	)
	if err != nil {
		return err
	}

	uptime, err := meter.Float64ObservableGauge(
		"dms.sys.uptime",
		metric.WithDescription("System uptime in seconds"),
		metric.WithUnit("s"),
		// did: string
	)
	if err != nil {
		return err
	}

	load15, err := meter.Float64ObservableGauge(
		"dms.sys.load.15",
		metric.WithDescription("15-minute load average"),
		metric.WithUnit("%"),
		// did: string
	)
	if err != nil {
		return err
	}

	networkIn, err := meter.Int64ObservableGauge(
		"dms.sys.network.in",
		metric.WithDescription("Network bytes received"),
		metric.WithUnit("By"),
		// did: string
	)
	if err != nil {
		return err
	}

	networkOut, err := meter.Int64ObservableGauge(
		"dms.sys.network.out",
		metric.WithDescription("Network bytes sent"),
		metric.WithUnit("By"),
		// did: string
	)
	if err != nil {
		return err
	}

	attrs := metric.WithAttributes(AttrDID)

	// collect
	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		metrics := collectSystemMetrics()

		// CPU usage
		if cpuUsage, ok := metrics["cpuUsage"].(float64); ok {
			o.ObserveFloat64(cpu, cpuUsage/100.0, attrs)
		}

		// RAM usage
		if ramUsedVal, ok := metrics["ramUsed"].(uint64); ok {
			o.ObserveInt64(ramUsed, int64(ramUsedVal), attrs)
		}
		if ramTotalVal, ok := metrics["ramTotal"].(uint64); ok {
			o.ObserveInt64(ramTotal, int64(ramTotalVal), attrs)
		}

		// Disk usage
		if diskUsedVal, ok := metrics["diskUsed"].(uint64); ok {
			o.ObserveInt64(diskUsed, int64(diskUsedVal), attrs)
		}
		if diskTotalVal, ok := metrics["diskTotal"].(uint64); ok {
			o.ObserveInt64(diskTotal, int64(diskTotalVal), attrs)
		}

		// Uptime
		if uptimeVal, ok := metrics["uptime"].(float64); ok {
			o.ObserveFloat64(uptime, uptimeVal, attrs)
		}

		// Load average
		if load15Val, ok := metrics["load15"].(float64); ok {
			o.ObserveFloat64(load15, load15Val, attrs)
		}

		// Network RX/TX
		if rxBytes, ok := metrics["rxBytes"].(uint64); ok {
			o.ObserveInt64(networkIn, int64(rxBytes), attrs)
		}
		if txBytes, ok := metrics["txBytes"].(uint64); ok {
			o.ObserveInt64(networkOut, int64(txBytes), attrs)
		}

		return nil
	}, cpu, ramUsed, ramTotal, diskUsed, diskTotal, uptime, load15, networkIn, networkOut)

	return err
}

func nodeMetrics(_ context.Context) error {
	meter := otel.Meter("node")
	var err error

	// define types
	NodeOnboarded, err = meter.Int64UpDownCounter("dms.node.onboarded",
		metric.WithDescription("Total number of onboarded nodes"),
		metric.WithUnit("{node}"),
		// did: string
	)
	if err != nil {
		return err
	}
	NodeOnboardedCPU, err = meter.Float64Gauge("dms.node.onboarded.cpu",
		metric.WithDescription("CPU cores assigned by the node"),
		metric.WithUnit("{core}"),
		// did: string
	)
	if err != nil {
		return err
	}
	NodeOnboardedRAM, err = meter.Int64Gauge("dms.node.onboarded.memory",
		metric.WithDescription("RAM assigned by the node"),
		metric.WithUnit("GBy"),
		// did: string
	)
	if err != nil {
		return err
	}
	NodeOnboardedDisk, err = meter.Int64Gauge("dms.node.onboarded.disk",
		metric.WithDescription("Disk space assigned by the node"),
		metric.WithUnit("MBy"),
		// did: string
	)
	if err != nil {
		return err
	}
	NodeOnboardedGPU, err = meter.Int64Gauge("dms.node.onboarded.gpu",
		metric.WithDescription("Number of GPUs assigned by the node"),
		metric.WithUnit("{gpu}"),
		// did: string
	)
	if err != nil {
		return err
	}
	NodeLocation, err = meter.Int64Gauge("dms.node.location",
		metric.WithDescription("Node geolocation"),
		metric.WithUnit("{location}"),
		// did: string
		// attribute.String("continent", location.Continent),
		// attribute.String("country", location.Country),
		// attribute.String("city", location.City),
		// attribute.Bool("onboarded", n.onboarding.IsOnboarded()),
	)
	if err != nil {
		return err
	}

	return nil
}

func deploymentMetrics(_ context.Context) error {
	meter := otel.Meter("deployment")
	var err error

	// BIDS

	// Bid metrics: track bid requests received and accepted
	BidReceived, err = meter.Int64Counter("dms.bid.received",
		metric.WithDescription("Bid requests received from orchestrators"),
		metric.WithUnit("{bid}"),
		// did: string
		// attribute.String("orchestratorID", o.id),
	)
	if err != nil {
		return err
	}
	BidAccepted, err = meter.Int64Counter("dms.bid.accepted",
		metric.WithDescription("Bid responses sent (bids accepted by node)"),
		metric.WithUnit("{bid}"),
		// did: string
		// attribute.String("orchestratorID", o.id),
	)
	if err != nil {
		return err
	}

	// DEPLOYMENT

	// define types
	DeploymentStatus, err = meter.Int64Counter("dms.deployment.status",
		metric.WithDescription("Deployment status change"),
		metric.WithUnit("{deployment}"),
		// did: string
		// attribute.String("orchestratorID", o.id),
		// attribute.String("status", status.String()),
	)
	if err != nil {
		return err
	}
	DeploymentSuccess, err = meter.Int64Counter("dms.deployment.success",
		metric.WithDescription("Deployment successful"),
		metric.WithUnit("{deployment}"),
		// did: string
		// attribute.Int("allocations", len(o.Manifest().Allocations)),
	)
	if err != nil {
		return err
	}
	DeploySuccessAllocations, err = meter.Int64Gauge("dms.deployment.success.allocations",
		metric.WithDescription("Number of allocations in successful deployment"),
		metric.WithUnit("{allocation}"),
		// did: string
		// attribute.String("orchestratorID", o.id),
	)
	if err != nil {
		return err
	}
	DeploySuccessCPUCoresAssigned, err = meter.Float64Gauge("dms.deployment.success.cpu.assigned",
		metric.WithDescription("CPU cores assigned in successful deployment"),
		metric.WithUnit("{core}"),
		// did: string
		// attribute.String("orchestratorID", o.id),
	)
	if err != nil {
		return err
	}
	DeploySuccessRAMGBAssigned, err = meter.Int64Gauge("dms.deployment.success.memory.assigned",
		metric.WithDescription("RAM gigabytes assigned in successful deployment"),
		metric.WithUnit("GBy"),
		// did: string
		// attribute.String("orchestratorID", o.id),
	)
	if err != nil {
		return err
	}
	DeploySuccessDiskMBAssigned, err = meter.Float64Gauge("dms.deployment.success.disk.assigned",
		metric.WithDescription("Disk space assigned in successful deployment"),
		metric.WithUnit("MBy"),
		// did: string
		// attribute.String("orchestratorID", o.id),
	)
	if err != nil {
		return err
	}
	DeploySuccessGPUCountAssigned, err = meter.Int64Gauge("dms.deployment.success.gpu.assigned",
		metric.WithDescription("Number of GPUs assigned in successful deployment"),
		metric.WithUnit("{gpu}"),
		// did: string
		// attribute.String("orchestratorID", o.id),
	)
	if err != nil {
		return err
	}

	// ALLOCATIONS

	AllocationHeartbeat, err = meter.Int64Counter("dms.allocation.heartbeat",
		metric.WithDescription("Periodic deployment heartbeat"),
		metric.WithUnit("{deployment}"),
		// did: string
		// attribute.String("orchestratorID", o.id),
		// attribute.String("allocationID", notification.AllocationID),
		// attribute.String("status", notification.Status),
	)
	if err != nil {
		return err
	}
	AllocationStatus, err = meter.Int64Gauge("dms.allocation.status",
		metric.WithDescription("Allocation status change"),
		metric.WithUnit("{allocation}"),
		// did: string
		// attribute.String("orchestratorID", o.id),
		// attribute.String("allocationID", notification.AllocationID),
		// attribute.String("status", notification.Status),
	)
	if err != nil {
		return err
	}
	AllocCPUUsage, err = meter.Float64Gauge("dms.allocation.cpu.usage",
		metric.WithDescription("Allocation CPU usage percent"),
		metric.WithUnit("%"),
		// did: string
		// attribute.String("orchestratorID", o.id),
		// attribute.String("allocationID", notification.AllocationID),
	)
	if err != nil {
		return err
	}
	AllocMemUsed, err = meter.Int64Gauge("dms.allocation.memory.used",
		metric.WithDescription("Allocation memory used bytes"),
		metric.WithUnit("By"),
		// did: string
		// attribute.String("orchestratorID", o.id),
		// attribute.String("allocationID", notification.AllocationID),
	)
	if err != nil {
		return err
	}
	AllocMemLimit, err = meter.Int64Gauge("dms.allocation.memory.limit",
		metric.WithDescription("Allocation memory limit bytes"),
		metric.WithUnit("By"),
		// did: string
		// attribute.String("orchestratorID", o.id),
		// attribute.String("allocationID", notification.AllocationID),
	)
	if err != nil {
		return err
	}
	AllocNetRx, err = meter.Int64Gauge("dms.allocation.network.rx",
		metric.WithDescription("Allocation network bytes received"),
		metric.WithUnit("By"),
		// did: string
		// attribute.String("orchestratorID", o.id),
		// attribute.String("allocationID", notification.AllocationID),
	)
	if err != nil {
		return err
	}
	AllocNetTx, err = meter.Int64Gauge("dms.allocation.network.tx",
		metric.WithDescription("Allocation network bytes sent"),
		metric.WithUnit("By"),
		// did: string
		// attribute.String("orchestratorID", o.id),
		// attribute.String("allocationID", notification.AllocationID),
	)
	if err != nil {
		return err
	}

	return nil
}

func transactionMetrics(_ context.Context) error {
	meter := otel.Meter("transaction")
	var err error

	// define types
	TxPaidAmount, err = meter.Float64Counter("dms.transaction.paid.amount",
		metric.WithDescription("Total amount of paid transactions"),
		metric.WithUnit("{NTX}"),
		// did: string
		// attribute.String("ContractDID", tx.ContractDID),
	)
	if err != nil {
		return err
	}
	TxPaidFeesAmount, err = meter.Float64Counter("dms.transaction.paid.fees.amount",
		metric.WithDescription("Total amount of paid fees"),
		metric.WithUnit("{NTX}"),
		// did: string
		// attribute.String("ContractDID", tx.ContractDID),
	)
	if err != nil {
		return err
	}
	TxCreatedAmount, err = meter.Float64Counter("dms.transaction.created.amount",
		metric.WithDescription("Total amount of created transactions"),
		metric.WithUnit("{NTX}"),
		// did: string
		// attribute.String("ContractDID", req.ContractDID),
	)
	if err != nil {
		return err
	}

	TxCreatedUSDAmount, err = meter.Float64Counter("dms.transaction.created.usd.amount",
		metric.WithDescription("Total amount of created transactions in USD"),
		metric.WithUnit("USD"),
		// did: string
		// attribute.String("ContractDID", req.ContractDID),
	)
	if err != nil {
		return err
	}

	return nil
}

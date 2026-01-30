package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esapi"
	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
)

var ErrNotFound = fmt.Errorf("logs not found")

const dmsStartMsg = "docker_client_init_started"

// keep in sync with /observability/labels.go
// TODO add CI indexes
var indexes = []string{
	"nunet-dms", "accounting-index", "metric-index", "allocation-index", "deployment-index",
	"node-index",
}

const (
	EnvESAPIKey = "DMS_ES_API_KEY"
	EnvESURL    = "DMS_ES_URL"
)

var args Args

type Args struct {
	DID        *DIDCmd        `arg:"subcommand:did" help:"Import a single DMS run via a DID"`
	Errors     *ErrorsCmd     `arg:"subcommand:errors" help:"List recent DIDs with errors"`
	Deployment *DeploymentCmd `arg:"subcommand:deployment" help:"Import all logs related to a deployment"`
}

func (Args) Description() string {
	n := os.Args[0]
	if strings.Contains(n, "go-build") {
		n = "./maint-scripts/e2e/ingest.sh"
	}
	return shared.Sprintf(`
		Import logs from ElasticSearch per DID, deployment ID, or list recent errors from remote logs.
	
		Requires %[2]s to be set, eg:
		$> export %[2]s=supersecretapikey
		
		Examples:
		
		Import all logs for deployment 7e529d8eca095cb3b21ad9bdc2adcaec58fce0b44666723ea3ee1733b01e322a
		$> %[1]s deployment 7e529d8eca095cb3b21ad9bdc2adcaec58fce0b44666723ea3ee1733b01e322a
		
		List recent runs for did:key:z6Mkr2gsLuCNBCVopzGQhM8uBPyrLGsjUB33SbPYAFhKZ9Ar
		$> %[1]s did did:key:z6Mkr2gsLuCNBCVopzGQhM8uBPyrLGsjUB33SbPYAFhKZ9Ar
		
		Download the latest run of did:key:z6Mkr2gsLuCNBCVopzGQhM8uBPyrLGsjUB33SbPYAFhKZ9Ar
		$> %[1]s did --download 1 \
			did:key:z6Mkr2gsLuCNBCVopzGQhM8uBPyrLGsjUB33SbPYAFhKZ9Ar
		
		Download logs 5m around 2025-09-24T06:30:25.517Z of did:key:z6Mkr2gsLuCNBCVopzGQhM8uBPyrLGsjUB33SbPYAFhKZ9Ar
		$> %[1]s did --timestamp 2025-09-24T06:30:25.517Z \
			did:key:z6Mkr2gsLuCNBCVopzGQhM8uBPyrLGsjUB33SbPYAFhKZ9Ar
		
		Show all errors from the last 2h
		$> %[1]s errors --duration 2h
		
		Show all errors from the last 24h, with stack traces, and save HTML output.
		$> %[1]s errors --stack-traces --output-html errors.html
		
	`, n, EnvESAPIKey)
}

type DIDCmd struct {
	DID              string        `arg:"positional,required" help:"User's DID"`
	MaxLines         int           `arg:"-m,--max-lines" help:"Max lines to import" default:"10000"`
	Output           string        `arg:"-o,--output" help:"Output file (default - DID.jsonl)"`
	Download         int           `arg:"-d,--download" help:"DMS run number to import (1 - latest)" default:"0"`
	Timestamp        string        `arg:"-t,--timestamp" help:"Download logs from a specific timestamp"`
	AdjacentDuration time.Duration `arg:"-a,--adjacent-duration" help:"Time to download around --timestamp" default:"5m"`
	// TODO ErrorsOnly
}

type ErrorsCmd struct {
	Duration    time.Duration `arg:"-d,--duration" help:"Max error age" default:"24h"`
	Limit       int           `arg:"-l,--limit" help:"Max errors to show" default:"100"`
	StackTraces bool          `arg:"-s,--stack-traces" help:"Show stack traces" default:"false"`
	OutputHTML  string        `arg:"-o,--output-html" help:"Render HTML to a file"`
	// TODO GroupBy DID,error
}

type DeploymentCmd struct {
	DeploymentID     string        `arg:"positional,required" help:"Deployment ID aka Orchestrator ID"`
	MaxLines         int           `arg:"-m,--max-lines" help:"Max lines to import (per log file)" default:"10000"`
	AdjacentDuration time.Duration `arg:"-a,--adjacent-duration" help:"Time to download around a bid_request" default:"5m"`
}

type ResponseRaw struct {
	Hits struct {
		Hits []struct {
			Source any `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type ResponseParsed struct {
	Hits struct {
		Hits []struct {
			Source struct {
				Timestamp string `json:"timestamp"`
				DID       string `json:"did"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func main() {
	// targetDID := "did:key:z6Mkr2gsLuCNBCVopzGQhM8uBPyrLGsjUB33SbPYAFhKZ9Ar"
	// apiKey := "Vk02VG9Kb0JfaUhBRTgzcWJoZTM6MmgzZGFkZGtUNmUwTy1qOV9XeVpNdw=="
	ctx := context.Background()

	// parse and validate
	p := arg.MustParse(&args)
	if p.Subcommand() == nil {
		p.WriteHelp(os.Stdout)
		os.Exit(0)
	}
	apiKey := os.Getenv(EnvESAPIKey)
	if apiKey == "" {
		p.Fail(EnvESAPIKey + " must be set")
	}

	// ES URL
	url := "https://telemetry.nunet.io"
	var transport http.RoundTripper
	if v := os.Getenv(EnvESURL); v != "" {
		url = v
		// custom URL, disable cert checks
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// init ES client
	cfg := elasticsearch.Config{
		Addresses: []string{url},
		APIKey:    apiKey,
		Transport: transport,
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		p.Fail("connecting to ElasticSearch: " + err.Error())
	}

	switch {
	case args.DID != nil:
		if args.DID.Download > 0 {
			if err := DownloadRun(ctx, es, args); err != nil {
				_ = p.FailSubcommand(err.Error(), "did")
			}
		} else if args.DID.Timestamp != "" {
			if err := DownloadTimestamp(ctx, es, args); err != nil {
				_ = p.FailSubcommand(err.Error(), "did")
			}
		} else if err := ListRuns(ctx, es, args); err != nil {
			_ = p.FailSubcommand(err.Error(), "did")
		}
	case args.Errors != nil:
		if err := ListErrors(ctx, es, args); err != nil {
			_ = p.FailSubcommand(err.Error(), "errors")
		}
	case args.Deployment != nil:
		if err := DownloadDeployment(ctx, es, args); err != nil {
			_ = p.FailSubcommand(err.Error(), "deployment")
		}
	}
}

func ListRuns(ctx context.Context, es *elasticsearch.Client, args Args) error {
	did := args.DID.DID
	runs, err := ESListRuns(ctx, es, did, 100)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d runs for %s:\n", len(runs), did)
	for i, run := range runs {
		fmt.Printf("%d: %s\n", i+1, run)
	}

	fmt.Println()
	fmt.Print(shared.Sprintf(`
		To download the latest run:
		$> nunet-ingest did --download 1 %s
	`, did) + "\n")

	return nil
}

func DownloadRun(ctx context.Context, es *elasticsearch.Client, args Args) error {
	did := args.DID.DID
	runs, err := ESListRuns(ctx, es, did, 100)
	if err != nil {
		return err
	}
	oldestTime := runs[0]
	newestTime := time.Now().Format(time.RFC3339)
	runIdx := args.DID.Download - 1
	if runIdx > 0 && len(runs) > runIdx {
		oldestTime = runs[runIdx]
		newestTime = runs[runIdx-1]
	}

	// find all logs after docker_client_init_started
	query := map[string]any{
		"size": args.DID.MaxLines,
		"sort": []map[string]any{
			{"timestamp": "asc"},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{"match": map[string]any{"did": did}},
				},
				"filter": []map[string]any{
					{
						"range": map[string]any{
							"timestamp": map[string]any{
								"lt":  newestTime,
								"gte": oldestTime,
							},
						},
					},
				},
			},
		},
	}
	filename := did + "-" + oldestTime
	if args.DID.Output != "" {
		filename = args.DID.Output
	}

	return download(ctx, es, query, filename)
}

func DownloadTimestamp(ctx context.Context, es *elasticsearch.Client, args Args) error {
	// validate
	did := args.DID.DID
	if did == "" {
		return fmt.Errorf("DID is required")
	}
	if args.DID.Timestamp == "" {
		return fmt.Errorf("timestamp is required")
	}

	// parse
	timestamp, err := shared.ParseTimestamp(args.DID.Timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %s", args.DID.Timestamp)
	}
	newestTime := timestamp.Add(args.DID.AdjacentDuration).Format(time.RFC3339)
	oldestTime := timestamp.Add(-args.DID.AdjacentDuration).Format(time.RFC3339)

	// find all logs after docker_client_init_started
	query := map[string]any{
		"size": args.DID.MaxLines,
		"sort": []map[string]any{
			{"timestamp": "asc"},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{"match": map[string]any{"did": did}},
				},
				"filter": []map[string]any{
					{
						"range": map[string]any{
							"timestamp": map[string]any{
								"lt":  newestTime,
								"gte": oldestTime,
							},
						},
					},
				},
			},
		},
	}
	filename := did + "-" + args.DID.AdjacentDuration.String() + "-" + args.DID.Timestamp
	if args.DID.Output != "" {
		filename = args.DID.Output
	}

	fmt.Printf(`Querying 'did:"%s" timestamp gte %s timestamp lt %s...\n`, did, oldestTime, newestTime)
	return download(ctx, es, query, filename)
}

func ListErrors(ctx context.Context, es *elasticsearch.Client, args Args) error {
	// find the latest docker_client_init_started
	errQuery := map[string]any{
		"size": args.Errors.Limit,
		"sort": []map[string]any{
			{"timestamp": "desc"},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{"match": map[string]any{"level": "ERROR"}},
				},
				"filter": []map[string]any{
					{
						"range": map[string]any{
							"timestamp": map[string]any{
								"gte": time.Now().Add(-args.Errors.Duration).Format(time.RFC3339),
							},
						},
					},
				},
			},
		},
	}
	res, err := ESSearch(ctx, es, errQuery)
	if err != nil {
		return fmt.Errorf("searching: %s", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("error: %s", res.String())
	}

	// parse log entries
	var r ResponseRaw
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	lines := make([]string, len(r.Hits.Hits))
	for i, rec := range r.Hits.Hits {
		// TODO optimize: avoid remarshalling
		b, err := json.Marshal(rec.Source)
		if err != nil {
			return fmt.Errorf("marshaling log: %w", err)
		}
		lines[i] = string(b)
	}

	processed := shared.ParseLines(lines)

	// render with specified fields
	renderArgs := shared.ArgsBasic{
		ExtraField: []string{"did", "behavior", "msg_from.did"},
	}
	if !args.Errors.StackTraces {
		renderArgs.SkipField = []string{"stack_trace"}
	}
	name := "errors-" + args.Errors.Duration.String() + ".html"
	output := ""
	if output, err = shared.RenderSlice(name, processed, renderArgs); err != nil {
		return fmt.Errorf("rendering tmp file: %w", err)
	}

	// save HTML
	if wd, err := os.Getwd(); err == nil && args.Errors.OutputHTML != "" {
		err := shared.SaveHTML(filepath.Join(wd, args.Errors.OutputHTML), output, true)
		if err != nil {
			return err
		}
		fmt.Printf("Created %s\n", args.Errors.OutputHTML)
	}

	return nil
}

func DownloadDeployment(ctx context.Context, es *elasticsearch.Client, args Args) error {
	orchestratorID := args.Deployment.DeploymentID
	var errDown error
	downloaded := make(map[string]bool)

	// query `from.inbox:"ORCHESTRATOR_ID" and msg:"bid_request"`
	query := map[string]any{
		"sort": []map[string]any{
			{"timestamp": "asc"},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{"match": map[string]any{"msg": "bid_request"}},
					{"match": map[string]any{"from.inbox": orchestratorID}},
				},
			},
		},
	}
	fmt.Printf(`Querying: 'from.inbox:"%s" and msg:"bid_request"'...\n`, orchestratorID)
	res, err := ESSearch(ctx, es, query)
	if err != nil {
		return fmt.Errorf("searching: %s", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("response: %s", res.String())
	}

	// parse and download matches
	var r ResponseParsed
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(r.Hits.Hits) == 0 {
		return fmt.Errorf(`no results for 'from.inbox:"%s" and msg:"bid_request"'`, orchestratorID)
	}
	for _, rec := range r.Hits.Hits {
		if downloaded[rec.Source.DID] {
			continue
		}
		err := DownloadTimestamp(ctx, es, Args{
			DID: &DIDCmd{
				DID:              rec.Source.DID,
				Timestamp:        rec.Source.Timestamp,
				AdjacentDuration: args.Deployment.AdjacentDuration,
				MaxLines:         args.Deployment.MaxLines,
			},
		})
		errDown = errors.Join(errDown, err)
		downloaded[rec.Source.DID] = true
	}

	// query 'ensembleID:"ORCHESTRATOR_ID"'
	query = map[string]any{
		"sort": []map[string]any{
			{"timestamp": "asc"},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{"match": map[string]any{"ensembleID": orchestratorID}},
				},
			},
		},
	}
	fmt.Printf(`Querying: 'ensembleID:"%s"'...\n`, orchestratorID)
	res, err = ESSearch(ctx, es, query)
	if err != nil {
		return fmt.Errorf("searching: %s", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("response: %s", res.String())
	}

	// parse and download matches
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(r.Hits.Hits) == 0 {
		return fmt.Errorf(`no results for 'ensembleID:"%s"'`, orchestratorID)
	}
	for _, rec := range r.Hits.Hits {
		if downloaded[rec.Source.DID] {
			continue
		}
		err := DownloadTimestamp(ctx, es, Args{
			DID: &DIDCmd{
				DID:              rec.Source.DID,
				Timestamp:        rec.Source.Timestamp,
				AdjacentDuration: args.Deployment.AdjacentDuration,
				MaxLines:         args.Deployment.MaxLines,
			},
		})
		errDown = errors.Join(errDown, err)
		downloaded[rec.Source.DID] = true
	}

	return errDown
}

// ELASTIC SEARCH

// ESListRuns returns the timestamps of all runs for a given DID.
func ESListRuns(ctx context.Context, es *elasticsearch.Client, did string, limit int) ([]string, error) {
	// find the latest docker_client_init_started
	anchorQuery := map[string]any{
		"size": limit,
		"sort": []map[string]any{
			{"timestamp": "desc"},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{"match": map[string]any{"did": did}},
					{"match": map[string]any{"msg": dmsStartMsg}}, // Use .keyword if exact match needed
				},
			},
		},
	}
	res, err := ESSearch(ctx, es, anchorQuery)
	if err != nil {
		return nil, fmt.Errorf("searching: %s", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("error: %s", res.String())
	}

	// extract timestamp of docker_client_init_started
	var anchorRes map[string]any
	if err := json.NewDecoder(res.Body).Decode(&anchorRes); err != nil {
		return nil, err
	}
	hits := anchorRes["hits"].(map[string]any)["hits"].([]any)
	if len(hits) == 0 {
		return nil, fmt.Errorf("%w: no logs found for %s", ErrNotFound, did)
	}

	ret := make([]string, len(hits))
	for i, hit := range hits {
		ret[i] = hit.(map[string]any)["_source"].(map[string]any)["timestamp"].(string)
	}

	return ret, nil
}

func download(ctx context.Context, es *elasticsearch.Client, query map[string]any, filename string) error {
	res, err := ESSearch(ctx, es, query)
	if err != nil {
		return fmt.Errorf("searching: %s", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("response: %s", res.String())
	}

	// parse log entries
	var r ResponseRaw
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	lines := make([]string, len(r.Hits.Hits))
	for i, rec := range r.Hits.Hits {
		// TODO optimize: avoid remarshalling
		b, err := json.Marshal(rec.Source)
		if err != nil {
			return fmt.Errorf("marshaling log: %w", err)
		}
		lines[i] = string(b)
	}

	// write to file
	if !strings.HasSuffix(filename, ".jsonl") {
		filename += ".jsonl"
	}
	if os.WriteFile(filename, []byte(strings.Join(lines, "\n")), 0o644) != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	fmt.Printf("Wrote %d log lines to %s\n", len(lines), filename)

	return nil
}

func ESSearch(ctx context.Context, es *elasticsearch.Client, q any) (*esapi.Response, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(q); err != nil {
		log.Fatalf("Error encoding query: %s", err)
	}

	return es.Search(
		es.Search.WithContext(ctx),
		es.Search.WithIndex(indexes...),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithPretty(),
	)
}

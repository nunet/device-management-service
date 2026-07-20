// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	dmsUtils "gitlab.com/nunet/device-management-service/utils"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

type artifactKind string

const (
	artifactLogs    artifactKind = "logs"
	artifactVolumes artifactKind = "volumes"
)

type artifactItem struct {
	Kind    artifactKind `json:"kind"`
	Name    string       `json:"name"`
	Path    string       `json:"path"`
	Bytes   int64        `json:"bytes"`
	ModTime time.Time    `json:"mod_time"`
}

type artifactListOpts struct {
	kinds   []artifactKind
	sortBy  string
	reverse bool
	match   string
	glob    string
	older   time.Duration
	limit   int
	jsonOut bool
	raw     bool
	short   bool
}

type artifactPruneOpts struct {
	kinds   []artifactKind
	sortBy  string
	reverse bool
	match   string
	glob    string
	older   time.Duration
}

func newArtifactsCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifacts",
		Short: "Manage deployment artifacts in work_dir",
		Long: `Manage deployment artifacts under general.work_dir:

  <work_dir>/jobs/      job allocation logs and results  (artifacts list|prune logs)
  <work_dir>/volumes/   local volume data                (artifacts list|prune volumes)

DMS process logs are not managed by this command.

Running "nunet artifacts" with no subcommand is equivalent to "nunet artifacts list --short".`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runArtifactsList(cmd, dmsCli, artifactListOpts{
				kinds: []artifactKind{artifactLogs, artifactVolumes},
				short: true,
			})
		},
	}

	cmd.AddCommand(newArtifactsListCmd(dmsCli))
	cmd.AddCommand(newArtifactsPruneCmd(dmsCli))
	return cmd
}

func newArtifactsListCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	opts := &artifactListOpts{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List deployment artifacts with sizes",
		Long:  `List job logs and/or volumes under general.work_dir. Defaults to both.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.kinds = []artifactKind{artifactLogs, artifactVolumes}
			return runArtifactsList(cmd, dmsCli, *opts)
		},
	}

	bindArtifactsListFlags(cmd, opts)

	cmd.AddCommand(newArtifactsListKindCmd(dmsCli, opts, artifactLogs, "List job allocation logs under work_dir/jobs"))
	cmd.AddCommand(newArtifactsListKindCmd(dmsCli, opts, artifactVolumes, "List local volumes under work_dir/volumes"))
	return cmd
}

func newArtifactsListKindCmd(dmsCli *cli.DmsCLI, opts *artifactListOpts, kind artifactKind, short string) *cobra.Command {
	return &cobra.Command{
		Use:   string(kind),
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o := *opts
			o.kinds = []artifactKind{kind}
			return runArtifactsList(cmd, dmsCli, o)
		},
	}
}

func bindArtifactsListFlags(cmd *cobra.Command, opts *artifactListOpts) {
	cmd.PersistentFlags().StringVar(&opts.sortBy, "sort", "size", `sort by: "size", "mtime", "name"`)
	cmd.PersistentFlags().BoolVar(&opts.reverse, "reverse", false, "reverse sort order")
	cmd.PersistentFlags().StringVar(&opts.match, "match", "", "substring match on name/path (case-insensitive)")
	cmd.PersistentFlags().StringVar(&opts.glob, "glob", "", `glob match on base name (e.g. "*stdout*")`)
	cmd.PersistentFlags().DurationVar(&opts.older, "older-than", 0, "only include items older than duration (e.g. 168h, 720h)")
	cmd.PersistentFlags().IntVar(&opts.limit, "limit", 0, "return at most N items after sorting (0 = no limit)")
	cmd.PersistentFlags().BoolVar(&opts.jsonOut, "json", false, "output JSON")
	cmd.PersistentFlags().BoolVar(&opts.raw, "bytes", false, "print raw byte counts")
	cmd.PersistentFlags().BoolVarP(&opts.short, "short", "s", false, "show a short summary instead of per-item listing")
}

func runArtifactsList(cmd *cobra.Command, dmsCli *cli.DmsCLI, opts artifactListOpts) error {
	cfg, err := dmsCli.Config()
	if err != nil {
		return err
	}

	items := collectArtifacts(dmsCli.FS(), cfg.General.WorkDir, opts.kinds)

	items = filterArtifacts(items, opts.match, opts.glob, opts.older)
	sortArtifacts(items, opts.sortBy, opts.reverse)
	if opts.limit > 0 && len(items) > opts.limit {
		items = items[:opts.limit]
	}

	if opts.short {
		return printArtifactsShort(cmd, cfg.General.WorkDir, items, opts.jsonOut, opts.raw)
	}

	if opts.jsonOut {
		enc, _ := json.MarshalIndent(items, "", "  ")
		cmd.Println(string(enc))
		return nil
	}

	var total int64
	for _, it := range items {
		total += it.Bytes
		cmd.Printf("%s\t%8s\t%s\t%s\n", it.Kind, formatBytes(it.Bytes, opts.raw), it.ModTime.UTC().Format(time.RFC3339), it.Path)
	}
	cmd.Printf("\ntotal\t%s\t\t%d items\n", formatBytes(total, opts.raw), len(items))
	return nil
}

func printArtifactsShort(cmd *cobra.Command, workDir string, items []artifactItem, jsonOut, raw bool) error {
	counts := map[artifactKind]int{}
	bytes := map[artifactKind]int64{}
	var total int64
	for _, it := range items {
		counts[it.Kind]++
		bytes[it.Kind] += it.Bytes
		total += it.Bytes
	}

	if jsonOut {
		payload := map[string]any{
			"work_dir": workDir,
			"count":    len(items),
			"bytes":    total,
			"size":     formatBytes(total, raw),
			"logs": map[string]any{
				"count": counts[artifactLogs],
				"bytes": bytes[artifactLogs],
				"size":  formatBytes(bytes[artifactLogs], raw),
			},
			"volumes": map[string]any{
				"count": counts[artifactVolumes],
				"bytes": bytes[artifactVolumes],
				"size":  formatBytes(bytes[artifactVolumes], raw),
			},
		}
		enc, _ := json.MarshalIndent(payload, "", "  ")
		cmd.Println(string(enc))
		return nil
	}

	cmd.Printf("work_dir\t%s\n", workDir)
	cmd.Printf("logs\t%d items\t%s\n", counts[artifactLogs], formatBytes(bytes[artifactLogs], raw))
	cmd.Printf("volumes\t%d items\t%s\n", counts[artifactVolumes], formatBytes(bytes[artifactVolumes], raw))
	return nil
}

func newArtifactsPruneCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	opts := &artifactPruneOpts{}

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Delete deployment artifacts",
		Long: `Remove deployment artifacts under general.work_dir.

With no target, prunes both job logs and volumes:
  nunet artifacts prune             prune logs and volumes
  nunet artifacts prune logs        prune job allocation logs (work_dir/jobs)
  nunet artifacts prune volumes     prune local volumes (work_dir/volumes)

Shows matching items, then asks for confirmation before deleting.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o := *opts
			o.kinds = []artifactKind{artifactLogs, artifactVolumes}
			return runArtifactsPrune(cmd, dmsCli, o)
		},
	}

	bindArtifactsPruneFlags(cmd, opts)
	cmd.AddCommand(newArtifactsPruneKindCmd(dmsCli, opts, artifactLogs, "Delete job allocation logs under work_dir/jobs"))
	cmd.AddCommand(newArtifactsPruneKindCmd(dmsCli, opts, artifactVolumes, "Delete local volumes under work_dir/volumes"))
	return cmd
}

func newArtifactsPruneKindCmd(dmsCli *cli.DmsCLI, opts *artifactPruneOpts, kind artifactKind, short string) *cobra.Command {
	return &cobra.Command{
		Use:   string(kind),
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o := *opts
			o.kinds = []artifactKind{kind}
			return runArtifactsPrune(cmd, dmsCli, o)
		},
	}
}

func bindArtifactsPruneFlags(cmd *cobra.Command, opts *artifactPruneOpts) {
	cmd.PersistentFlags().StringVar(&opts.match, "match", "", "substring match on name/path (case-insensitive)")
	cmd.PersistentFlags().StringVar(&opts.glob, "glob", "", `glob match on base name (e.g. "*alloc*")`)
	cmd.PersistentFlags().DurationVar(&opts.older, "older-than", 0, "only prune items older than duration (e.g. 168h)")
	cmd.PersistentFlags().StringVar(&opts.sortBy, "sort", "size", `sort by: "size", "mtime", "name"`)
	cmd.PersistentFlags().BoolVar(&opts.reverse, "reverse", false, "reverse sort order")
}

func runArtifactsPrune(cmd *cobra.Command, dmsCli *cli.DmsCLI, opts artifactPruneOpts) error {
	cfg, err := dmsCli.Config()
	if err != nil {
		return err
	}

	items := collectArtifacts(dmsCli.FS(), cfg.General.WorkDir, opts.kinds)

	items = filterArtifacts(items, opts.match, opts.glob, opts.older)
	sortArtifacts(items, opts.sortBy, opts.reverse)

	if len(items) == 0 {
		cmd.Println("no matching artifacts")
		return nil
	}

	var total int64
	for _, it := range items {
		total += it.Bytes
		cmd.Printf("%s\t%s\t%s\n", it.Kind, formatBytes(it.Bytes, false), it.Path)
	}
	cmd.Printf("total\t%s\t%d items\n", formatBytes(total, false), len(items))

	ok, err := dmsUtils.PromptYesNo(
		cmd.InOrStdin(),
		cmd.OutOrStdout(),
		pruneConfirmPrompt(opts.kinds, len(items), total),
	)
	if err != nil {
		return err
	}
	if !ok {
		cmd.Println("aborted")
		return nil
	}

	fs := afero.Afero{Fs: dmsCli.FS()}
	for _, it := range items {
		if err := fs.RemoveAll(it.Path); err != nil {
			return fmt.Errorf("delete %s: %w", it.Path, err)
		}
		cmd.Printf("deleted\t%s\n", it.Path)
	}
	return nil
}

func pruneConfirmPrompt(kinds []artifactKind, count int, totalBytes int64) string {
	hasLogs, hasVolumes := false, false
	for _, k := range kinds {
		switch k {
		case artifactLogs:
			hasLogs = true
		case artifactVolumes:
			hasVolumes = true
		}
	}

	target := "artifacts"
	switch {
	case hasLogs && hasVolumes:
		target = "all job logs and volumes"
	case hasLogs:
		target = "job logs"
	case hasVolumes:
		target = "volumes"
	}

	return fmt.Sprintf("Delete %s: %d items (%s)", target, count, formatBytes(totalBytes, false))
}

func collectArtifacts(fs afero.Fs, workDir string, kinds []artifactKind) []artifactItem {
	kset := map[artifactKind]bool{}
	for _, k := range kinds {
		kset[k] = true
	}

	afs := afero.Afero{Fs: fs}
	var out []artifactItem

	if workDir == "" {
		return out
	}

	if kset[artifactLogs] {
		jobsRoot := filepath.Join(workDir, "jobs")
		children, _ := afero.ReadDir(fs, jobsRoot)
		for _, de := range children {
			if !de.IsDir() {
				continue
			}
			p := filepath.Join(jobsRoot, de.Name())
			b, mt := dirSizeAndLatestModTime(afs, p)
			out = append(out, artifactItem{Kind: artifactLogs, Name: de.Name(), Path: p, Bytes: b, ModTime: mt})
		}
	}

	if kset[artifactVolumes] {
		volRoot := filepath.Join(workDir, "volumes")
		owners, _ := afero.ReadDir(fs, volRoot)
		for _, owner := range owners {
			if !owner.IsDir() {
				continue
			}
			ownerPath := filepath.Join(volRoot, owner.Name())
			volumes, _ := afero.ReadDir(fs, ownerPath)
			foundChild := false
			for _, vol := range volumes {
				if !vol.IsDir() {
					continue
				}
				foundChild = true
				p := filepath.Join(ownerPath, vol.Name())
				b, mt := dirSizeAndLatestModTime(afs, p)
				out = append(out, artifactItem{
					Kind:    artifactVolumes,
					Name:    filepath.Join(owner.Name(), vol.Name()),
					Path:    p,
					Bytes:   b,
					ModTime: mt,
				})
			}
			if !foundChild {
				b, mt := dirSizeAndLatestModTime(afs, ownerPath)
				out = append(out, artifactItem{Kind: artifactVolumes, Name: owner.Name(), Path: ownerPath, Bytes: b, ModTime: mt})
			}
		}
	}

	return out
}

func filterArtifacts(items []artifactItem, match, glob string, olderThan time.Duration) []artifactItem {
	match = strings.ToLower(strings.TrimSpace(match))
	glob = strings.TrimSpace(glob)

	var cutoff time.Time
	if olderThan > 0 {
		cutoff = time.Now().Add(-olderThan)
	}

	out := make([]artifactItem, 0, len(items))
	for _, it := range items {
		if match != "" {
			hay := strings.ToLower(it.Name + "\n" + it.Path)
			if !strings.Contains(hay, match) {
				continue
			}
		}
		if glob != "" {
			ok, err := filepath.Match(glob, filepath.Base(it.Path))
			if err != nil || !ok {
				continue
			}
		}
		if !cutoff.IsZero() && !it.ModTime.IsZero() && !it.ModTime.Before(cutoff) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func sortArtifacts(items []artifactItem, sortBy string, reverse bool) {
	sortBy = strings.ToLower(strings.TrimSpace(sortBy))

	less := func(i, j int) bool { return items[i].Bytes < items[j].Bytes }
	switch sortBy {
	case "mtime", "time", "date":
		less = func(i, j int) bool { return items[i].ModTime.Before(items[j].ModTime) }
	case "name", "path":
		less = func(i, j int) bool { return items[i].Path < items[j].Path }
	}

	sort.Slice(items, func(i, j int) bool {
		if reverse {
			return !less(i, j)
		}
		return less(i, j)
	})
}

func dirSizeAndLatestModTime(afs afero.Afero, root string) (bytes int64, latest time.Time) {
	_ = afs.Walk(root, func(path string, info os.FileInfo, err error) error { //nolint
		if err != nil || info == nil {
			return nil
		}
		if info.Mode().IsRegular() {
			bytes += info.Size()
		}
		mt := info.ModTime()
		if latest.IsZero() || mt.After(latest) {
			latest = mt
		}
		return nil
	})
	return bytes, latest
}

func formatBytes(n int64, raw bool) string {
	if n < 0 {
		n = 0
	}
	if raw {
		return fmt.Sprintf("%d", n)
	}
	s, err := convert.ToBytesFormat(uint64(n))
	if err != nil {
		return fmt.Sprintf("%d", n)
	}
	return s
}

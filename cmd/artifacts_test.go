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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	cmdutils "gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/internal/config"
)

const testArtifactsWorkDir = "/tmp/nunet/work"

func setupArtifactFS(t *testing.T) afero.Fs {
	t.Helper()
	fs := afero.NewMemMapFs()
	afs := afero.Afero{Fs: fs}
	workDir := testArtifactsWorkDir

	require.NoError(t, afs.MkdirAll(filepath.Join(workDir, "jobs", "ensemble-alloc-1"), 0o755))
	require.NoError(t, afs.WriteFile(filepath.Join(workDir, "jobs", "ensemble-alloc-1", "stdout.log"), []byte("hello stdout"), 0o644))

	require.NoError(t, afs.MkdirAll(filepath.Join(workDir, "volumes", "owner-1", "data-vol"), 0o755))
	// Keep volume larger than the job log so --sort size --reverse is deterministic
	require.NoError(t, afs.WriteFile(filepath.Join(workDir, "volumes", "owner-1", "data-vol", "state.db"), []byte("volume-data-here!!!!"), 0o644))

	// DMS logs under work_dir/logs must never be collected by artifacts commands
	require.NoError(t, afs.MkdirAll(filepath.Join(workDir, "logs"), 0o755))
	require.NoError(t, afs.WriteFile(filepath.Join(workDir, "logs", "flightrec.trace"), []byte("trace"), 0o644))
	require.NoError(t, afs.WriteFile(filepath.Join(workDir, "logs", "nunet-dms-logs.jsonl"), []byte("logline"), 0o644))

	return fs
}

func testArtifactsCLI(t *testing.T, fs afero.Fs) *cli.DmsCLI {
	t.Helper()
	workDir := testArtifactsWorkDir
	cfg := config.DefaultConfig
	cfg.General.WorkDir = workDir
	cfg.General.DataDir = filepath.Join(workDir, "data")
	cfg.General.UserDir = filepath.Join(workDir, "user")
	cfg.Observability.Logging.File = filepath.Join(workDir, "logs", "nunet-dms-logs.jsonl")
	return cmdutils.NewTestCli(cli.WithFS(fs), cli.WithConfig(&cfg))
}

func TestCollectArtifacts(t *testing.T) {
	workDir := testArtifactsWorkDir
	fs := setupArtifactFS(t)

	items := collectArtifacts(fs, workDir, []artifactKind{artifactLogs, artifactVolumes})
	require.Len(t, items, 2)

	kinds := map[artifactKind]int{}
	for _, it := range items {
		kinds[it.Kind]++
		assert.NotEmpty(t, it.Path)
		assert.Greater(t, it.Bytes, int64(0))
		assert.NotContains(t, it.Path, filepath.Join(workDir, "logs"))
	}
	assert.Equal(t, 1, kinds[artifactLogs])
	assert.Equal(t, 1, kinds[artifactVolumes])
}

func TestArtifactsListCmd(t *testing.T) {
	fs := setupArtifactFS(t)
	dmsCli := testArtifactsCLI(t, fs)

	cmd := newArtifactsListCmd(dmsCli)
	out, _, err := cmdutils.ExecuteCommand(cmd, "--json", "--sort", "size", "--reverse")
	require.NoError(t, err)

	var items []artifactItem
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &items))
	require.Len(t, items, 2)
	assert.Equal(t, artifactVolumes, items[0].Kind)
	assert.Equal(t, artifactLogs, items[1].Kind)
	assert.GreaterOrEqual(t, items[0].Bytes, items[1].Bytes)
}

func TestArtifactsListLogsOnly(t *testing.T) {
	workDir := testArtifactsWorkDir
	fs := setupArtifactFS(t)
	dmsCli := testArtifactsCLI(t, fs)

	cmd := newArtifactsListCmd(dmsCli)
	out, _, err := cmdutils.ExecuteCommand(cmd, "logs", "--match", "ensemble")
	require.NoError(t, err)
	assert.Contains(t, out, "ensemble-alloc-1")
	assert.NotContains(t, out, "data-vol")
	assert.NotContains(t, out, filepath.Join(workDir, "logs"))
}

func TestArtifactsListVolumesOnly(t *testing.T) {
	fs := setupArtifactFS(t)
	dmsCli := testArtifactsCLI(t, fs)

	cmd := newArtifactsListCmd(dmsCli)
	out, _, err := cmdutils.ExecuteCommand(cmd, "volumes")
	require.NoError(t, err)
	assert.Contains(t, out, "data-vol")
	assert.NotContains(t, out, "ensemble-alloc-1")
}

func TestArtifactsPruneDefaultCleansEverything(t *testing.T) {
	workDir := testArtifactsWorkDir
	fs := setupArtifactFS(t)
	dmsCli := testArtifactsCLI(t, fs)
	afs := afero.Afero{Fs: fs}

	jobPath := filepath.Join(workDir, "jobs", "ensemble-alloc-1")
	volPath := filepath.Join(workDir, "volumes", "owner-1", "data-vol")
	dmsLog := filepath.Join(workDir, "logs", "nunet-dms-logs.jsonl")

	prune := newArtifactsPruneCmd(dmsCli)
	out, _, err := cmdutils.ExecuteCommandWithInput(prune, [][]byte{[]byte("y\n")})
	require.NoError(t, err)
	assert.Contains(t, out, "all job logs and volumes")
	assert.Contains(t, out, jobPath)
	assert.Contains(t, out, volPath)

	_, err = afs.Stat(jobPath)
	assert.Error(t, err)
	_, err = afs.Stat(volPath)
	assert.Error(t, err)
	_, err = afs.Stat(dmsLog)
	require.NoError(t, err)
}

func TestArtifactsPruneLogsConfirmAndCancel(t *testing.T) {
	workDir := testArtifactsWorkDir
	fs := setupArtifactFS(t)
	dmsCli := testArtifactsCLI(t, fs)
	afs := afero.Afero{Fs: fs}

	jobPath := filepath.Join(workDir, "jobs", "ensemble-alloc-1")
	volPath := filepath.Join(workDir, "volumes", "owner-1", "data-vol")
	_, err := afs.Stat(jobPath)
	require.NoError(t, err)

	prune := newArtifactsPruneCmd(dmsCli)
	out, _, err := cmdutils.ExecuteCommandWithInput(prune, [][]byte{[]byte("n\n")}, "logs", "--match", "ensemble")
	require.NoError(t, err)
	assert.Contains(t, out, jobPath)
	assert.Contains(t, out, "job logs")
	assert.Contains(t, out, "aborted")

	_, err = afs.Stat(jobPath)
	require.NoError(t, err)

	out, _, err = cmdutils.ExecuteCommandWithInput(prune, [][]byte{[]byte("y\n")}, "logs", "--match", "ensemble")
	require.NoError(t, err)
	assert.Contains(t, out, "deleted")

	_, err = afs.Stat(jobPath)
	assert.Error(t, err)
	_, err = afs.Stat(volPath)
	require.NoError(t, err)
}

func TestArtifactsPruneVolumes(t *testing.T) {
	workDir := testArtifactsWorkDir
	fs := setupArtifactFS(t)
	dmsCli := testArtifactsCLI(t, fs)
	afs := afero.Afero{Fs: fs}

	volPath := filepath.Join(workDir, "volumes", "owner-1", "data-vol")
	jobPath := filepath.Join(workDir, "jobs", "ensemble-alloc-1")

	prune := newArtifactsPruneCmd(dmsCli)
	out, _, err := cmdutils.ExecuteCommandWithInput(prune, [][]byte{[]byte("y\n")}, "volumes")
	require.NoError(t, err)
	assert.Contains(t, out, "volumes")
	assert.Contains(t, out, volPath)
	assert.NotContains(t, out, jobPath)

	_, err = afs.Stat(volPath)
	assert.Error(t, err)
	_, err = afs.Stat(jobPath)
	require.NoError(t, err)
}

func TestArtifactsPruneLogsOlderThan(t *testing.T) {
	workDir := testArtifactsWorkDir
	fs := afero.NewMemMapFs()
	afs := afero.Afero{Fs: fs}

	oldJob := filepath.Join(workDir, "jobs", "old-alloc")
	require.NoError(t, afs.MkdirAll(oldJob, 0o755))
	require.NoError(t, afs.WriteFile(filepath.Join(oldJob, "stdout.log"), []byte("old"), 0o644))

	newJob := filepath.Join(workDir, "jobs", "new-alloc")
	require.NoError(t, afs.MkdirAll(newJob, 0o755))
	require.NoError(t, afs.WriteFile(filepath.Join(newJob, "stdout.log"), []byte("new"), 0o644))

	oldTime := time.Now().Add(-48 * time.Hour)
	require.NoError(t, afs.Chtimes(oldJob, oldTime, oldTime))
	require.NoError(t, afs.Chtimes(filepath.Join(oldJob, "stdout.log"), oldTime, oldTime))

	dmsCli := testArtifactsCLI(t, fs)
	prune := newArtifactsPruneCmd(dmsCli)
	out, _, err := cmdutils.ExecuteCommandWithInput(prune, [][]byte{[]byte("y\n")}, "logs", "--older-than", "24h")
	require.NoError(t, err)
	assert.Contains(t, out, "old-alloc")
	assert.NotContains(t, out, "new-alloc")

	_, err = afs.Stat(oldJob)
	assert.Error(t, err)
	_, err = afs.Stat(newJob)
	require.NoError(t, err)
}

func TestArtifactsListShortFlag(t *testing.T) {
	workDir := testArtifactsWorkDir
	fs := setupArtifactFS(t)
	dmsCli := testArtifactsCLI(t, fs)

	cmd := newArtifactsListCmd(dmsCli)
	out, _, err := cmdutils.ExecuteCommand(cmd, "--short")
	require.NoError(t, err)
	assert.Contains(t, out, "work_dir")
	assert.Contains(t, out, workDir)
	assert.Contains(t, out, "logs")
	assert.Contains(t, out, "volumes")
	assert.Contains(t, out, "1 items")
	assert.NotContains(t, out, "ensemble-alloc-1")
}

func TestArtifactsDefaultIsListShort(t *testing.T) {
	fs := setupArtifactFS(t)
	dmsCli := testArtifactsCLI(t, fs)

	cmd := newArtifactsCmd(dmsCli)
	out, _, err := cmdutils.ExecuteCommand(cmd)
	require.NoError(t, err)
	assert.Contains(t, out, "work_dir")
	assert.Contains(t, out, "logs")
	assert.Contains(t, out, "volumes")
	assert.NotContains(t, out, "ensemble-alloc-1")
}

func TestFormatBytes(t *testing.T) {
	assert.Equal(t, "512 B", formatBytes(512, false))
	assert.Equal(t, "1.0 kB", formatBytes(1024, false))
	assert.Equal(t, "1536", formatBytes(1536, true))
}

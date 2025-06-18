// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

// TODO: tests

// EnsembleCfgReader provides read-only access to an EnsembleConfig
type EnsembleCfgReader struct {
	cfg EnsembleConfig
}

// NewEnsembleCfgReader creates a new reader with a deep copy of the config
func NewEnsembleCfgReader(cfg EnsembleConfig) EnsembleCfgReader {
	return EnsembleCfgReader{cfg: cfg.Clone()}
}

// Read returns the Reader's config which was cloned
// from another payload by the constructor.
func (r EnsembleCfgReader) Read() EnsembleConfig {
	return r.cfg
}

// ManifestReader provides read-only access to an EnsembleManifest
type ManifestReader struct {
	manifest EnsembleManifest
}

// NewManifestReader creates a new reader with a deep copy of the manifest
func NewManifestReader(manifest EnsembleManifest) ManifestReader {
	return ManifestReader{manifest: manifest.Clone()}
}

// Read returns the Reader's config which was cloned
// from another payload by the constructor.
func (r ManifestReader) Read() EnsembleManifest {
	return r.manifest
}

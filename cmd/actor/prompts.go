// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"fmt"
	"io"

	"github.com/manifoldco/promptui"
	"gitlab.com/nunet/device-management-service/cmd/cli"
)

type selectPromptItem struct {
	Label    string
	Selected bool
}

type writerCloser struct {
	io.Writer
}

func (wc *writerCloser) Close() error {
	return nil
}

func runSelectPrompt(label string, items []*selectPromptItem, multiple bool, streams cli.Streams) ([]string, error) {
	const doneLabel = "Done"
	if multiple && len(items) > 0 && items[0].Label != doneLabel {
		items = append([]*selectPromptItem{{Label: doneLabel}}, items...)
	}

	template := &promptui.SelectTemplates{
		Label:    "{{ .Label }}",
		Active:   "{{ if .Selected }}{{ \"✓\" | green }} {{ end }}→ {{ .Label | cyan | bold }}",
		Inactive: "{{ if .Selected }}{{ \"✓\" | green }} {{ end }}  {{ .Label | faint }}",
		Selected: "{{ .Label | green | bold }}",
	}

	p := promptui.Select{
		Label:        label,
		Items:        items,
		Templates:    template,
		HideSelected: true,
		Size:         len(items),
		Stdin:        io.NopCloser(streams.In),
		Stdout:       &writerCloser{streams.Out},
	}

	for done := false; !done; {
		index, _, err := p.Run()
		if err != nil {
			return nil, fmt.Errorf("prompt failed %w", err)
		}
		selectedItem := items[index]
		if multiple && selectedItem.Label != doneLabel {
			selectedItem.Selected = !selectedItem.Selected
			done = false
		} else {
			done = true
		}
	}

	var selected []string
	for _, item := range items {
		if item.Selected {
			selected = append(selected, item.Label)
		}
	}

	return selected, nil
}

func selectPromptMultiple(label string, items []*selectPromptItem, streams cli.Streams) ([]string, error) {
	return runSelectPrompt(label, items, true, streams)
}

func prompt(label string, validate func(string) error, streams cli.Streams) (string, error) {
	p := promptui.Prompt{
		Label: label,
		Templates: &promptui.PromptTemplates{
			Valid: "{{ \"✓\" | green }} {{ . }} {{ \":\" | bold}} ",
		},
		Validate: validate,
		Stdin:    io.NopCloser(streams.In),
		Stdout:   &writerCloser{streams.Out},
	}

	result, err := p.Run()
	return result, err
}

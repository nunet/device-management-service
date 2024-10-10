package actor

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

type selectPromptItem struct {
	Label    string
	Selected bool
}

func selectPrompt(label string, items []*selectPromptItem) ([]string, error) {
	// Always prepend a "Done" item to the slice if it doesn't
	// already exist.
	const doneLabel = "Done"
	if len(items) > 0 && items[0].Label != doneLabel {
		items = append([]*selectPromptItem{{Label: doneLabel}}, items...)
	}

	template := &promptui.SelectTemplates{
		Label:    "{{ .Label }}",
		Active:   "{{ if .Selected }}{{ \"✔\" | green }} {{ end }}→ {{ .Label | cyan | bold }}",
		Inactive: "{{ if .Selected }}{{ \"✔\" | green }} {{ end }}  {{ .Label | faint }}",
		Selected: "{{ .Label | green | bold }}",
	}

	p := promptui.Select{
		Label:     label,
		Items:     items,
		Templates: template,
		Size:      len(items),
	}

	index, _, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt failed %w", err)
	}

	selectedItem := items[index]
	if selectedItem.Label != doneLabel {
		selectedItem.Selected = !selectedItem.Selected
		return selectPrompt(label, items)
	}

	var selected []string
	for _, item := range items {
		if item.Selected {
			selected = append(selected, item.Label)
		}
	}

	return selected, nil
}

func prompt(label string, validate func(string) error) (string, error) {
	p := promptui.Prompt{
		Label:    label,
		Validate: validate,
	}

	result, err := p.Run()
	return result, err
}

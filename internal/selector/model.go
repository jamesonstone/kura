package selector

type Item struct {
	ID          string
	Name        string
	Description string
}

type Model struct {
	items    []Item
	focused  int
	selected map[string]bool
}

func NewModel(items []Item) *Model {
	return &Model{items: append([]Item(nil), items...), selected: make(map[string]bool)}
}

func (model *Model) Move(delta int) {
	if len(model.items) == 0 {
		return
	}
	model.focused = (model.focused + delta + len(model.items)) % len(model.items)
}

func (model *Model) Toggle() {
	if len(model.items) == 0 {
		return
	}
	id := model.items[model.focused].ID
	model.selected[id] = !model.selected[id]
}

func (model *Model) Selected() []string {
	result := make([]string, 0, len(model.selected))
	for _, item := range model.items {
		if model.selected[item.ID] {
			result = append(result, item.ID)
		}
	}
	return result
}

func (model *Model) Items() []Item {
	return append([]Item(nil), model.items...)
}

func (model *Model) Focused() int {
	return model.focused
}

func (model *Model) IsSelected(id string) bool {
	return model.selected[id]
}

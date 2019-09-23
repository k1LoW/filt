package history

type History struct {
	path string
	data []string
}

// New ...
func New(path string) *History {
	return &History{
		path: path,
		data: []string{},
	}
}

func (h *History) Load() error {
	return nil
}

func (h *History) Save() error {
	return nil
}

func (h *History) Raw() []string {
	return h.data
}

func (h *History) Append(inputStr string) error {
	h.data = unique(append([]string{inputStr}, h.data...))
	return nil
}

func unique(strs []string) []string {
	keys := make(map[string]bool)
	uniqStrs := []string{}
	for _, s := range strs {
		if s == "" {
			continue
		}
		if _, value := keys[s]; !value {
			keys[s] = true
			uniqStrs = append(uniqStrs, s)
		}
	}
	return uniqStrs
}

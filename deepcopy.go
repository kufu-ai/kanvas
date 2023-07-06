package kanvas

import "encoding/json"

func DeepCopyComponent(c Component) (*Component, error) {
	a, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	var b Component
	if err := json.Unmarshal(a, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

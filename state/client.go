package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	_ "time"
)

type ContainerState struct {
	ID string `json:"id"`
	// PID     int       `json:"pid"`
	// Status  string    `json:"status"`
	// Image   string    `json:"image"`
	// Rootfs  string    `json:"rootfs"`
	// Created time.Time `json:"created"`
}

func WriteState(state ContainerState) error {
	target := fmt.Sprintf("/run/vessetrt/%s/state.json", state.ID)

	os.MkdirAll(filepath.Dir(target), 0755)

	f, _ := os.Create(target)
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	data, err := json.Marshal(state)

	if err != nil {
		return fmt.Errorf("error encoding state to json: %v", err)
	}
	_ = encoder.Encode(string(data))
	return nil
}

func ReadState(containerID string) (ContainerState, error) {
	target := filepath.Join("/run/vessetrt/%s/state.json", containerID)
	file, err := os.ReadFile(target)
	var state *ContainerState = &ContainerState{}
	if err != nil {
		return *state, fmt.Errorf("error reading state from disk: %v", err)
	}
	err = json.Unmarshal(file, state)
	if err != nil {
		return *state, fmt.Errorf("error parsing state json: %v", err)
	}
	return *state, nil
}

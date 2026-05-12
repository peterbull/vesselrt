package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ContainerState struct {
	ID      string    `json:"id"`
	PID     int       `json:"pid"`
	Status  string    `json:"status"`
	Image   string    `json:"image"`
	Rootfs  string    `json:"rootfs"`
	Created time.Time `json:"created"`
}

func DeleteState(containerId string) error {
	target := fmt.Sprintf("/run/vesselrt/%s", containerId)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("couldn't remove state file: %v", err)
	}
	return nil
}

func WriteState(state ContainerState) error {
	target := fmt.Sprintf("/run/vesselrt/%s/state.json", state.ID)

	os.MkdirAll(filepath.Dir(target), 0755)

	f, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("error creating state file: %v", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(state); err != nil {
		return fmt.Errorf("error encoding state to json: %v", err)
	}
	return nil
}

func ReadState(containerID string) (ContainerState, error) {
	target := fmt.Sprintf("/run/vesselrt/%s/state.json", containerID)
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

package types

import (
	"encoding/json"
	"testing"
)

func TestDriverRecoveryConfigUnmarshal(t *testing.T) {
	config := ResourceConfig{}
	err := json.Unmarshal([]byte(`{"resourceName":"mlx5","driverRecovery":{"desiredDriver":"mlx5_core"}}`), &config)
	if err != nil {
		t.Fatalf("failed to unmarshal resource config: %v", err)
	}
	if config.DriverRecovery == nil {
		t.Fatal("driverRecovery was not unmarshaled")
	}
	if config.DriverRecovery.DesiredDriver != "mlx5_core" {
		t.Fatalf("unexpected desired driver: %q", config.DriverRecovery.DesiredDriver)
	}
}

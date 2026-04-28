package fibonacci

import (
	"testing"

	"github.com/agbru/fibcalc/internal/fibonacci/threshold"
)

func TestNewDoublingFrameworkWithDynamicThresholds(t *testing.T) {
	t.Parallel()

	t.Run("Create with dynamic thresholds", func(t *testing.T) {
		t.Parallel()
		strategy := &AdaptiveStrategy{}
		dtm := threshold.NewDynamicThresholdManager(1000000, 4096)

		framework := NewDoublingFrameworkWithDynamicThresholds(strategy, dtm)

		if framework == nil {
			t.Fatal("Framework should not be nil")
		}
		if framework.strategy != strategy {
			t.Error("Strategy should be set correctly")
		}
		if framework.dynamicThreshold != dtm {
			t.Error("Dynamic threshold manager should be set correctly")
		}
	})

	t.Run("Create with nil dynamic thresholds", func(t *testing.T) {
		t.Parallel()
		strategy := &AdaptiveStrategy{}

		framework := NewDoublingFrameworkWithDynamicThresholds(strategy, nil)

		if framework == nil {
			t.Fatal("Framework should not be nil")
		}
		if framework.strategy != strategy {
			t.Error("Strategy should be set correctly")
		}
		if framework.dynamicThreshold != nil {
			t.Error("Dynamic threshold manager should be nil")
		}
	})

}

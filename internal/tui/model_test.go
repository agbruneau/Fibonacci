package tui

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/progress"
)

// mockCalculator implements fibonacci.Calculator for testing.
type mockCalculator struct {
	name string
}

func (m mockCalculator) Calculate(_ context.Context, _ chan<- progress.ProgressUpdate, _ int, _ uint64, _ fibonacci.Options) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (m mockCalculator) Name() string { return m.name }

// Verify interface compliance at compile time.
var _ fibonacci.Calculator = mockCalculator{}

func newTestModel(t *testing.T) Model {
	t.Helper()
	ctx := context.Background()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	m := NewModel(ctx, nil, cfg, "v0.1.0")
	t.Cleanup(m.cancel)
	return m
}

func newTestModelWithSize(t *testing.T, w, h int) Model {
	t.Helper()
	m := newTestModel(t)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(Model)
}

func TestNewModel(t *testing.T) {
	t.Parallel()
	model := newTestModel(t)

	if model.paused {
		t.Error("expected model to not be paused initially")
	}
	if model.done {
		t.Error("expected model to not be done initially")
	}
	if model.exitCode != apperrors.ExitSuccess {
		t.Errorf("expected exit code %d, got %d", apperrors.ExitSuccess, model.exitCode)
	}
}

func TestNewModel_WithCalculators(t *testing.T) {
	t.Parallel()
	calcs := []fibonacci.Calculator{
		mockCalculator{name: "Fast Doubling"},
		mockCalculator{name: "Matrix"},
	}
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	model := NewModel(context.Background(), calcs, cfg, "v1.0.0")
	defer model.cancel()

	if len(model.calculators) != 2 {
		t.Errorf("expected 2 calculators, got %d", len(model.calculators))
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	model := NewModel(context.Background(), nil, cfg, "v0.1.0")
	defer model.cancel()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, cmd := model.Update(msg)
	m := updated.(Model)

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
	if cmd != nil {
		t.Error("expected no command from window size update")
	}
}

func TestModel_Update_ProgressMsg(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	model := NewModel(context.Background(), nil, cfg, "v0.1.0")
	defer model.cancel()

	// Set size first so viewport is initialized
	sized, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := sized.(Model)

	msg := ProgressMsg{
		CalculatorIndex: 0,
		Value:           0.5,
		AverageProgress: 0.5,
		ETA:             30 * time.Second,
	}
	updated, _ := m.Update(msg)
	result := updated.(Model)

	if result.chart.averageProgress == 0 {
		t.Error("expected chart to have progress after progress update")
	}
}

func TestModel_Update_ProgressMsg_Paused(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	model := NewModel(context.Background(), nil, cfg, "v0.1.0")
	defer model.cancel()
	model.paused = true

	msg := ProgressMsg{
		CalculatorIndex: 0,
		Value:           0.5,
		AverageProgress: 0.5,
		ETA:             30 * time.Second,
	}
	updated, _ := model.Update(msg)
	result := updated.(Model)

	if result.chart.averageProgress != 0 {
		t.Error("expected chart to have no progress when paused")
	}
}

func TestModel_Update_CalculationComplete(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	model := NewModel(context.Background(), nil, cfg, "v0.1.0")
	defer model.cancel()

	msg := CalculationCompleteMsg{ExitCode: 0}
	updated, _ := model.Update(msg)
	result := updated.(Model)

	if !result.done {
		t.Error("expected model to be done after calculation complete")
	}
	if result.exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.exitCode)
	}
}

func TestModel_Update_ErrorMsg(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	model := NewModel(context.Background(), nil, cfg, "v0.1.0")
	defer model.cancel()

	// Set size first
	sized, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := sized.(Model)

	msg := ErrorMsg{Err: context.DeadlineExceeded, Duration: time.Second}
	updated, _ := m.Update(msg)
	result := updated.(Model)

	if !result.done {
		t.Error("expected model to be done after error")
	}
	if !result.footer.hasErr {
		t.Error("expected footer to show error state")
	}
}

func TestModel_View_Initializing(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	model := NewModel(context.Background(), nil, cfg, "v0.1.0")
	defer model.cancel()

	view := model.View()
	if view != "Initializing..." {
		t.Errorf("expected 'Initializing...' when no size set, got %q", view)
	}
}

func TestModel_View_WithSize(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	model := NewModel(context.Background(), nil, cfg, "v0.1.0")
	defer model.cancel()

	sized, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := sized.(Model)

	view := m.View()
	if view == "Initializing..." {
		t.Error("expected rendered dashboard, got 'Initializing...'")
	}
	if len(view) == 0 {
		t.Error("expected non-empty view")
	}
}

func TestModel_HandleKey_Pause(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	model := NewModel(context.Background(), nil, cfg, "v0.1.0")
	defer model.cancel()

	// Press space to pause
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	m := updated.(Model)
	if !m.paused {
		t.Error("expected model to be paused after space key")
	}

	// Press space to resume
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	if m.paused {
		t.Error("expected model to be unpaused after second space key")
	}
}

func TestModel_HandleKey_Restart(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 80, 24)

	// Add some chart data and log entries
	m.chart.AddDataPoint(0.5, 0.5, 10*time.Second)
	m.logs.AddProgressEntry(ProgressMsg{CalculatorIndex: 0, Value: 0.5})
	m.done = true
	m.footer.SetDone(true)

	initialGen := m.generation

	// Press 'r' to restart
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	result := updated.(Model)

	if result.chart.averageProgress != 0 {
		t.Error("expected chart to be reset after restart")
	}
	if len(result.logs.entries) != 0 {
		t.Error("expected logs to be cleared after restart")
	}
	if result.done {
		t.Error("expected done to be false after restart")
	}
	if result.generation != initialGen+1 {
		t.Errorf("expected generation %d, got %d", initialGen+1, result.generation)
	}
	if cmd == nil {
		t.Error("expected commands to be returned for restarting calculation")
	}
}

func TestModel_Update_ComparisonResultsMsg(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 120, 40)

	msg := ComparisonResultsMsg{
		Results: []orchestration.CalculationResult{
			{Name: "Fast", Duration: 100 * time.Millisecond},
		},
	}
	updated, cmd := m.Update(msg)
	result := updated.(Model)

	if cmd != nil {
		t.Error("expected no command from comparison results")
	}
	if len(result.logs.entries) == 0 {
		t.Error("expected logs to have entries after comparison results")
	}
}

func TestModel_Update_FinalResultMsg(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 120, 40)

	msg := FinalResultMsg{
		Result: orchestration.CalculationResult{
			Name:     "Fast",
			Result:   big.NewInt(55),
			Duration: 50 * time.Millisecond,
		},
		N:       10,
		Verbose: true,
	}
	updated, cmd := m.Update(msg)
	result := updated.(Model)

	if cmd == nil {
		t.Error("expected a command to compute indicators from final result")
	}
	if len(result.logs.entries) == 0 {
		t.Error("expected logs to have entries after final result")
	}

	// Execute the command and verify it returns an IndicatorsMsg
	indicatorsMsg := cmd()
	if _, ok := indicatorsMsg.(IndicatorsMsg); !ok {
		t.Errorf("expected IndicatorsMsg, got %T", indicatorsMsg)
	}
}

func TestModel_Update_ProgressDoneMsg(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	updated, cmd := m.Update(ProgressDoneMsg{})
	result := updated.(Model)

	if cmd != nil {
		t.Error("expected no command from progress done")
	}
	// ProgressDoneMsg is a no-op, model state should not change
	if result.done {
		t.Error("expected model not to be done after ProgressDoneMsg")
	}
}

func TestModel_Update_ContextCancelledMsg(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	msg := ContextCancelledMsg{Err: context.Canceled, Generation: m.generation}
	updated, cmd := m.Update(msg)
	result := updated.(Model)

	if !result.done {
		t.Error("expected model to be done after context cancelled")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command from context cancelled")
	}
}

func TestModel_Update_ContextCancelledMsg_StaleGeneration(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	// Stale generation should be ignored
	msg := ContextCancelledMsg{Err: context.Canceled, Generation: m.generation + 1}
	updated, cmd := m.Update(msg)
	result := updated.(Model)

	if result.done {
		t.Error("expected stale context cancelled to be ignored")
	}
	if cmd != nil {
		t.Error("expected no command from stale context cancelled")
	}
}

func TestModel_Update_CalculationComplete_NonZeroExitCode(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	msg := CalculationCompleteMsg{ExitCode: apperrors.ExitErrorMismatch}
	updated, _ := m.Update(msg)
	result := updated.(Model)

	if result.exitCode != apperrors.ExitErrorMismatch {
		t.Errorf("expected exit code %d, got %d", apperrors.ExitErrorMismatch, result.exitCode)
	}
	if !result.footer.done {
		t.Error("expected footer to show done state")
	}
}

func TestModel_Update_MemStatsMsg(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	msg := MemStatsMsg{
		Alloc:        1024 * 1024 * 10,
		NumGC:        5,
		NumGoroutine: 12,
	}
	updated, _ := m.Update(msg)
	result := updated.(Model)

	if result.metrics.alloc != msg.Alloc {
		t.Errorf("expected alloc %d, got %d", msg.Alloc, result.metrics.alloc)
	}
}

func TestModel_Update_TickMsg_NotPaused(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	updated, cmd := m.Update(TickMsg(time.Now()))
	_ = updated

	// When not paused, should return a batch of sampleMemStatsCmd + tickCmd
	if cmd == nil {
		t.Error("expected commands from tick when not paused")
	}
}

func TestModel_Update_TickMsg_Paused(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)
	m.paused = true

	updated, cmd := m.Update(TickMsg(time.Now()))
	_ = updated

	// When paused, should return only tickCmd (no mem stats sampling)
	if cmd == nil {
		t.Error("expected tick command even when paused")
	}
}

func TestModel_HandleKey_Quit_Q(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected tea.Quit command from 'q' key")
	}
}

func TestModel_HandleKey_Quit_CtrlC(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected tea.Quit command from Ctrl+C")
	}
}

func TestModel_HandleKey_ScrollKeys(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 120, 40)

	// Add enough content to scroll
	for i := 0; i < 50; i++ {
		m.logs.AddProgressEntry(ProgressMsg{CalculatorIndex: 0, Value: float64(i) / 50})
	}

	keys := []tea.KeyMsg{
		{Type: tea.KeyUp},
		{Type: tea.KeyDown},
		{Type: tea.KeyPgUp},
		{Type: tea.KeyPgDown},
		{Type: tea.KeyRunes, Runes: []rune{'k'}},
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
	}
	for _, k := range keys {
		updated, cmd := m.Update(k)
		m = updated.(Model)
		if cmd != nil {
			t.Errorf("expected no command from scroll key %v", k)
		}
	}
}

func TestModel_HandleKey_Unknown(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	result := updated.(Model)

	if cmd != nil {
		t.Error("expected no command from unknown key")
	}
	if result.paused {
		t.Error("unknown key should not change state")
	}
}

func TestModel_Update_UnknownMsg(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	// Send an arbitrary message type
	type unknownMsg struct{}
	updated, cmd := m.Update(unknownMsg{})
	_ = updated

	if cmd != nil {
		t.Error("expected no command from unknown message type")
	}
}

func TestModel_LayoutPanels_VerySmallTerminal(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	// Very small terminal — bodyHeight would be negative without the clamp
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 8})
	result := updated.(Model)

	// Should not panic and should render something
	view := result.View()
	if len(view) == 0 {
		t.Error("expected non-empty view for small terminal")
	}
}

func TestModel_LayoutPanels_MinBodyHeight(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)

	// Height = 6 means bodyHeight = 0 -> clamped to 4
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 6})
	result := updated.(Model)

	view := result.View()
	if view == "Initializing..." {
		t.Error("expected rendered dashboard, not initializing")
	}
}

func TestModel_HandleKey_Restart_ClearsMetrics(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 80, 24)

	// Set some metrics
	m.metrics.lastUpdate = time.Now().Add(-time.Second)
	m.metrics.UpdateProgress(0.5)

	if m.metrics.speed == 0 {
		t.Fatal("precondition: metrics speed should be non-zero")
	}

	// Restart
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	result := updated.(Model)

	if result.metrics.speed != 0 {
		t.Error("expected metrics speed to be reset to 0")
	}
}

func TestModel_Init_ReturnsCommands(t *testing.T) {
	t.Parallel()
	m := newTestModel(t)
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected Init to return a non-nil command batch")
	}
}

func TestSampleMemStatsCmd_ReturnsMemStatsMsg(t *testing.T) {
	t.Parallel()
	cmd := sampleMemStatsCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command from sampleMemStatsCmd")
	}
	msg := cmd()
	if _, ok := msg.(MemStatsMsg); !ok {
		t.Errorf("expected MemStatsMsg, got %T", msg)
	}
}

func TestWatchContextCmd_SendsOnCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := watchContextCmd(ctx, 0)
	if cmd == nil {
		t.Fatal("expected non-nil command from watchContextCmd")
	}

	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()

	cancel()

	select {
	case msg := <-done:
		if _, ok := msg.(ContextCancelledMsg); !ok {
			t.Errorf("expected ContextCancelledMsg, got %T", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ContextCancelledMsg")
	}
}

func TestWatchContextCmd_AlreadyCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before creating cmd

	cmd := watchContextCmd(ctx, 0)
	msg := cmd()
	if _, ok := msg.(ContextCancelledMsg); !ok {
		t.Errorf("expected ContextCancelledMsg, got %T", msg)
	}
}

func TestModel_layoutPanels_Percentages(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 100, 40)

	// Logs should get 60% of width
	expectedLogsWidth := 100 * 60 / 100
	if m.logs.width != expectedLogsWidth {
		t.Errorf("expected logs width %d, got %d", expectedLogsWidth, m.logs.width)
	}

	// Right column gets the rest
	expectedRightWidth := 100 - expectedLogsWidth
	if m.metrics.width != expectedRightWidth {
		t.Errorf("expected metrics width %d, got %d", expectedRightWidth, m.metrics.width)
	}
	if m.chart.width != expectedRightWidth {
		t.Errorf("expected chart width %d, got %d", expectedRightWidth, m.chart.width)
	}
}

func TestModel_layoutPanels_MinimumSizes(t *testing.T) {
	t.Parallel()
	// Very small terminal: bodyHeight would be negative, clamped to minBodyHeight
	m := newTestModelWithSize(t, 20, 4)

	// bodyHeight = 4 - 3 - 3 = -2, clamped to 4
	expectedBodyHeight := minBodyHeight
	// Logs height should equal bodyHeight
	if m.logs.height != expectedBodyHeight {
		t.Errorf("expected logs height %d, got %d", expectedBodyHeight, m.logs.height)
	}
}

func TestModel_metricsHeight_ConsistentWithLayout(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 100, 40)

	// metricsHeight() should match the value assigned during layoutPanels
	if m.metricsHeight() != m.metrics.height {
		t.Errorf("metricsHeight()=%d differs from metrics.height=%d", m.metricsHeight(), m.metrics.height)
	}
}

func TestModel_View_ContainsAllComponents(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 120, 40)

	view := m.View()
	if len(view) == 0 {
		t.Fatal("expected non-empty view")
	}
	// The view should contain the FibGo title from header
	if !strings.Contains(view, "FibGo") {
		t.Error("expected view to contain 'FibGo' from header")
	}
	// The view should contain Heap from the metrics panel
	if !strings.Contains(view, "Heap") {
		t.Error("expected view to contain 'Heap' from metrics panel")
	}
	// The view should contain Progress Chart from the chart panel
	if !strings.Contains(view, "Progress Chart") {
		t.Error("expected view to contain 'Progress Chart' from chart panel")
	}
}

func TestModel_Update_VeryWideTerminal(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 500, 40)

	// Should not panic
	view := m.View()
	if len(view) == 0 {
		t.Error("expected non-empty view for wide terminal")
	}
}

func TestModel_metricsHeight_SmallTerminal(t *testing.T) {
	t.Parallel()
	// When height is very small, bodyHeight gets clamped to minBodyHeight.
	// metricsFixedH (8) is capped at bodyHeight*2/3 = 4*2/3 = 2.
	m := newTestModelWithSize(t, 80, 4)
	mh := m.metricsHeight()
	expected := minBodyHeight * 2 / 3
	if mh != expected {
		t.Errorf("expected metricsHeight=%d for small terminal, got %d", expected, mh)
	}
}

func TestTickCmd_ReturnsCmd(t *testing.T) {
	t.Parallel()
	cmd := tickCmd()
	if cmd == nil {
		t.Error("expected non-nil command from tickCmd")
	}
}

func TestStartCalculationCmd_ReturnsCompleteMsg(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calcs := []fibonacci.Calculator{mockCalculator{name: "Fast"}}
	cfg := config.AppConfig{N: 10, Timeout: 10 * time.Second}
	cmd := startCalculationCmd(ref, ctx, calcs, cfg, 0)
	if cmd == nil {
		t.Fatal("expected non-nil command from startCalculationCmd")
	}

	msg := cmd()
	if _, ok := msg.(CalculationCompleteMsg); !ok {
		t.Errorf("expected CalculationCompleteMsg, got %T", msg)
	}
}

func TestModel_Update_SysStatsMsg(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 80, 24)

	msg := SysStatsMsg{CPUPercent: 25.5, MemPercent: 60.0}
	updated, cmd := m.Update(msg)
	result := updated.(Model)

	if cmd != nil {
		t.Error("expected no command from SysStatsMsg")
	}
	if result.chart.cpuHistory.Len() != 1 {
		t.Errorf("expected 1 cpu sample, got %d", result.chart.cpuHistory.Len())
	}
	if result.chart.memHistory.Len() != 1 {
		t.Errorf("expected 1 mem sample, got %d", result.chart.memHistory.Len())
	}
	if result.chart.cpuHistory.Last() != 25.5 {
		t.Errorf("expected cpu 25.5, got %f", result.chart.cpuHistory.Last())
	}
}

func TestSampleSysStatsCmd_ReturnsSysStatsMsg(t *testing.T) {
	t.Parallel()
	cmd := sampleSysStatsCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command from sampleSysStatsCmd")
	}
	msg := cmd()
	if _, ok := msg.(SysStatsMsg); !ok {
		t.Errorf("expected SysStatsMsg, got %T", msg)
	}
}

func TestModel_HandleKey_Restart_ClearsSysStats(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 80, 24)

	m.chart.UpdateSysStats(50.0, 70.0)
	m.chart.UpdateSysStats(55.0, 72.0)

	if m.chart.cpuHistory.Len() != 2 {
		t.Fatal("precondition: expected 2 cpu samples")
	}

	// Press 'r' to restart
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	result := updated.(Model)

	if result.chart.cpuHistory.Len() != 0 {
		t.Error("expected cpuHistory to be cleared after restart")
	}
	if result.chart.memHistory.Len() != 0 {
		t.Error("expected memHistory to be cleared after restart")
	}
}

func TestModel_HandleKey_Quit_CancelsContext(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	m := NewModel(context.Background(), nil, cfg, "v1.0.0")

	calcCtx := m.ctx
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Child context should be cancelled after quit
	select {
	case <-calcCtx.Done():
		// Good — context was cancelled
	default:
		t.Error("expected context to be cancelled after quit")
	}
}

// --- R4.10: adaptive layout for narrow / short terminals ---

// TestModel_AdaptiveLayout_Narrow verifies that a 60x18 terminal switches to
// the single-column layout (logs full width, metrics + chart stacked below)
// without producing zero-sized panels or an empty view.
func TestModel_AdaptiveLayout_Narrow(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 60, 18)

	if !m.isNarrow() {
		t.Fatalf("expected isNarrow()=true for width=60 (threshold=%d)", minNormalWidth)
	}
	if !m.isShort() {
		t.Fatalf("expected isShort()=true for height=18 (threshold=%d)", minNormalHeight)
	}

	// Logs panel must take the full terminal width in single-column mode.
	if m.logs.width != 60 {
		t.Errorf("expected logs width 60 (full width), got %d", m.logs.width)
	}
	// Metrics + chart must also span the full width.
	if m.metrics.width != 60 {
		t.Errorf("expected metrics width 60, got %d", m.metrics.width)
	}
	if m.chart.width != 60 {
		t.Errorf("expected chart width 60, got %d", m.chart.width)
	}

	// Compact metrics: metricsHeight is bounded by the right column's space
	// but must remain strictly positive (no division by zero downstream).
	if m.metrics.height <= 0 {
		t.Errorf("expected metrics height > 0, got %d", m.metrics.height)
	}
	if m.chart.height <= 0 {
		t.Errorf("expected chart height > 0, got %d", m.chart.height)
	}
	if m.logs.height <= 0 {
		t.Errorf("expected logs height > 0, got %d", m.logs.height)
	}

	// The compact mode must shrink the metrics base height.
	if m.baseMetricsHeight() != compactMetricsHeight {
		t.Errorf("expected baseMetricsHeight()=%d, got %d", compactMetricsHeight, m.baseMetricsHeight())
	}

	// The full body partition must fit inside the available body height
	// (no overlapping panels in the stacked layout).
	if m.logsHeight()+m.rightColumnHeight() > m.bodyHeight() {
		t.Errorf("logs+right=%d exceeds bodyHeight=%d", m.logsHeight()+m.rightColumnHeight(), m.bodyHeight())
	}

	// View() must produce a non-empty rendering.
	view := m.View()
	if view == "Initializing..." {
		t.Fatal("expected rendered view, got 'Initializing...'")
	}
	if !strings.Contains(view, "FibGo") {
		t.Error("expected narrow view to contain header 'FibGo'")
	}
	if !strings.Contains(view, "Heap") {
		t.Error("expected narrow view to contain 'Heap' from metrics panel")
	}
}

// TestModel_AdaptiveLayout_NormalUnchanged confirms that terminals at or
// above the thresholds keep the side-by-side layout with the historical
// 60/40 split.
func TestModel_AdaptiveLayout_NormalUnchanged(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 80, 24)

	if m.isNarrow() {
		t.Errorf("expected isNarrow()=false for width=80")
	}
	if m.isShort() {
		t.Errorf("expected isShort()=false for height=24")
	}
	if m.baseMetricsHeight() != MetricsPanelHeight {
		t.Errorf("expected baseMetricsHeight()=%d at normal size, got %d", MetricsPanelHeight, m.baseMetricsHeight())
	}
	if m.logs.width != 80*LogsPanelWidthPercent/100 {
		t.Errorf("expected logs width %d at normal size, got %d", 80*LogsPanelWidthPercent/100, m.logs.width)
	}
	if m.metrics.width != 80-(80*LogsPanelWidthPercent/100) {
		t.Errorf("expected metrics width %d at normal size, got %d", 80-(80*LogsPanelWidthPercent/100), m.metrics.width)
	}
}

// TestModel_AdaptiveLayout_ShortOnly verifies the compact-metrics branch
// when only the height threshold is crossed.
func TestModel_AdaptiveLayout_ShortOnly(t *testing.T) {
	t.Parallel()
	m := newTestModelWithSize(t, 120, 18)

	if m.isNarrow() {
		t.Errorf("expected isNarrow()=false for width=120")
	}
	if !m.isShort() {
		t.Errorf("expected isShort()=true for height=18")
	}
	if m.baseMetricsHeight() != compactMetricsHeight {
		t.Errorf("expected compact metrics base height %d, got %d", compactMetricsHeight, m.baseMetricsHeight())
	}
	// Side-by-side layout preserved.
	if m.logs.width != 120*LogsPanelWidthPercent/100 {
		t.Errorf("expected side-by-side logs width %d, got %d", 120*LogsPanelWidthPercent/100, m.logs.width)
	}
	if m.metrics.height <= 0 || m.chart.height <= 0 {
		t.Errorf("expected positive panel heights, got metrics=%d chart=%d", m.metrics.height, m.chart.height)
	}
	view := m.View()
	if len(view) == 0 {
		t.Error("expected non-empty view for short terminal")
	}
}

// TestLayoutManager_NoNegativeDimensions exercises a sweep of pathological
// sizes (including 0x0 and 1x1) to make sure no calculation returns a
// negative or zero dimension that would crash downstream renderers.
func TestLayoutManager_NoNegativeDimensions(t *testing.T) {
	t.Parallel()
	cases := []struct{ w, h int }{
		{0, 0},
		{1, 1},
		{40, 10},
		{60, 18},
		{79, 19},
		{80, 20},
		{120, 40},
		{500, 200},
	}
	for _, c := range cases {
		l := LayoutManager{width: c.w, height: c.h}
		if l.bodyHeight() < minBodyHeight {
			t.Errorf("[%dx%d] bodyHeight=%d < min=%d", c.w, c.h, l.bodyHeight(), minBodyHeight)
		}
		if l.logsWidth() < 0 {
			t.Errorf("[%dx%d] logsWidth=%d", c.w, c.h, l.logsWidth())
		}
		if l.rightWidth() < 0 {
			t.Errorf("[%dx%d] rightWidth=%d", c.w, c.h, l.rightWidth())
		}
		if l.metricsHeight() < 1 {
			t.Errorf("[%dx%d] metricsHeight=%d < 1", c.w, c.h, l.metricsHeight())
		}
		if l.chartHeight() < 1 {
			t.Errorf("[%dx%d] chartHeight=%d < 1", c.w, c.h, l.chartHeight())
		}
		if l.logsHeight() < 1 {
			t.Errorf("[%dx%d] logsHeight=%d < 1", c.w, c.h, l.logsHeight())
		}
	}
}

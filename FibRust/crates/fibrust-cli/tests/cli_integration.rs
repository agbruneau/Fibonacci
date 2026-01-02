//! Integration tests for the FibRust CLI binary.
//!
//! These tests verify the CLI behavior by running the actual binary
//! and checking its output and exit codes.

use assert_cmd::Command;
use predicates::prelude::*;

/// Returns a Command configured to run the fibrust CLI binary.
fn fibrust_cmd() -> Command {
    Command::cargo_bin("fibrust").unwrap()
}

// ============================================================================
// Basic Calculation Tests
// ============================================================================

#[test]
fn cli_calculates_fibonacci_10() {
    fibrust_cmd()
        .arg("10")
        .assert()
        .success()
        .stdout(predicate::str::contains("F(10)"));
}

#[test]
fn cli_calculates_fibonacci_0() {
    fibrust_cmd()
        .arg("0")
        .assert()
        .success()
        .stdout(predicate::str::contains("F(0)"));
}

#[test]
fn cli_calculates_fibonacci_1() {
    fibrust_cmd()
        .arg("1")
        .assert()
        .success()
        .stdout(predicate::str::contains("F(1)"));
}

// ============================================================================
// Named Argument Tests
// ============================================================================

#[test]
fn cli_accepts_named_argument_n() {
    fibrust_cmd()
        .args(["--n", "20"])
        .assert()
        .success()
        .stdout(predicate::str::contains("F(20)"));
}

// ============================================================================
// Algorithm Selection Tests
// ============================================================================

#[test]
fn cli_fast_doubling_algorithm() {
    fibrust_cmd()
        .args(["100", "-a", "fast-doubling"])
        .assert()
        .success()
        .stdout(predicate::str::contains("Fast Doubling"));
}

#[test]
fn cli_parallel_algorithm() {
    fibrust_cmd()
        .args(["100", "-a", "parallel"])
        .assert()
        .success()
        .stdout(predicate::str::contains("Parallel"));
}

#[test]
fn cli_fft_algorithm() {
    fibrust_cmd()
        .args(["100", "-a", "fft"])
        .assert()
        .success()
        .stdout(predicate::str::contains("FFT"));
}

#[test]
fn cli_adaptive_algorithm() {
    fibrust_cmd()
        .args(["100", "-a", "adaptive"])
        .assert()
        .success()
        .stdout(predicate::str::contains("Adaptive"));
}

#[test]
fn cli_all_algorithms_comparison() {
    fibrust_cmd()
        .args(["1000", "-a", "all"])
        .assert()
        .success()
        .stdout(predicate::str::contains("Comparison Summary"))
        .stdout(predicate::str::contains("Success"));
}

// ============================================================================
// Detail Flag Tests
// ============================================================================

#[test]
fn cli_detail_flag_shows_analysis() {
    fibrust_cmd()
        .args(["1000", "--detail"])
        .assert()
        .success()
        .stdout(predicate::str::contains("Detailed result analysis"))
        .stdout(predicate::str::contains("Number of digits"))
        .stdout(predicate::str::contains("209")); // F(1000) has 209 digits
}

#[test]
fn cli_detail_short_flag() {
    fibrust_cmd()
        .args(["100", "-d"])
        .assert()
        .success()
        .stdout(predicate::str::contains("Detailed result analysis"));
}

// ============================================================================
// Range Subcommand Tests
// ============================================================================

#[test]
fn cli_range_subcommand() {
    fibrust_cmd()
        .args(["range", "10", "15"])
        .assert()
        .success()
        .stdout(predicate::str::contains("Calculating Range F(10)..F(15)"))
        .stdout(predicate::str::contains("Generated 5 numbers"));
}

#[test]
fn cli_range_single_element() {
    fibrust_cmd()
        .args(["range", "100", "101"])
        .assert()
        .success()
        .stdout(predicate::str::contains("Generated 1 numbers"));
}

#[test]
fn cli_range_empty() {
    fibrust_cmd()
        .args(["range", "100", "100"])
        .assert()
        .success()
        .stdout(predicate::str::contains("Generated 0 numbers"));
}

// ============================================================================
// Help and Version Tests
// ============================================================================

#[test]
fn cli_help_displays() {
    fibrust_cmd()
        .arg("--help")
        .assert()
        .success()
        .stdout(predicate::str::contains(
            "High-performance Fibonacci calculator",
        ))
        .stdout(predicate::str::contains("--algorithm"))
        .stdout(predicate::str::contains("--detail"));
}

#[test]
fn cli_version_displays() {
    fibrust_cmd()
        .arg("--version")
        .assert()
        .success()
        .stdout(predicate::str::contains("fibrust"));
}

// ============================================================================
// Error Handling Tests
// ============================================================================

#[test]
fn cli_invalid_algorithm_shows_error() {
    fibrust_cmd()
        .args(["100", "-a", "invalid_algo"])
        .assert()
        .failure()
        .stderr(predicate::str::contains("error"));
}

// ============================================================================
// Sequential Mode Tests
// ============================================================================

#[test]
fn cli_sequential_flag() {
    fibrust_cmd().args(["1000", "--seq"]).assert().success();
}

#[test]
fn cli_sequential_short_flag() {
    fibrust_cmd().args(["1000", "-s"]).assert().success();
}

// ============================================================================
// Output Consistency Tests
// ============================================================================

#[test]
fn cli_shows_execution_configuration() {
    fibrust_cmd()
        .arg("100")
        .assert()
        .success()
        .stdout(predicate::str::contains("Execution Configuration"))
        .stdout(predicate::str::contains("FibRust"))
        .stdout(predicate::str::contains("logical processors"));
}

#[test]
fn cli_shows_result_binary_size() {
    fibrust_cmd()
        .arg("1000")
        .assert()
        .success()
        .stdout(predicate::str::contains("Result binary size"))
        .stdout(predicate::str::contains("bits"));
}

//! FibRust CLI - High-performance Fibonacci calculator.
//!
//! A command-line interface for calculating Fibonacci numbers using the `fibrust-core` library.
//! Supports single number calculation, range generation, and detailed performance analysis.

use clap::{Parser, Subcommand, ValueEnum};
use fibrust_core::{
    fib_range_parallel, fibonacci_adaptive, fibonacci_fast_doubling, fibonacci_fft,
    fibonacci_parallel, run_all_parallel,
};
use ibig::UBig;
use indicatif::{ProgressBar, ProgressStyle};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

const VERSION: &str = env!("CARGO_PKG_VERSION");

/// Calculation algorithm selection.
#[derive(Clone, Copy, PartialEq, ValueEnum)]
enum Algorithm {
    /// Adaptive: Auto-selects best algorithm (Fast Doubling for $n < 1M$, FFT for larger).
    Adaptive,
    /// Fast Doubling: $O(\log n)$ sequential algorithm.
    FastDoubling,
    /// Parallel: Parallelized Fast Doubling.
    Parallel,
    /// FFT: FFT-based multiplication.
    Fft,
    /// All: Runs all algorithms and compares performance (benchmarking mode).
    All,
}

/// CLI arguments structure.
#[derive(Parser)]
#[command(name = "fibrust", version, about = "High-performance Fibonacci calculator", long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,

    /// Calculate $F(n)$ (Positional argument).
    #[arg(conflicts_with = "n")]
    number: Option<u64>,

    /// Calculate $F(n)$ using `--n`.
    #[arg(long, conflicts_with = "number")]
    n: Option<u64>,

    /// Algorithm to use for single calculation.
    #[arg(short, long, value_enum, default_value_t = Algorithm::Adaptive)]
    algorithm: Algorithm,

    /// Show detailed result analysis (digits, scientific notation).
    #[arg(short, long)]
    detail: bool,

    /// Run sequentially (disable parallelism where applicable).
    #[arg(short, long)]
    seq: bool,
}

/// Available subcommands.
#[derive(Subcommand)]
enum Commands {
    /// Calculate a range of Fibonacci numbers $F(\text{start}) \dots F(\text{end})$ in parallel.
    Range {
        /// Start index (inclusive).
        start: u64,
        /// End index (exclusive).
        end: u64,
        /// Chunk size for parallel range (default: 0 = auto).
        #[arg(long, default_value_t = 0)]
        chunk_size: usize,
    },
}

struct AlgorithmResult {
    name: String,
    duration: Duration,
    result: UBig,
}

fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();

    // Pre-warm the system for consistent performance
    fibrust_core::prewarm_system();

    let num_cpus = std::thread::available_parallelism()
        .map(|p| p.get())
        .unwrap_or(1);

    println!("--- Execution Configuration ---");
    println!("FibRust v{}", VERSION);
    println!("Environment: {} logical processors.", num_cpus);

    // Determine what to do based on flags and subcommands
    if let Some(command) = &cli.command {
        match command {
            Commands::Range {
                start,
                end,
                chunk_size,
            } => {
                println!("Calculating Range F({})..F({})", start, end);
                if *chunk_size == 0 {
                    println!("Strategy: Parallel chunks (size: Auto)");
                } else {
                    println!("Strategy: Parallel chunks (size: {})", chunk_size);
                }

                let start_time = Instant::now();
                let results = fib_range_parallel(*start, *end, *chunk_size);
                let duration = start_time.elapsed();

                println!();
                println!("--- Execution Complete ---");
                println!(
                    "Generated {} numbers in {}.",
                    results.len(),
                    format_duration(duration)
                );

                if !results.is_empty() {
                    println!("First: F({}) = {}...", start, format_preview(&results[0]));
                    println!(
                        "Last : F({}) = {}...",
                        end - 1,
                        format_preview(results.last().unwrap())
                    );
                }
            }
        }
    } else {
        // Handle Single Calculation (Positional OR --n)
        let n = if let Some(n) = cli.n {
            n
        } else if let Some(number) = cli.number {
            number
        } else {
            // No argument provided, show help
            use clap::CommandFactory;
            Cli::command().print_help()?;
            return Ok(());
        };

        run_single_calculation(n, cli.algorithm, cli.detail, !cli.seq);
    }

    Ok(())
}

/// Executes a single Fibonacci calculation and displays the result.
///
/// Handles the selection of algorithms, parallel/sequential execution,
/// and optional result formatting/analysis.
///
/// # Arguments
///
/// * `n` - The index of the Fibonacci number to compute.
/// * `algorithm` - The selected algorithm strategy.
/// * `show_preview` - Whether to show detailed analysis (digits, scientific notation).
/// * `parallel` - Whether to enable parallel execution (where applicable).
fn run_single_calculation(n: u64, algorithm: Algorithm, show_preview: bool, parallel: bool) {
    println!("Calculating F({})", n);
    println!("Optimization: Parallelism=50k bits, FFT=50k bits.");

    let mode_str = match (algorithm, parallel) {
        (Algorithm::Adaptive, _) => "Adaptive (auto-selects Fast Doubling or FFT).",
        (Algorithm::All, true) => "Parallel comparison of all algorithms.",
        (Algorithm::All, false) => "Sequential comparison of all algorithms.",
        (Algorithm::FastDoubling, _) => "Fast Doubling only.",
        (Algorithm::Parallel, _) => "Parallel Fast Doubling only.",
        (Algorithm::Fft, _) => "FFT-Based Doubling only.",
    };
    println!("Mode: {}", mode_str);
    println!();
    println!("--- Starting Execution ---");

    let mut results: Vec<AlgorithmResult> = Vec::new();

    // Progress bar setup
    let pb = ProgressBar::new(100);
    pb.set_style(
        ProgressStyle::default_bar()
            .template("Avg progress: {percent:>6.2}% [{bar:40.green/dim}] ETA: {eta}")
            .unwrap()
            .progress_chars("████"),
    );

    let progress = Arc::new(AtomicU64::new(0));
    let progress_clone = progress.clone();
    let pb_clone = pb.clone();

    let progress_handle = std::thread::spawn(move || {
        let start = Instant::now();
        loop {
            let current = progress_clone.load(Ordering::Relaxed);
            if current >= 100 {
                pb_clone.set_position(100);
                break;
            }
            let elapsed_ms = start.elapsed().as_millis() as u64;
            let estimated = (elapsed_ms.min(10000) * 99 / 10000).max(current);
            pb_clone.set_position(estimated);
            std::thread::sleep(Duration::from_millis(30));
        }
    });

    if algorithm == Algorithm::All && parallel {
        let parallel_results = run_all_parallel(n);
        for (name, duration, result) in parallel_results {
            results.push(AlgorithmResult {
                name,
                duration,
                result,
            });
        }
    } else {
        if algorithm == Algorithm::Adaptive {
            let start = Instant::now();
            let result = fibonacci_adaptive(n);
            let duration = start.elapsed();
            let algo_used = if n < 1_000_000 {
                "Fast Doubling"
            } else {
                "FFT"
            };
            results.push(AlgorithmResult {
                name: format!("Adaptive (using {})", algo_used),
                duration,
                result,
            });
        }

        if algorithm == Algorithm::All || algorithm == Algorithm::FastDoubling {
            let start = Instant::now();
            let result = fibonacci_fast_doubling(n);
            let duration = start.elapsed();
            results.push(AlgorithmResult {
                name: "Fast Doubling (O(log n), Parallel, Zero-Alloc)".to_string(),
                duration,
                result,
            });
        }

        if algorithm == Algorithm::All || algorithm == Algorithm::Parallel {
            let start = Instant::now();
            let result = fibonacci_parallel(n);
            let duration = start.elapsed();
            results.push(AlgorithmResult {
                name: "Parallel Fast Doubling (O(log n), Multicore)".to_string(),
                duration,
                result,
            });
        }

        if algorithm == Algorithm::All || algorithm == Algorithm::Fft {
            let start = Instant::now();
            let result = fibonacci_fft(n);
            let duration = start.elapsed();
            results.push(AlgorithmResult {
                name: "FFT".to_string(),
                duration,
                result,
            });
        }
    }

    progress.store(100, Ordering::Relaxed);
    let _ = progress_handle.join();
    pb.finish_and_clear();
    println!("Avg progress: 100.00% [████████████████████████████████████████] ETA: < 1s");
    println!();

    results.sort_by(|a, b| a.duration.cmp(&b.duration));

    let consistent = if results.len() > 1 {
        results.windows(2).all(|w| w[0].result == w[1].result)
    } else {
        true
    };

    println!("--- Comparison Summary ---");
    println!("{:<55} {:>10}   Status", "Algorithm", "Duration");
    for res in &results {
        println!(
            "{:<55} {:>10}   ✅ Success",
            res.name,
            format_duration(res.duration)
        );
    }
    println!();

    let first_result = &results[0].result;
    let bits = first_result.bit_len();

    if results.len() > 1 {
        if consistent {
            println!("Global Status: Success. All valid results are consistent.");
        } else {
            println!("Global Status: WARNING. Results differ!");
        }
    } else {
        println!("Global Status: Success.");
    }
    println!("Result binary size: {:} bits.", format_number(bits));

    if show_preview {
        let fastest_duration = results[0].duration;
        let result_str = first_result.to_string();
        let digits = result_str.len();

        let scientific = if digits > 1 {
            format!(
                "{}.{}e+{}",
                &result_str[..1],
                &result_str[1..7.min(digits)],
                digits - 1
            )
        } else {
            result_str.clone()
        };

        println!();
        println!("--- Detailed result analysis ---");
        println!(
            "Calculation time     : {}",
            format_duration(fastest_duration)
        );
        println!("Number of digits     : {}", format_number(digits));
        println!("Scientific notation  : {}", scientific);
    }
}

/// Formats a UBig for preview display.
///
/// Truncates very long numbers to show the first 10 digits and the total length.
fn format_preview(n: &UBig) -> String {
    let s = n.to_string();
    if s.len() > 20 {
        format!("{}..({})", &s[..10], s.len())
    } else {
        s
    }
}

/// Formats a duration into a human-readable string (ms or s).
fn format_duration(duration: Duration) -> String {
    let millis = duration.as_millis();
    if millis < 1 {
        let micros = duration.as_micros();
        format!("{:.2}ms", micros as f64 / 1000.0)
    } else if millis < 1000 {
        format!("{}ms", millis)
    } else {
        format!("{:.2}s", duration.as_secs_f64())
    }
}

/// Formats a large number with comma separators for readability (e.g., "1,000,000").
fn format_number(n: usize) -> String {
    let s = n.to_string();
    let mut result = String::new();
    for (i, c) in s.chars().rev().enumerate() {
        if i > 0 && i % 3 == 0 {
            result.push(',');
        }
        result.push(c);
    }
    result.chars().rev().collect()
}

//! Criterion benchmarks for FibRust algorithms.
//!
//! Run with: `cargo bench`
//! View HTML reports in: `target/criterion/report/index.html`

use criterion::{black_box, criterion_group, criterion_main, BenchmarkId, Criterion, Throughput};
use fibrust_core::{
    fibonacci_fast_doubling, fibonacci_fft, fibonacci_parallel, FibNumber, FibRange,
};

/// Naive iterative implementation O(n) for comparison purposes.
fn fibonacci_naive(n: u64) -> FibNumber {
    if n == 0 {
        return FibNumber::from(0u32);
    }
    if n == 1 {
        return FibNumber::from(1u32);
    }

    let mut a = FibNumber::from(0u32);
    let mut b = FibNumber::from(1u32);

    for _ in 2..=n {
        let temp = &a + &b;
        a = b;
        b = temp;
    }
    b
}

/// Benchmark comparing Naive vs Fast Doubling (proof of O(n) vs O(log n)).
fn naive_vs_fast_comparison(c: &mut Criterion) {
    let mut group = c.benchmark_group("naive_vs_fast");
    // Limit sample size for naive as it gets very slow
    group.sample_size(10);

    for n in [100u64, 1_000, 5_000, 10_000, 20_000] {
        group.throughput(Throughput::Elements(1));

        group.bench_with_input(BenchmarkId::new("naive", n), &n, |b, &n| {
            b.iter(|| fibonacci_naive(black_box(n)))
        });

        group.bench_with_input(BenchmarkId::new("fast_doubling", n), &n, |b, &n| {
            b.iter(|| fibonacci_fast_doubling(black_box(n)))
        });
    }

    group.finish();
}

/// Benchmark comparing all three algorithms at various input sizes.
fn algorithm_comparison(c: &mut Criterion) {
    let mut group = c.benchmark_group("algorithm_comparison");

    // Test at different scales
    for n in [1_000u64, 10_000, 50_000, 100_000, 500_000, 1_000_000] {
        group.throughput(Throughput::Elements(1));

        group.bench_with_input(BenchmarkId::new("fast_doubling", n), &n, |b, &n| {
            b.iter(|| fibonacci_fast_doubling(black_box(n)))
        });

        group.bench_with_input(BenchmarkId::new("parallel", n), &n, |b, &n| {
            b.iter(|| fibonacci_parallel(black_box(n)))
        });

        // FFT only for larger values where it shines
        if n >= 50_000 {
            group.bench_with_input(BenchmarkId::new("fft", n), &n, |b, &n| {
                b.iter(|| fibonacci_fft(black_box(n)))
            });
        }
    }

    group.finish();
}

/// Benchmark Fast Doubling scaling behavior.
fn fast_doubling_scaling(c: &mut Criterion) {
    let mut group = c.benchmark_group("fast_doubling_scaling");
    group.sample_size(50);

    for exp in 3..=7 {
        let n = 10u64.pow(exp);
        group.throughput(Throughput::Elements(1));

        group.bench_with_input(BenchmarkId::from_parameter(n), &n, |b, &n| {
            b.iter(|| fibonacci_fast_doubling(black_box(n)))
        });
    }

    group.finish();
}

/// Benchmark FFT scaling for large inputs.
fn fft_scaling(c: &mut Criterion) {
    let mut group = c.benchmark_group("fft_scaling");
    group.sample_size(20); // Fewer samples for slow benchmarks

    for n in [100_000u64, 500_000, 1_000_000, 5_000_000, 10_000_000] {
        group.throughput(Throughput::Elements(1));

        group.bench_with_input(BenchmarkId::from_parameter(n), &n, |b, &n| {
            b.iter(|| fibonacci_fft(black_box(n)))
        });
    }

    group.finish();
}

/// Benchmark FibRange iterator performance.
fn iterator_benchmark(c: &mut Criterion) {
    let mut group = c.benchmark_group("fib_range_iterator");

    // Benchmark range iteration (start from different positions)
    for (start, count) in [(0u64, 100), (1000, 100), (10000, 100), (100000, 50)] {
        let id = format!("F({}..{})", start, start + count);
        group.throughput(Throughput::Elements(count));

        group.bench_function(&id, |b| {
            b.iter(|| {
                let _: Vec<_> = FibRange::new(black_box(start), black_box(start + count)).collect();
            })
        });
    }

    group.finish();
}

/// Benchmark small inputs (where u128 optimization kicks in).
fn small_input_benchmark(c: &mut Criterion) {
    let mut group = c.benchmark_group("small_inputs");

    for n in [10u64, 50, 100, 150, 186, 200, 500] {
        group.bench_with_input(BenchmarkId::from_parameter(n), &n, |b, &n| {
            b.iter(|| fibonacci_fast_doubling(black_box(n)))
        });
    }

    group.finish();
}

/// Benchmark scalability with number of cores.
fn scalability_benchmark(c: &mut Criterion) {
    let mut group = c.benchmark_group("scalability_cores");
    let n = 5_000_000u64; // Large enough to see parallel benefits
    group.sample_size(10);
    group.throughput(Throughput::Elements(1));

    // Test with different thread counts
    for threads in [1, 2, 4, 8, 16] {
        // We can't easily change the global rayon thread pool at runtime once initialized,
        // so we use a custom thread pool for each iteration.
        group.bench_with_input(BenchmarkId::new("threads", threads), &threads, |b, &t| {
            b.iter_custom(|iters| {
                let pool = rayon::ThreadPoolBuilder::new()
                    .num_threads(t)
                    .build()
                    .unwrap();

                let start = std::time::Instant::now();
                for _ in 0..iters {
                    pool.install(|| {
                        fibonacci_parallel(black_box(n));
                    });
                }
                start.elapsed()
            })
        });
    }

    group.finish();
}

criterion_group!(
    benches,
    algorithm_comparison,
    fast_doubling_scaling,
    fft_scaling,
    iterator_benchmark,
    small_input_benchmark,
    naive_vs_fast_comparison,
    scalability_benchmark,
);
criterion_main!(benches);

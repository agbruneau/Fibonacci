use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::Instant;
use std::cmp;
use num_bigint::BigInt;
use num_traits::{One, Zero};

const MAX_FIB_VALUE: usize = 100_000_001; // Maximum value of n that can be calculated

lazy_static::lazy_static! {
    // MEMO is a thread-safe, shared cache for storing previously computed Fibonacci values.
    static ref MEMO: Arc<Mutex<HashMap<usize, BigInt>>> = Arc::new(Mutex::new(HashMap::new()));
}

// fib_doubling calculates the nth Fibonacci number using the doubling method
fn fib_doubling(n: usize) -> Result<BigInt, String> {
    // If n is less than 2, return the value directly (base cases: F(0) = 0, F(1) = 1).
    if n < 2 {
        return Ok(BigInt::from(n));
    } else if n > MAX_FIB_VALUE {
        // If n is too large, return an error indicating the limit.
        return Err("n is too large for this implementation".to_string());
    }
    // Compute the Fibonacci value using an iterative helper function.
    let result = fib_doubling_helper_iterative(n);
    Ok(result)
}

// fib_doubling_helper_iterative is an iterative function that uses the doubling method to compute Fibonacci numbers
fn fib_doubling_helper_iterative(n: usize) -> BigInt {
    {
        // Attempt to acquire the lock on MEMO to check if the value has already been calculated.
        let memo = MEMO.lock().expect("Failed to acquire lock on MEMO");
        if let Some(val) = memo.get(&n) {
            // If the value is found in the cache, return it.
            return val.clone();
        }
    } // Release the lock as early as possible to minimize the critical section.

    // Initialize base Fibonacci values: F(0) = 0 and F(1) = 1.
    let mut a = BigInt::zero();
    let mut b = BigInt::one();

    // Determine the number of bits needed to represent n.
    let bit_length = 64 - n.leading_zeros() as usize;

    // Iterate over each bit from the most significant to the least significant.
    for i in (0..bit_length).rev() {
        // Use the doubling formulas to calculate intermediate Fibonacci values.
        // F(2k) = F(k) * [2 * F(k+1) - F(k)]
        let mut c = &b * 2u32 - &a;
        c = &a * &c;
        // F(2k + 1) = F(k)^2 + F(k+1)^2
        let d = &a * &a + &b * &b;

        // Update a and b based on the current bit of n.
        if (n >> i) & 1 == 0 {
            // If the bit is 0, set F(2k) to a and F(2k + 1) to b.
            a = c.clone();
            b = d.clone();
        } else {
            // If the bit is 1, set F(2k + 1) to a and F(2k + 2) to b.
            a = d.clone();
            b = c + d;
        }
    }

    // Cache the result for future use.
    let result = a.clone();
    let mut memo = MEMO.lock().expect("Failed to acquire lock on MEMO");
    memo.insert(n, result.clone());
    result
}

// print_error prints an error message in a consistent format
fn print_error(n: usize, err: &str) {
    println!("fib_doubling({}): {}", n, err);
}

// benchmark_fib benchmarks the Fibonacci calculations for a list of values
fn benchmark_fib(n_values: &[usize], repetitions: usize) {
    // Clear the memoization cache before benchmarking to ensure consistent results.
    let mut memo = MEMO.lock().expect("Failed to acquire lock on MEMO");
    memo.clear();
    drop(memo);

    for &n in n_values {
        let mut accumulated_exec_time = 0u128;
        for _ in 0..repetitions {
            // Start the timer before calculating the Fibonacci number.
            let start = Instant::now();
            match fib_doubling(n) {
                Ok(_) => {
                    // Record the time taken for the calculation.
                    let elapsed = start.elapsed().as_nanos();
                    accumulated_exec_time += elapsed;
                }
                Err(err) => {
                    // Print an error message if the calculation fails.
                    print_error(n, &err);
                    continue;
                }
            }
        }
        // Calculate the average execution time over the number of repetitions.
        let exec_time = accumulated_exec_time / cmp::max(1, repetitions as u128);
        println!("fib_doubling({}) averaged over {} runs: {} nanoseconds", n, repetitions, exec_time);
    }
}

fn main() {
    // Define the list of values for which to benchmark the Fibonacci calculation.
    let n_values = vec![1_000_, 10_000, 10_000_000]; // List of values to benchmark
    let repetitions = 3; // Number of repetitions for better accuracy
    benchmark_fib(&n_values, repetitions);
}

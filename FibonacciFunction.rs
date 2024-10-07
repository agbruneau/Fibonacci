use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::Instant;
use std::cmp;
use num_bigint::BigInt;
use num_traits::{One, Zero};

const MAX_FIB_VALUE: usize = 100_000_001; // Maximum value of n that can be calculated
lazy_static::lazy_static! {
    static ref MEMO: Arc<Mutex<HashMap<usize, BigInt>>> = Arc::new(Mutex::new(HashMap::new()));
}

// fib_doubling calculates the nth Fibonacci number using the doubling method
fn fib_doubling(n: usize) -> Result<BigInt, String> {
    if n < 2 {
        return Ok(BigInt::from(n));
    } else if n > MAX_FIB_VALUE {
        return Err("n is too large for this implementation".to_string());
    }
    let result = fib_doubling_helper_iterative(n);
    Ok(result)
}

// fib_doubling_helper_iterative is an iterative function that uses the doubling method to compute Fibonacci numbers
fn fib_doubling_helper_iterative(n: usize) -> BigInt {
    let memo = MEMO.lock().unwrap();
    if let Some(val) = memo.get(&n) {
        return val.clone();
    }
    drop(memo);

    let mut a = BigInt::zero();
    let mut b = BigInt::one();

    let bit_length = 64 - n.leading_zeros() as usize;

    for i in (0..bit_length).rev() {
        let mut c = &b * 2u32 - &a;
        c = &a * &c;
        let d = &a * &a + &b * &b;

        if (n >> i) & 1 == 0 {
            a = c.clone();
            b = d.clone();
        } else {
            a = d.clone();
            b = c + d;
        }
    }

    let result = a.clone();
    let mut memo = MEMO.lock().unwrap();
    memo.insert(n, result.clone());
    result
}

// print_error prints an error message in a consistent format
fn print_error(n: usize, err: &str) {
    println!("fib_doubling({}): {}", n, err);
}

// benchmark_fib benchmarks the Fibonacci calculations for a list of values
fn benchmark_fib(n_values: &[usize], repetitions: usize) {
    let mut memo = MEMO.lock().unwrap();
    memo.clear();
    drop(memo);

    for &n in n_values {
        let mut total_exec_time = 0u128;
        for _ in 0..repetitions {
            let start = Instant::now();
            match fib_doubling(n) {
                Ok(_) => {
                    let elapsed = start.elapsed().as_nanos();
                    total_exec_time += elapsed;
                }
                Err(err) => {
                    print_error(n, &err);
                    continue;
                }
            }
        }
        let exec_time = total_exec_time / cmp::max(1, repetitions as u128);
        println!("fib_doubling({}) averaged over {} runs: {} nanoseconds", n, repetitions, exec_time);
    }
}

fn main() {
    let n_values = vec![1_000_, 10_000, 10_000_000];
    let repetitions = 3;
    benchmark_fib(&n_values, repetitions);
}

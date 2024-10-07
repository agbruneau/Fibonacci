import time
from functools import lru_cache
from typing import Optional

MAX_FIB_VALUE = 100000001  # Maximum value of n that can be calculated

# Memoization cache to store calculated Fibonacci values using LRU cache
@lru_cache(maxsize=None)
def fib_doubling(n: int) -> Optional[int]:
    # fib_doubling calculates the nth Fibonacci number using the doubling method
    if n < 2:
        return n  # Base cases: F(0) = 0, F(1) = 1
    elif n > MAX_FIB_VALUE:
        # Error if the value is too large to calculate within reasonable time
        raise ValueError(f"n is too large for this implementation. Please use a value less than or equal to {MAX_FIB_VALUE}.")
    # Calculate the Fibonacci value using an iterative helper function
    result = fib_doubling_helper_iterative(n)
    return result

# fib_doubling_helper_iterative is an iterative function that uses the doubling method to compute Fibonacci numbers
def fib_doubling_helper_iterative(n: int) -> int:
    # Initialize base Fibonacci values F(0) = 0 and F(1) = 1
    a, b = 0, 1

    # Determine the number of bits needed to represent n
    bit_length = n.bit_length()  # This helps in iterating efficiently over the binary representation of n

    # Iterate over each bit from the most significant to the least significant
    for i in reversed(range(bit_length)):
        # Use the doubling formulas to calculate F(2k) and F(2k + 1)
        # F(2k) = F(k) * [2 * F(k+1) - F(k)]
        c = a * ((b << 1) - a)  # Calculate F(2k) using F(k) and F(k+1)
        # F(2k + 1) = F(k)^2 + F(k+1)^2
        d = a * a + b * b  # Calculate F(2k + 1) using F(k) and F(k+1)

        # Update a and b based on the current bit of n
        if (n >> i) & 1 == 0:
            a, b = c, d  # If the bit is 0, set F(2k) to a and F(2k+1) to b
        else:
            a, b = d, c + d  # If the bit is 1, set F(2k+1) to a and F(2k+2) to b

    return a  # Return the nth Fibonacci number

# benchmark_fib benchmarks the Fibonacci calculations for a list of values
def benchmark_fib(n_values: list[int], repetitions: int):
    # Note: Clearing the cache ensures consistent results but may affect performance for repeated calculations.
    # Consider if this step is necessary based on your use case.

    for n in n_values:
        total_exec_time = 0
        min_exec_time = float('inf')  # Initialize min execution time to a very high value
        max_exec_time = float('-inf')  # Initialize max execution time to a very low value
        # Repeat the calculation for more accurate benchmarking
        for _ in range(repetitions):
            start = time.perf_counter()  # Start timing the calculation
            try:
                fib_doubling(n)  # Calculate the nth Fibonacci number
            except ValueError as err:
                # Handle the error if n is too large
                print(f"fib_doubling({n}): {err}")
                continue
            exec_time = (time.perf_counter() - start) * 1e9  # Convert seconds to nanoseconds
            total_exec_time += exec_time
            min_exec_time = min(min_exec_time, exec_time)  # Update min execution time if current is lower
            max_exec_time = max(max_exec_time, exec_time)  # Update max execution time if current is higher
        # Calculate the average execution time
        avg_exec_time = total_exec_time / repetitions
        # Print detailed benchmarking results including min, max, and average execution times
        print(f"fib_doubling({n}) averaged over {repetitions} runs: {avg_exec_time:.0f} nanoseconds (min: {min_exec_time:.0f} ns, max: {max_exec_time:.0f} ns)")

# main function to execute the benchmarking
def main():
    n_values = [1000000, 10000000, 100000000]  # List of values to benchmark
    repetitions = 3  # Number of repetitions for better accuracy
    benchmark_fib(n_values, repetitions)  # Run the benchmark

if __name__ == "__main__":
    main()
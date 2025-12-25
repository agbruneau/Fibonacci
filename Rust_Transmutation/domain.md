# Domain Dictionary (Ubiquitous Language)

This document establishes the ubiquitous language for the **Rust Transmutation** project. It acts as a strict glossary to ensure consistency between the existing Go implementation and the new Rust port.

## Core Entities

### Fibonacci Number
*   **Definition**: The mathematical result $F(n)$ of the Fibonacci sequence.
*   **Context**: The primary output of the application.
*   **Type**: Arbitrary-precision integer.
*   **Code Rule**:
    *   Variable name: `result` or `fib_n`.
    *   Type: `BigInt` (from `num-bigint` crate).
    *   Do **not** use `u64` or `u128` for the result, as it overflows for $n > 93$.

### Index ($n$)
*   **Definition**: The position in the Fibonacci sequence to calculate.
*   **Context**: Input parameter.
*   **Type**: Unsigned 64-bit integer.
*   **Code Rule**:
    *   Variable name: `n`.
    *   Type: `u64`.
    *   Constraint: $n \ge 0$.

### Calculator
*   **Definition**: The component responsible for orchestrating the calculation of a Fibonacci number. It abstracts the underlying algorithm.
*   **Relations**:
    *   A `Calculator` implements a specific `Algorithm`.
    *   A `Calculator` accepts `Options`.
*   **Code Rule**:
    *   Trait name: `Calculator`.
    *   Method: `calculate(n: u64, opts: Options) -> Result<BigInt, Error>`.

## Algorithms

### Algorithm
*   **Definition**: A specific mathematical strategy used to compute $F(n)$.
*   **Values**:
    *   `FastDoubling`: The default, high-performance algorithm ($O(\log n)$).
    *   `MatrixExponentiation`: Computes via matrix powers ($O(\log n)$).
    *   `Iterative`: Simple addition loop ($O(n)$), used only for small $n$ (usually $n \le 93$).
*   **Code Rule**:
    *   Enum/Type names: `FastDoubling`, `MatrixExponentiation`.
    *   CLI Flag values: `fast`, `matrix`.

### Fast Doubling
*   **Definition**: An algorithm based on the identity $F(2n) = F(n)(2F(n+1) - F(n))$. It avoids full matrix multiplication overhead.
*   **Context**: The default and typically fastest algorithm for large $n$.
*   **Code Rule**: Struct name `FastDoublingCalculator`.

### Matrix Exponentiation
*   **Definition**: An algorithm that computes $\begin{pmatrix} 1 & 1 \\ 1 & 0 \end{pmatrix}^n$.
*   **Relations**: Can use **Strassen's Algorithm** for matrix multiplication if the matrix size exceeds the `StrassenThreshold`.
*   **Code Rule**: Struct name `MatrixCalculator`.

## Optimization & Configuration

### Threshold
*   **Definition**: A configuration value (usually in bits) that determines when the software switches execution strategies to optimize performance.
*   **Sub-types**:
    *   `ParallelThreshold`: Bit size above which multiplication is parallelized.
    *   `FFTThreshold`: Bit size above which multiplication uses Fast Fourier Transform (BigFFT).
    *   `StrassenThreshold`: Bit size above which Matrix Exponentiation uses Strassen's algorithm for matrix multiplication.
*   **Code Rule**:
    *   Field names: `parallel_threshold_bits`, `fft_threshold_bits`.
    *   Type: `usize` (as it relates to bit length).

### Calibration
*   **Definition**: The process of empirically benchmarking the system to determine the optimal `Threshold` values for the current hardware.
*   **Relations**: Produces a `Profile`.
*   **Code Rule**:
    *   Function: `auto_calibrate()`.
    *   CLI Flag: `--calibrate`.

### Profile
*   **Definition**: A set of saved `Threshold` values derived from `Calibration`.
*   **Code Rule**:
    *   Struct: `CalibrationProfile`.
    *   Persistence: JSON file (e.g., `fibcalc_profile.json`).

## Low-Level Operations

### BigInt
*   **Definition**: The fundamental data type for large integers.
*   **Context**: Wrapped from `num-bigint`.
*   **Code Rule**: Use `BigInt` or `BigUint` (if strictly positive) from `num-bigint`.

### BigFFT
*   **Definition**: Large integer multiplication using the Fast Fourier Transform. used when operands exceed `FFTThreshold`.
*   **Relations**: Used within `FastDoubling` or `MatrixExponentiation` for the underlying multiplication of huge numbers.
*   **Code Rule**: Module/Crate `bigfft`.

### Strassen
*   **Definition**: Strassen algorithm for matrix multiplication. Reduces complexity from $O(N^3)$ to $O(N^{\log_2 7})$.
*   **Context**: Only applies to `MatrixExponentiation` when elements are very large.
*   **Code Rule**: Function `multiply_matrix_strassen`.

## System & Interface

### Progress Reporter
*   **Definition**: A mechanism to report calculation progress to the user (CLI or API) without blocking the calculation.
*   **Code Rule**:
    *   Type: `ProgressSender` (channel based) or callback.
    *   Unit: Float `0.0` to `1.0`.

### Options
*   **Definition**: A structure containing all configuration parameters for a calculation request.
*   **Code Rule**:
    *   Struct: `CalculationOptions`.
    *   Fields: `parallel_threshold`, `fft_threshold`, etc.

### API Response
*   **Definition**: The standardized JSON structure returned by the REST API for a calculation request.
*   **Fields**: `n` (input), `result` (output), `duration`, `algorithm`, `error` (optional).
*   **Code Rule**:
    *   Struct: `CalculationResponse`.
    *   Serialize with `serde`.

### Error Response
*   **Definition**: The standardized JSON structure returned by the REST API when an error occurs.
*   **Fields**: `error` (code), `message` (description).
*   **Code Rule**:
    *   Struct: `ErrorResponse`.

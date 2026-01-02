use std::fmt::{Debug, Display};
use std::ops::{Add, AddAssign, Mul, MulAssign, Shl, ShlAssign, Sub, SubAssign};

/// Error type for Fibonacci calculations.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FibError {
    /// Input is too large for the current system configuration.
    InputTooLarge { n: u64, max: u64 },
    /// Memory limit exceeded (estimated).
    MemoryLimitExceeded {
        required_bytes: u64,
        limit_bytes: u64,
    },
}

impl Display for FibError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            FibError::InputTooLarge { n, max } => {
                write!(f, "Input n={} is too large (max supported: {})", n, max)
            }
            FibError::MemoryLimitExceeded {
                required_bytes,
                limit_bytes,
            } => write!(
                f,
                "Estimated memory requirement {} bytes exceeds limit {} bytes",
                required_bytes, limit_bytes
            ),
        }
    }
}

impl std::error::Error for FibError {}

// ============================================================================
// Algorithm Selection
// ============================================================================

/// Fibonacci algorithm selection.
///
/// This enum is shared between CLI and Server for consistent algorithm naming.
///
/// # Variants
///
/// - `FastDoubling`: Sequential O(log n) algorithm, optimal for small inputs.
/// - `Parallel`: Parallelized Fast Doubling, optimal for medium inputs.
/// - `Fft`: FFT-based multiplication, optimal for massive inputs.
/// - `Adaptive`: Auto-selects best algorithm based on input size.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Default)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum Algorithm {
    /// Fast Doubling: $O(\log n)$ sequential algorithm.
    ///
    /// Best for $n < 40,000$ where thread overhead exceeds parallelization gains.
    #[cfg_attr(feature = "serde", serde(rename = "fd", alias = "fast-doubling"))]
    FastDoubling,

    /// Parallel Fast Doubling: Parallelized multiplications.
    ///
    /// Best for $40,000 \le n < 200,000$ where multi-core advantage is significant.
    #[cfg_attr(
        feature = "serde",
        serde(rename = "par", alias = "parallel", alias = "mx")
    )]
    Parallel,

    /// FFT-based multiplication: $O(n \log n)$ for huge numbers.
    ///
    /// Best for $n \ge 200,000$ where FFT beats Karatsuba asymptotically.
    #[cfg_attr(feature = "serde", serde(rename = "fft"))]
    Fft,

    /// Adaptive: Automatically selects the best algorithm.
    ///
    /// Uses thresholds from `config::thresholds` to choose between
    /// FastDoubling, Parallel, and Fft based on input size.
    #[default]
    #[cfg_attr(feature = "serde", serde(rename = "adaptive"))]
    Adaptive,
}

impl std::fmt::Display for Algorithm {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Algorithm::FastDoubling => write!(f, "Fast Doubling"),
            Algorithm::Parallel => write!(f, "Parallel Fast Doubling"),
            Algorithm::Fft => write!(f, "FFT"),
            Algorithm::Adaptive => write!(f, "Adaptive"),
        }
    }
}

/// Trait defining the operations required for Fibonacci calculations.
/// This allows abstracting over `ibig::UBig` and `rug::Integer`.
pub trait FibOps:
    Sized
    + Clone
    + Debug
    + Display
    + PartialEq
    + PartialOrd
    + Add<Output = Self>
    + for<'a> Add<&'a Self, Output = Self>
    + AddAssign
    + for<'a> AddAssign<&'a Self>
    + Sub<Output = Self>
    + for<'a> Sub<&'a Self, Output = Self>
    + SubAssign
    + for<'a> SubAssign<&'a Self>
    + Mul<Output = Self>
    + for<'a> Mul<&'a Self, Output = Self>
    + MulAssign
    + for<'a> MulAssign<&'a Self>
    + Shl<usize, Output = Self>
    + ShlAssign<usize>
    + From<u32>
    + From<u64>
    + From<u128>
{
    /// Returns the number of bits required to represent the number.
    fn bit_len(&self) -> usize;

    /// Computes self^exp.
    fn pow(&self, exp: u32) -> Self;

    /// Returns true if the number is zero.
    fn is_zero(&self) -> bool {
        self.bit_len() == 0
    }

    /// Returns the number as little-endian bytes.
    fn to_le_bytes(&self) -> Vec<u8>;

    /// Creates a number from little-endian bytes.
    fn from_le_bytes(bytes: &[u8]) -> Self;
}

// ----------------------------------------------------------------------------
// IMPL: ibig::UBig
// ----------------------------------------------------------------------------
#[cfg(not(feature = "gmp"))]
pub type FibNumber = ibig::UBig;

#[cfg(not(feature = "gmp"))]
impl FibOps for ibig::UBig {
    #[inline]
    fn bit_len(&self) -> usize {
        self.bit_len()
    }

    #[inline]
    fn pow(&self, exp: u32) -> Self {
        ibig::UBig::pow(self, exp as usize)
    }

    #[inline]
    fn to_le_bytes(&self) -> Vec<u8> {
        ibig::UBig::to_le_bytes(self)
    }

    #[inline]
    fn from_le_bytes(bytes: &[u8]) -> Self {
        ibig::UBig::from_le_bytes(bytes)
    }
}

// ----------------------------------------------------------------------------
// IMPL: rug::Integer (GMP)
// ----------------------------------------------------------------------------
#[cfg(feature = "gmp")]
pub type FibNumber = rug::Integer;

#[cfg(feature = "gmp")]
impl FibOps for rug::Integer {
    #[inline]
    fn bit_len(&self) -> usize {
        self.significant_bits() as usize
    }

    #[inline]
    fn pow(&self, exp: u32) -> Self {
        // rug's pow takes u32 directly
        match self.clone().pow_u32(exp).into() {
            Ok(v) => v,
            // rug's pow_u32 logic with integer handles arbitrary size,
            // but strictly returns Self.
            // Wait, rug::Integer::pow_u32 consumes self.
            _ => unreachable!("rug pow failed"),
        }
    }

    #[inline]
    fn to_le_bytes(&self) -> Vec<u8> {
        // GMP stores in limbs, export to bytes
        // order: 1 for little endian (least significant byte first)
        // size: 1 for 1-byte elements
        // endian: 0 for native endian (not relevant for byte array) or 1?
        // Let's use to_digits generic.
        // Actually to_digits with order=lsb is standard.
        // rug::Integer::write_digits matches this.
        let mut bytes = Vec::new();
        // order: Order::Lsf (Least significant first)
        self.write_digits(&mut bytes, rug::integer::Order::Lsf);
        bytes
    }

    #[inline]
    fn from_le_bytes(bytes: &[u8]) -> Self {
        rug::Integer::from_digits(bytes, rug::integer::Order::Lsf)
    }
}

// Ensure trait impls required for FibOps are met by rug::Integer
// rug::Integer implements standard ops, but typically by reference for heavy lifting.
// We might need to ensure wrapper refs work.
// Checked: rug::Integer implements Add<Integer> and Add<&Integer>.
// impl From<u32> for Integer -> Yes.
// impl From<u64> for Integer -> Yes.
// impl From<u128> for Integer -> Yes.

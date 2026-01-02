use crate::algo::fast_doubling::fib_pair;
use crate::FibNumber;
use rayon::iter::plumbing::{bridge, Producer};
use rayon::prelude::*;

// ============================================================================
// Lazy Fibonacci Range Iterator
// ============================================================================

/// Lazy iterator for Fibonacci sequence over a range $[start, end)$.
///
/// # Performance
/// - **Initialization**: $O(\log \text{start})$ using Fast Doubling to compute $(F(\text{start}), F(\text{start}+1))$.
/// - **Iteration**: $O(1)$ addition of the previous two terms for each `.next()` call.
/// - **Memory**: Only 2 `UBig` values are kept in memory (zero-allocation streaming), regardless of range size.
///
/// # Example
/// ```
/// use fibrust_core::FibRange;
///
/// // Get F(1000) to F(1009) lazily
/// let fibs: Vec<_> = FibRange::new(1000, 1010).collect();
///
/// // Stop early without computing remaining values
/// let first_three: Vec<_> = FibRange::new(1_000_000, 2_000_000).take(3).collect();
/// ```
pub struct FibRange {
    current: FibNumber,
    next: FibNumber,
    position: u64,
    end: u64,
    // State for DoubleEndedIterator
    back_current: FibNumber, // F(end-1)
    back_next: FibNumber,    // F(end)
}

impl FibRange {
    /// Creates a new lazy iterator for the range $[F(\text{start}), F(\text{end}))$.
    ///
    /// # Arguments
    ///
    /// * `start` - The starting index (inclusive).
    /// * `end` - The ending index (exclusive).
    ///
    /// # Complexity
    ///
    /// Initializing the iterator takes $O(\log \text{start})$ time to compute the starting pair.
    pub fn new(start: u64, end: u64) -> Self {
        if start >= end {
            return Self {
                current: FibNumber::from(0u32),
                next: FibNumber::from(0u32),
                position: 0,
                end: 0,
                back_current: FibNumber::from(0u32),
                back_next: FibNumber::from(0u32),
            };
        }

        // Fast Doubling to get (F(start), F(start+1)) in O(log start)
        let (f_a, f_a_plus_1) = fib_pair(start);

        // Fast Doubling to get (F(end-1), F(end)) in O(log end)
        // Needed for DoubleEndedIterator
        let (back_current, back_next) = if end > 0 {
            fib_pair(end - 1)
        } else {
            (FibNumber::from(0u32), FibNumber::from(0u32))
        };

        Self {
            current: f_a,
            next: f_a_plus_1,
            position: start,
            end,
            back_current,
            back_next,
        }
    }

    /// Returns the current position index in the Fibonacci sequence.
    #[inline]
    pub fn position(&self) -> u64 {
        self.position
    }
}

impl Iterator for FibRange {
    type Item = FibNumber;

    #[inline]
    fn next(&mut self) -> Option<Self::Item> {
        if self.position >= self.end {
            return None;
        }

        let result = self.current.clone();

        // Simple iteration: F(n+1) = F(n) + F(n-1)
        // Uses only 2 registers, pushes result on demand
        let new_next = &self.current + &self.next;
        self.current = std::mem::replace(&mut self.next, new_next);
        self.position += 1;

        Some(result)
    }

    #[inline]
    fn size_hint(&self) -> (usize, Option<usize>) {
        let remaining = self.end.saturating_sub(self.position) as usize;
        (remaining, Some(remaining))
    }
}

impl DoubleEndedIterator for FibRange {
    #[inline]
    fn next_back(&mut self) -> Option<Self::Item> {
        if self.position >= self.end {
            return None;
        }

        self.end -= 1;
        let result = self.back_current.clone();

        // Calculate previous state
        // We have back_current = F(end)
        // We have back_next = F(end+1)
        // We want new back_current = F(end-1) = F(end+1) - F(end)

        // new_back_next = old_back_current
        // new_back_current = old_back_next - old_back_current

        let new_back_current = &self.back_next - &self.back_current;
        self.back_next = std::mem::replace(&mut self.back_current, new_back_current);

        Some(result)
    }
}

impl ExactSizeIterator for FibRange {}

// ============================================================================
// Parallel Fibonacci Iterator
// ============================================================================

/// Parallel iterator wrapper for `FibRange`.
///
/// This struct allows `FibRange` to be used with Rayon for parallel processing.
/// It splits the range into smaller sub-ranges, initializing each sub-range
/// independently in $O(\log \text{sub\_start})$ time.
pub struct ParFibRange {
    pub(crate) start: u64,
    pub(crate) end: u64,
}

impl ParallelIterator for ParFibRange {
    type Item = FibNumber;

    fn drive_unindexed<C>(self, consumer: C) -> C::Result
    where
        C: rayon::iter::plumbing::UnindexedConsumer<Self::Item>,
    {
        bridge(self, consumer)
    }

    fn opt_len(&self) -> Option<usize> {
        Some((self.end.saturating_sub(self.start)) as usize)
    }
}

impl IndexedParallelIterator for ParFibRange {
    fn len(&self) -> usize {
        (self.end.saturating_sub(self.start)) as usize
    }

    fn drive<C>(self, consumer: C) -> C::Result
    where
        C: rayon::iter::plumbing::Consumer<Self::Item>,
    {
        bridge(self, consumer)
    }

    fn with_producer<CB>(self, callback: CB) -> CB::Output
    where
        CB: rayon::iter::plumbing::ProducerCallback<Self::Item>,
    {
        callback.callback(FibProducer {
            start: self.start,
            end: self.end,
        })
    }
}

/// Producer that splits the Fibonacci range into chunks.
struct FibProducer {
    start: u64,
    end: u64,
}

impl Producer for FibProducer {
    type Item = FibNumber;
    type IntoIter = FibRange;

    fn into_iter(self) -> Self::IntoIter {
        FibRange::new(self.start, self.end)
    }

    fn split_at(self, index: usize) -> (Self, Self) {
        let mid = self.start + index as u64;
        (
            FibProducer {
                start: self.start,
                end: mid,
            },
            FibProducer {
                start: mid,
                end: self.end,
            },
        )
    }
}

impl IntoParallelIterator for FibRange {
    type Item = FibNumber;
    type Iter = ParFibRange;

    fn into_par_iter(self) -> Self::Iter {
        ParFibRange {
            start: self.position,
            end: self.end,
        }
    }
}

// ============================================================================
// Infinite Fibonacci Iterator
// ============================================================================

/// Infinite lazy iterator for the Fibonacci sequence starting at any index.
///
/// Similar to `FibRange` but never terminates. Use `.take(n)` to limit the output.
///
/// # Example
///
/// ```
/// use fibrust_core::FibIter;
///
/// // Infinite iterator starting from F(0)
/// let iter = FibIter::new();
///
/// // Get the first 5 numbers
/// let first_five: Vec<_> = iter.take(5).collect();
/// assert_eq!(first_five.len(), 5);
/// ```
pub struct FibIter {
    current: FibNumber,
    next: FibNumber,
    position: u64,
}

impl FibIter {
    /// Creates an infinite iterator starting at $F(\text{start})$.
    pub fn from(start: u64) -> Self {
        let (f_a, f_a_plus_1) = fib_pair(start);
        Self {
            current: f_a,
            next: f_a_plus_1,
            position: start,
        }
    }

    /// Creates an infinite iterator starting at $F(0)$.
    pub fn new() -> Self {
        Self::from(0)
    }

    /// Returns the current position index in the Fibonacci sequence.
    #[inline]
    pub fn position(&self) -> u64 {
        self.position
    }
}

impl Default for FibIter {
    fn default() -> Self {
        Self::new()
    }
}

impl Iterator for FibIter {
    type Item = FibNumber;

    #[inline]
    fn next(&mut self) -> Option<Self::Item> {
        let result = self.current.clone();
        let new_next = &self.current + &self.next;
        self.current = std::mem::replace(&mut self.next, new_next);
        self.position += 1;
        Some(result)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // ========================================================================
    // Tests for FibRange
    // ========================================================================

    #[test]
    fn fib_range_empty_when_start_equals_end() {
        let range: Vec<FibNumber> = FibRange::new(10, 10).collect();
        assert!(range.is_empty());
    }

    #[test]
    fn fib_range_empty_when_start_greater_than_end() {
        let range: Vec<FibNumber> = FibRange::new(100, 50).collect();
        assert!(range.is_empty());
    }

    #[test]
    fn fib_range_single_element() {
        let range: Vec<FibNumber> = FibRange::new(10, 11).collect();
        assert_eq!(range.len(), 1);

        // F(10) = 55
        assert_eq!(range[0], FibNumber::from(55u32));
    }

    #[test]
    fn fib_range_position_tracking() {
        let mut range = FibRange::new(100, 105);

        assert_eq!(range.position(), 100);
        range.next();
        assert_eq!(range.position(), 101);
        range.next();
        assert_eq!(range.position(), 102);
    }

    #[test]
    fn fib_range_size_hint_accurate() {
        let range = FibRange::new(0, 100);
        assert_eq!(range.size_hint(), (100, Some(100)));

        let mut range = FibRange::new(0, 10);
        assert_eq!(range.size_hint(), (10, Some(10)));
        range.next();
        assert_eq!(range.size_hint(), (9, Some(9)));
    }

    #[test]
    fn fib_range_exact_size_iterator() {
        let range = FibRange::new(0, 50);
        assert_eq!(range.len(), 50);
    }

    // ========================================================================
    // Tests for FibRange::next_back (DoubleEndedIterator)
    // ========================================================================

    #[test]
    fn fib_range_next_back_single() {
        let mut range = FibRange::new(10, 15);

        // Should return F(14), F(13), F(12), F(11), F(10)
        let last = range.next_back().expect("Should have last element");
        assert_eq!(last, fib_pair(14).0); // F(14)
    }

    #[test]
    fn fib_range_next_back_all() {
        let mut range = FibRange::new(0, 5);
        let mut backward: Vec<FibNumber> = Vec::new();

        while let Some(val) = range.next_back() {
            backward.push(val);
        }

        // Should be F(4), F(3), F(2), F(1), F(0)
        assert_eq!(backward.len(), 5);
        assert_eq!(backward[0], FibNumber::from(3u32)); // F(4)
        assert_eq!(backward[4], FibNumber::from(0u32)); // F(0)
    }

    #[test]
    fn fib_range_mixed_forward_backward() {
        let mut range = FibRange::new(0, 10);

        // Take from front
        let f0 = range.next().expect("F(0)");
        let f1 = range.next().expect("F(1)");

        // Take from back
        let f9 = range.next_back().expect("F(9)");
        let f8 = range.next_back().expect("F(8)");

        assert_eq!(f0, FibNumber::from(0u32));
        assert_eq!(f1, FibNumber::from(1u32));
        assert_eq!(f9, FibNumber::from(34u32)); // F(9)
        assert_eq!(f8, FibNumber::from(21u32)); // F(8)

        // Remaining should be F(2)..F(7) = 6 elements
        assert_eq!(range.len(), 6);
    }

    #[test]
    fn fib_range_next_back_empty() {
        let mut range = FibRange::new(5, 5);
        assert!(range.next_back().is_none());
    }

    // ========================================================================
    // Tests for ParFibRange (ParallelIterator)
    // ========================================================================

    #[test]
    fn par_fib_range_matches_sequential() {
        let seq_range: Vec<FibNumber> = FibRange::new(100, 200).collect();
        let par_range: Vec<FibNumber> = FibRange::new(100, 200).into_par_iter().collect();

        assert_eq!(seq_range, par_range);
    }

    #[test]
    fn par_fib_range_large() {
        // Just verify it runs and returns correct count, not checking all values here
        let count = FibRange::new(0, 1000).into_par_iter().count();
        assert_eq!(count, 1000);
    }

    #[test]
    fn par_fib_range_sum() {
        // Sum F(0)..F(10)
        // 0, 1, 1, 2, 3, 5, 8, 13, 21, 34 => Sum = 88
        let sum: FibNumber = FibRange::new(0, 10)
            .into_par_iter()
            .reduce(|| FibNumber::from(0u32), |a, b| a + b);
        assert_eq!(sum, FibNumber::from(88u32));
    }

    // ========================================================================
    // Tests for FibIter
    // ========================================================================

    #[test]
    fn fib_iter_new_starts_at_zero() {
        let mut iter = FibIter::new();

        assert_eq!(iter.position(), 0);
        assert_eq!(iter.next(), Some(FibNumber::from(0u32))); // F(0)
        assert_eq!(iter.next(), Some(FibNumber::from(1u32))); // F(1)
        assert_eq!(iter.next(), Some(FibNumber::from(1u32))); // F(2)
    }

    #[test]
    fn fib_iter_from_starts_at_index() {
        let mut iter = FibIter::from(10);

        assert_eq!(iter.position(), 10);
        let f10 = iter.next().expect("F(10)");
        assert_eq!(f10, FibNumber::from(55u32));
        assert_eq!(iter.position(), 11);
    }

    #[test]
    fn fib_iter_default() {
        let iter = FibIter::default();
        assert_eq!(iter.position(), 0);
    }

    #[test]
    fn fib_iter_position_tracking() {
        let mut iter = FibIter::from(100);

        assert_eq!(iter.position(), 100);
        iter.next();
        assert_eq!(iter.position(), 101);

        for _ in 0..10 {
            iter.next();
        }
        assert_eq!(iter.position(), 111);
    }

    #[test]
    fn fib_iter_infinite_take() {
        // FibIter never returns None
        let vals: Vec<FibNumber> = FibIter::new().take(100).collect();
        assert_eq!(vals.len(), 100);
        assert_eq!(vals[0], FibNumber::from(0u32));
        assert_eq!(vals[10], FibNumber::from(55u32));
    }

    #[test]
    fn fib_iter_always_returns_some() {
        let mut iter = FibIter::new();

        // Should never return None
        for _ in 0..1000 {
            assert!(iter.next().is_some());
        }
    }
}

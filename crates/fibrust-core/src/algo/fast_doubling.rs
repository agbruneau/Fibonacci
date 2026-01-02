use crate::FibNumber;
#[allow(unused_imports)]
use crate::FibOps;

/// Computes the nth Fibonacci number using optimized Fast Doubling.
///
/// This function acts as a wrapper around the `fib_pair` function, utilizing the same
/// O(log n) Fast Doubling algorithm. It extracts the first element of the pair (F(n))
/// and returns it.
///
/// # Arguments
///
/// * `n` - The index of the Fibonacci number to compute.
///
/// # Returns
///
/// * `FibNumber` - The nth Fibonacci number.
///
/// # Example
///
/// ```
/// use fibrust_core::algo::fast_doubling::fibonacci;
/// use fibrust_core::FibNumber;
/// let f10 = fibonacci(10);
/// assert_eq!(f10, FibNumber::from(55u32));
/// ```
#[inline]
pub fn fibonacci(n: u64) -> FibNumber {
    // Optimization: Use native u128 for n <= 186.
    // F(186) is the largest Fibonacci number that fits in u128.
    if n <= 186 {
        return FibNumber::from(fibonacci_u128(n));
    }
    fib_pair(n).0
}

/// Alias for `fibonacci` to maintain backward compatibility and explicit naming.
#[inline]
pub fn fibonacci_fast_doubling(n: u64) -> FibNumber {
    fibonacci(n)
}

/// Precomputed table for F(0) to F(186).
/// Generated using `generate_table.py`.
static FIB_U128_LOOKUP: [u128; 187] = [
    0,
    1,
    1,
    2,
    3,
    5,
    8,
    13,
    21,
    34,
    55,
    89,
    144,
    233,
    377,
    610,
    987,
    1597,
    2584,
    4181,
    6765,
    10946,
    17711,
    28657,
    46368,
    75025,
    121393,
    196418,
    317811,
    514229,
    832040,
    1346269,
    2178309,
    3524578,
    5702887,
    9227465,
    14930352,
    24157817,
    39088169,
    63245986,
    102334155,
    165580141,
    267914296,
    433494437,
    701408733,
    1134903170,
    1836311903,
    2971215073,
    4807526976,
    7778742049,
    12586269025,
    20365011074,
    32951280099,
    53316291173,
    86267571272,
    139583862445,
    225851433717,
    365435296162,
    591286729879,
    956722026041,
    1548008755920,
    2504730781961,
    4052739537881,
    6557470319842,
    10610209857723,
    17167680177565,
    27777890035288,
    44945570212853,
    72723460248141,
    117669030460994,
    190392490709135,
    308061521170129,
    498454011879264,
    806515533049393,
    1304969544928657,
    2111485077978050,
    3416454622906707,
    5527939700884757,
    8944394323791464,
    14472334024676221,
    23416728348467685,
    37889062373143906,
    61305790721611591,
    99194853094755497,
    160500643816367088,
    259695496911122585,
    420196140727489673,
    679891637638612258,
    1100087778366101931,
    1779979416004714189,
    2880067194370816120,
    4660046610375530309,
    7540113804746346429,
    12200160415121876738,
    19740274219868223167,
    31940434634990099905,
    51680708854858323072,
    83621143489848422977,
    135301852344706746049,
    218922995834555169026,
    354224848179261915075,
    573147844013817084101,
    927372692193078999176,
    1500520536206896083277,
    2427893228399975082453,
    3928413764606871165730,
    6356306993006846248183,
    10284720757613717413913,
    16641027750620563662096,
    26925748508234281076009,
    43566776258854844738105,
    70492524767089125814114,
    114059301025943970552219,
    184551825793033096366333,
    298611126818977066918552,
    483162952612010163284885,
    781774079430987230203437,
    1264937032042997393488322,
    2046711111473984623691759,
    3311648143516982017180081,
    5358359254990966640871840,
    8670007398507948658051921,
    14028366653498915298923761,
    22698374052006863956975682,
    36726740705505779255899443,
    59425114757512643212875125,
    96151855463018422468774568,
    155576970220531065681649693,
    251728825683549488150424261,
    407305795904080553832073954,
    659034621587630041982498215,
    1066340417491710595814572169,
    1725375039079340637797070384,
    2791715456571051233611642553,
    4517090495650391871408712937,
    7308805952221443105020355490,
    11825896447871834976429068427,
    19134702400093278081449423917,
    30960598847965113057878492344,
    50095301248058391139327916261,
    81055900096023504197206408605,
    131151201344081895336534324866,
    212207101440105399533740733471,
    343358302784187294870275058337,
    555565404224292694404015791808,
    898923707008479989274290850145,
    1454489111232772683678306641953,
    2353412818241252672952597492098,
    3807901929474025356630904134051,
    6161314747715278029583501626149,
    9969216677189303386214405760200,
    16130531424904581415797907386349,
    26099748102093884802012313146549,
    42230279526998466217810220532898,
    68330027629092351019822533679447,
    110560307156090817237632754212345,
    178890334785183168257455287891792,
    289450641941273985495088042104137,
    468340976726457153752543329995929,
    757791618667731139247631372100066,
    1226132595394188293000174702095995,
    1983924214061919432247806074196061,
    3210056809456107725247980776292056,
    5193981023518027157495786850488117,
    8404037832974134882743767626780173,
    13598018856492162040239554477268290,
    22002056689466296922983322104048463,
    35600075545958458963222876581316753,
    57602132235424755886206198685365216,
    93202207781383214849429075266681969,
    150804340016807970735635273952047185,
    244006547798191185585064349218729154,
    394810887814999156320699623170776339,
    638817435613190341905763972389505493,
    1033628323428189498226463595560281832,
    1672445759041379840132227567949787325,
    2706074082469569338358691163510069157,
    4378519841510949178490918731459856482,
    7084593923980518516849609894969925639,
    11463113765491467695340528626429782121,
    18547707689471986212190138521399707760,
    30010821454963453907530667147829489881,
    48558529144435440119720805669229197641,
    78569350599398894027251472817058687522,
    127127879743834334146972278486287885163,
    205697230343233228174223751303346572685,
    332825110087067562321196029789634457848,
];

/// Computes Fibonacci number for n <= 186 using a precomputed lookup table.
/// This provides O(1) access for small inputs.
///
/// # Note on Size Limit
///
/// $F(186)$ is the largest Fibonacci number that fits within a `u128`.
/// $F(186) \approx 3.3 \times 10^{38}$, while `u128::MAX` $\approx 3.4 \times 10^{38}$.
///
/// # Arguments
/// * `n` - Index (must be <= 186)
#[inline]
fn fibonacci_u128(n: u64) -> u128 {
    if n > 186 {
        panic!("fibonacci_u128 called with n > 186");
    }
    FIB_U128_LOOKUP[n as usize]
}

/// Returns (F(n), F(n+1)) using Fast Doubling in O(log n).
///
/// This is the core building block for the lazy FibRange iterator and the main
/// Fibonacci calculation, allowing efficient "jump" to any position in the sequence.
///
/// # Algorithm
///
/// Uses the Fast Doubling identities:
/// - F(2k) = F(k) * [2*F(k+1) - F(k)]
/// - F(2k+1) = F(k)² + F(k+1)²
///
/// # Arguments
/// * `n` - The index of the Fibonacci number.
///
/// # Returns
/// * `(FibNumber, FibNumber)` - A tuple containing (F(n), F(n+1)).
#[inline]
pub fn fib_pair(n: u64) -> (FibNumber, FibNumber) {
    // F(186) fits in u128, so for the pair (F(n), F(n+1)), max n is 185.
    if n <= 185 {
        let (a, b) = fib_pair_u128(n);
        return (FibNumber::from(a), FibNumber::from(b));
    }

    if n == 0 {
        return (FibNumber::from(0u32), FibNumber::from(1u32));
    }

    let highest_bit = 63 - n.leading_zeros() as usize;
    // Optimization: Skip the MSB (which is always 1) and start with state for F(1), F(2).
    // State (a, b) corresponds to (F(k), F(k+1)).
    // After MSB (1), k=1, so we start with (F(1), F(2)) = (1, 1).
    let mut a = FibNumber::from(1u32);
    let mut b = FibNumber::from(1u32);

    for i in (0..highest_bit).rev() {
        // Calculate d = F(2k+1) = F(k)² + F(k+1)²
        // Optimization: Use explicit multiplication for potential reuse/avoid generic overhead
        let a_sq = &a * &a;
        let b_sq = &b * &b;
        let d = &a_sq + &b_sq;

        // Calculate e = F(2k+2) = b * (2a + b)
        // We compute this conditionally to save one multiplication.

        if (n >> i) & 1 == 0 {
            // Bit is 0: Next state is (F(2k), F(2k+1))
            // c = F(2k) = a * (2b - a)
            let mut two_b_minus_a = b.clone(); // Clone b to modify
            two_b_minus_a <<= 1;
            two_b_minus_a -= &a;

            // Reuse 'a' memory where possible
            let mut c = a;
            c *= &two_b_minus_a;

            a = c;
            b = d;
        } else {
            // Bit is 1: Next state is (F(2k+1), F(2k+2))
            // e = F(2k+2) = b * (2a + b)
            let mut two_a_plus_b = a; // Move a
            two_a_plus_b <<= 1;
            two_a_plus_b += &b;

            // Reuse 'b' memory where possible
            let mut e = b;
            e *= &two_a_plus_b;

            a = d;
            b = e;
        }
    }

    (a, b)
}

/// Computes (F(n), F(n+1)) for n <= 185 using native u128 arithmetic.
///
/// # Arguments
/// * `n` - Index (must be <= 185)
#[inline]
fn fib_pair_u128(n: u64) -> (u128, u128) {
    if n == 0 {
        return (0, 1);
    }

    let mut a: u128 = 0;
    let mut b: u128 = 1;

    for _ in 0..n {
        let next = a + b;
        a = b;
        b = next;
    }

    (a, b)
}

#[cfg(test)]
mod tests {
    use super::*;

    // ========================================================================
    // Tests for fibonacci_u128 (private function)
    // ========================================================================

    #[test]
    fn fibonacci_u128_base_cases() {
        assert_eq!(fibonacci_u128(0), 0);
        assert_eq!(fibonacci_u128(1), 1);
        assert_eq!(fibonacci_u128(2), 1);
    }

    #[test]
    fn fibonacci_u128_known_values() {
        assert_eq!(fibonacci_u128(10), 55);
        assert_eq!(fibonacci_u128(20), 6765);
        assert_eq!(fibonacci_u128(50), 12586269025);
    }

    #[test]
    fn fibonacci_u128_max_boundary() {
        // F(186) is the largest Fibonacci number that fits in u128
        let f185 = fibonacci_u128(185);
        let f186 = fibonacci_u128(186);

        // Verify they are different and F(186) > F(185)
        assert!(f186 > f185);

        // F(186) has 39 decimal digits
        let f186_str = f186.to_string();
        assert_eq!(f186_str.len(), 39, "F(186) should have 39 digits");
    }

    // ========================================================================
    // Tests for fib_pair_u128 (private function)
    // ========================================================================

    #[test]
    fn fib_pair_u128_base_cases() {
        assert_eq!(fib_pair_u128(0), (0, 1));
        assert_eq!(fib_pair_u128(1), (1, 1));
        assert_eq!(fib_pair_u128(2), (1, 2));
    }

    #[test]
    fn fib_pair_u128_consistency() {
        // Verify F(n) + F(n+1) = F(n+2)
        for n in 0..100 {
            let (fn_val, fn1_val) = fib_pair_u128(n);
            let (fn1_check, fn2_val) = fib_pair_u128(n + 1);
            assert_eq!(fn1_val, fn1_check, "F(n+1) mismatch at n={}", n);
            assert_eq!(fn_val + fn1_val, fn2_val, "Recurrence failed at n={}", n);
        }
    }

    #[test]
    fn fib_pair_u128_boundary() {
        // n=185 is max for fib_pair_u128 (F(185), F(186) both fit in u128)
        let (f185, f186) = fib_pair_u128(185);
        assert!(f186 > f185);
        assert_eq!(f185, fibonacci_u128(185));
        assert_eq!(f186, fibonacci_u128(186));
    }

    // ========================================================================
    // Tests for fib_pair (public function)
    // ========================================================================

    #[test]
    fn fib_pair_base_cases() {
        assert_eq!(fib_pair(0), (FibNumber::from(0u32), FibNumber::from(1u32)));
        assert_eq!(fib_pair(1), (FibNumber::from(1u32), FibNumber::from(1u32)));
        assert_eq!(fib_pair(2), (FibNumber::from(1u32), FibNumber::from(2u32)));
    }

    #[test]
    fn fib_pair_transition_boundary() {
        // n=185 uses u128 path
        let (f185_small, f186_small) = fib_pair(185);

        // n=186 triggers the big integer path (n > 185)
        let (f186_big, f187_big) = fib_pair(186);

        // They should be consistent
        assert_eq!(f186_small, f186_big, "F(186) mismatch between paths");

        // Verify F(185) + F(186) = F(187)
        assert_eq!(&f185_small + &f186_small, f187_big);
    }

    #[test]
    fn fib_pair_recurrence_large() {
        // Test recurrence for larger values (uses UBig path)
        for n in [200u64, 500, 1000] {
            let (fn_val, fn1_val) = fib_pair(n);
            let (fn1_check, fn2_val) = fib_pair(n + 1);
            assert_eq!(fn1_val, fn1_check, "F(n+1) mismatch at n={}", n);
            assert_eq!(&fn_val + &fn1_val, fn2_val, "Recurrence failed at n={}", n);
        }
    }

    // ========================================================================
    // Tests for fibonacci (wrapper function)
    // ========================================================================

    #[test]
    fn fibonacci_uses_u128_for_small() {
        // For n <= 186, should use u128 path internally
        let f100 = fibonacci(100);
        assert_eq!(f100, FibNumber::from(fibonacci_u128(100)));
    }

    #[test]
    fn fibonacci_handles_large() {
        // For n > 186, uses UBig path
        let f200 = fibonacci(200);
        // Verify against fib_pair
        let (f200_pair, _) = fib_pair(200);
        assert_eq!(f200, f200_pair);
    }

    #[test]
    fn fibonacci_fast_doubling_alias() {
        // fibonacci_fast_doubling should be identical to fibonacci
        for n in [0, 1, 100, 200, 500] {
            assert_eq!(fibonacci(n), fibonacci_fast_doubling(n));
        }
    }
}

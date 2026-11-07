//! Calcul exact des termes de la suite de Fibonacci.
use num_bigint::BigUint;
use std::str::FromStr;

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum Algo {
    Iterative,
    Matrix,
    Fast,
}

impl Algo {
    pub const ALL: [Algo; 3] = [Algo::Iterative, Algo::Matrix, Algo::Fast];

    pub fn name(self) -> &'static str {
        match self {
            Algo::Iterative => "iterative",
            Algo::Matrix => "matrix",
            Algo::Fast => "fast",
        }
    }
}

#[derive(Debug)]
pub struct UnknownAlgo;

/// Le nom doit être exactement `iterative`, `matrix` ou `fast`, casse comprise.
impl FromStr for Algo {
    type Err = UnknownAlgo;

    fn from_str(s: &str) -> Result<Self, UnknownAlgo> {
        Algo::ALL.into_iter().find(|a| a.name() == s).ok_or(UnknownAlgo)
    }
}

/// Rend F(n) par l'algorithme algo; l'enum et le u64 excluent un nom inconnu et n < 0 (refus dans FromStr).
pub fn fib(n: u64, algo: Algo) -> BigUint {
    algorithm(algo)(n)
}

/// L'aiguillage seul, séparé de `fib` pour qu'un test compare les adresses (PERF-RS-13) : les trois fonctions rendent
/// les mêmes valeurs, et aucun test de valeur ne trahit une coquille ici.
fn algorithm(algo: Algo) -> fn(u64) -> BigUint {
    match algo {
        Algo::Iterative => iterative,
        Algo::Matrix => matrix,
        Algo::Fast => fast,
    }
}

/// Un limbe décimal vaut 10^18 : deux limbes et une retenue tiennent dans un u64 (10^19 y déborderait).
const BASE: u64 = 1_000_000_000_000_000_000;
const LIMB_DIGITS: usize = 18;

/// F(0) … F(n) en décimal, par l'algorithme iterative sur des limbes décimaux (poids faible d'abord) :
/// l'addition y est directement en base 10, sans conversion binaire → décimal des n + 1 termes.
pub fn fib_list(n: u64) -> Vec<String> {
    let (mut a, mut b) = (vec![0], vec![1]);
    let mut terms = vec![decimal(&a)];
    for _ in 0..n {
        add(&mut a, &b); // a ← F(k) + F(k+1), sur place : aucune allocation par pas
        std::mem::swap(&mut a, &mut b);
        terms.push(decimal(&a));
    }
    terms
}

/// a += b en base 10^18, avec a.len() <= b.len() (F(k) n'a pas plus de limbes que F(k+1)).
fn add(a: &mut Vec<u64>, b: &[u64]) {
    a.resize(b.len(), 0);
    let mut carry = 0;
    for (x, &y) in a.iter_mut().zip(b) {
        let sum = *x + y + carry;
        carry = u64::from(sum >= BASE); // sans branche : la retenue est imprévisible
        *x = sum - carry * BASE;
    }
    if carry != 0 {
        a.push(carry);
    }
}

/// Les 10 000 groupes de 4 chiffres "0000" … "9999" : une lecture par groupe plutôt qu'une division par 10 par chiffre.
static QUADS: [[u8; 4]; 10_000] = {
    let mut t = [[0; 4]; 10_000];
    let mut i = 0;
    while i < 10_000 {
        t[i] =
            [b'0' + (i / 1000) as u8, b'0' + (i / 100 % 10) as u8, b'0' + (i / 10 % 10) as u8, b'0' + (i % 10) as u8];
        i += 1;
    }
    t
};

/// Écrit x (< 10^18) sur 18 chiffres, zéros de tête compris : 2 chiffres de tête, puis 4 groupes de 4.
fn put_limb(x: u64, out: &mut [u8; LIMB_DIGITS]) {
    out[..2].copy_from_slice(&QUADS[(x / 10_000_000_000_000_000) as usize][2..]);
    let mut rest = x % 10_000_000_000_000_000;
    for group in out[2..].rchunks_exact_mut(4) {
        group.copy_from_slice(&QUADS[(rest % 10_000) as usize]);
        rest /= 10_000;
    }
}

/// Les limbes (poids faible d'abord, jamais vide) en décimal : le limbe de tête sans zéros de tête, les autres sur 18.
fn decimal(limbs: &[u64]) -> String {
    let (&top, rest) = limbs.split_last().expect("au moins un limbe");
    let top_len = top.checked_ilog10().map_or(1, |d| d as usize + 1); // F(0) s'écrit « 0 »
    // Écriture en place dans un tampon de taille exacte : ni réallocation ni copie par limbe.
    let mut out = vec![0; top_len + rest.len() * LIMB_DIGITS];
    let (head, tail) = out.split_at_mut(top_len);
    let mut buf = [0; LIMB_DIGITS];
    put_limb(top, &mut buf);
    head.copy_from_slice(&buf[LIMB_DIGITS - top_len..]);
    for (chunk, &limb) in tail.chunks_exact_mut(LIMB_DIGITS).zip(rest.iter().rev()) {
        put_limb(limb, chunk.try_into().expect("morceau de 18 chiffres"));
    }
    String::from_utf8(out).expect("chiffres ASCII")
}

/// Applique la définition indice par indice, avec deux termes consécutifs en mémoire.
fn iterative(n: u64) -> BigUint {
    let (mut a, mut b) = (BigUint::ZERO, BigUint::ONE);
    for _ in 0..n {
        a += &b; // sur place, sans clone : a prend F(k) + F(k+1), puis les rôles s'échangent
        std::mem::swap(&mut a, &mut b);
    }
    a
}

/// Q^k = [[F(k+1), F(k)], [F(k), F(k−1)]] est symétrique : trois coefficients (p, q, r) suffisent.
struct QPower {
    p: BigUint,
    q: BigUint,
    r: BigUint,
}

/// Q^n, Q = [[1,1],[1,0]], par carrés successifs du poids fort au poids faible de n.
/// Un bit à 1 multiplie par Q, ce qui ne coûte que des additions : Q^k·Q = [[p+q, p], [q+r, q]], et q+r = p.
fn matrix(n: u64) -> BigUint {
    let bits = u64::BITS - n.leading_zeros();
    let mut m = QPower { p: BigUint::ONE, q: BigUint::ZERO, r: BigUint::ONE }; // Q^0
    for i in (1..bits).rev() {
        // (Q^k)² : trois carrés, et q' = p' − r' puisque p² − r² = q·(p + r)
        let (p2, q2, r2) = (&m.p * &m.p, &m.q * &m.q, &m.r * &m.r);
        let (p, r) = (p2 + &q2, q2 + r2);
        m = QPower { q: &p - &r, p, r };
        if (n >> i) & 1 == 1 {
            m = QPower { p: &m.p + &m.q, r: m.q, q: m.p };
        }
    }
    // Dernier bit : seul F(n) compte. Bit à 0, F(n) = q' = q·(p + r) (0 pour n = 0); bit à 1, F(n) = p' = p² + q².
    if n & 1 == 0 { &m.q * (&m.p + &m.r) } else { &m.p * &m.p + &m.q * &m.q }
}

/// x ← x + 2(−1)^k, pour k de parité donnée : +2 si k est pair, −2 s'il est impair.
fn add_alternating_two(x: &mut BigUint, k_odd: bool) {
    if k_odd {
        *x -= 2u8;
    } else {
        *x += 2u8;
    }
}

/// Doublement rapide : (F(k), F(k−1)) devient (F(2k), F(2k−1)) ou (F(2k+1), F(2k)) à chaque bit de n, du poids fort
/// au poids faible, avec deux carrés par bit (formules de GMP); la dernière étape ne calcule que F(n).
fn fast(n: u64) -> BigUint {
    let bits = u64::BITS - n.leading_zeros();
    if bits < 2 {
        return BigUint::from(n); // F(0) = 0 et F(1) = 1
    }
    let (mut a, mut b) = (BigUint::ONE, BigUint::ZERO); // k = 1 : le bit de poids fort est consommé
    for i in (1..bits - 1).rev() {
        let (a2, b2) = (&a * &a, &b * &b);
        let odd_inv = &a2 + &b2; // F(2k−1) = F(k)² + F(k−1)²
        let mut odd = (a2 << 2u32) - &b2; // F(2k+1) = 4F(k)² − F(k−1)² + 2(−1)^k : le 2 vient après, en non signé
        add_alternating_two(&mut odd, (n >> (i + 1)) & 1 == 1);
        let even = &odd - &odd_inv; // F(2k) = F(2k+1) − F(2k−1)
        (a, b) = if (n >> i) & 1 == 1 { (odd, even) } else { (even, odd_inv) };
    }
    // Dernier bit, avec une seule multiplication au lieu de deux carrés : F(2k) = F(k)·(F(k) + 2F(k−1)),
    // F(2k+1) = (2F(k) − F(k−1))·(2F(k) + F(k−1)) + 2(−1)^k.
    if n & 1 == 0 {
        return &a * (&a + (b << 1u32));
    }
    let a2 = a << 1u32;
    let mut odd = (&a2 - &b) * (a2 + &b);
    add_alternating_two(&mut odd, (n >> 1) & 1 == 1);
    odd
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sum(a: &[u64], b: &[u64]) -> Vec<u64> {
        let mut a = a.to_vec();
        add(&mut a, b);
        a
    }

    // PERF-RS-4, hors catalogue : garde de l'optimisation « additionneur décimal en limbes ».
    #[test]
    fn perf_rs_4_une_somme_de_limbes_egale_a_10_puissance_18_propage_sa_retenue() {
        assert_eq!(sum(&[BASE / 2, 3], &[BASE / 2, 4]), [0, 8]);
    }

    // PERF-RS-5, hors catalogue : garde de l'optimisation « additionneur décimal en limbes ».
    #[test]
    fn perf_rs_5_la_retenue_franchit_les_limbes_pleins() {
        assert_eq!(sum(&[1, 0, 7], &[BASE - 1, BASE - 1, 0]), [0, 0, 8]);
    }

    // PERF-RS-6, hors catalogue : garde de l'optimisation « additionneur décimal en limbes ».
    #[test]
    fn perf_rs_6_une_retenue_hors_du_dernier_limbe_en_cree_un_nouveau() {
        assert_eq!(sum(&[BASE - 1, BASE - 1], &[1, 0]), [0, 0, 1]);
    }

    // PERF-RS-7, hors catalogue : garde de l'optimisation « additionneur décimal en limbes ».
    #[test]
    fn perf_rs_7_une_somme_de_limbes_au_dessus_de_10_puissance_18_garde_son_reste() {
        assert_eq!(sum(&[BASE - 1], &[BASE - 1]), [BASE - 2, 1]);
    }

    // PERF-RS-13 (F2 et F6), hors catalogue : chaque algorithme est aiguillé vers sa propre fonction. Les trois rendent
    // les mêmes valeurs, et PERF-RS-12 espionne le nom demandé, non la fonction qui calcule derrière : seule l'adresse
    // de la fonction distingue une coquille d'aiguillage. La garde ne compare donc ni valeur calculée ni durée, et
    // vérifie d'abord que les trois adresses diffèrent, sans quoi l'égalité passerait à vide. Rouge sous les six
    // échanges d'une branche de `algorithm` : chacun des trois noms vers l'une des deux autres fonctions, dont
    // `Algo::Matrix => fast`. `fib` qui contourne `algorithm` (`iterative(n)` en dur) ne rougit aucun test, mais clippy
    // `-D warnings` le refuse (`algo` inutilisé ; `algorithm`, `matrix` et `fast` morts) ; `matrix(n)` ou `fast(n)` en
    // dur rougissent en plus PERF-RS-9, qui compte les allocations d'`iterative`.
    #[test]
    fn perf_rs_13_chaque_algorithme_est_aiguille_vers_sa_propre_fonction() {
        type Term = fn(u64) -> BigUint;
        let routes: [(Algo, Term); 3] = [(Algo::Iterative, iterative), (Algo::Matrix, matrix), (Algo::Fast, fast)];
        for (i, (a, own_a)) in routes.iter().enumerate() {
            for (b, own_b) in &routes[i + 1..] {
                assert!(!std::ptr::fn_addr_eq(*own_a, *own_b), "{} et {} ont la même adresse", a.name(), b.name());
            }
        }
        for (algo, own) in routes {
            assert!(
                std::ptr::fn_addr_eq(algorithm(algo), own),
                "{} n'est pas aiguillé vers sa propre fonction",
                algo.name()
            );
        }
    }
}

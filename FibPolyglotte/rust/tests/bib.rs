//! Seam 1 : la bibliothèque. Les attendus des cas BIB viennent de l'oracle (contracts/b000045.txt) ou de faits
//! mathématiques; les gardes PERF-RS-1 à 3, hors seams, comparent à un autre algorithme.
use fib::{Algo, fib, fib_list};
use num_bigint::BigUint;

/// F(0) … F(2000), lus à l'exécution. Un oracle absent ou invalide fait échouer le test;
/// il ne sert jamais d'attendu par défaut.
fn oracle_text() -> Vec<String> {
    let path = concat!(env!("CARGO_MANIFEST_DIR"), "/../contracts/b000045.txt");
    let text = std::fs::read_to_string(path).expect("oracle illisible : contracts/b000045.txt");
    let terms: Vec<String> = text
        .lines()
        .filter(|l| !l.is_empty())
        .enumerate()
        .map(|(i, l)| {
            let (n, term) = l.split_once(' ').expect("oracle invalide : ligne sans espace");
            assert_eq!(n, i.to_string(), "oracle invalide : indice hors séquence");
            term.to_string()
        })
        .collect();
    assert_eq!(terms.len(), 2001, "oracle invalide : 2 001 termes attendus");
    terms
}

/// Les mêmes termes en entiers : un terme non décimal fait échouer le test.
fn oracle() -> Vec<BigUint> {
    oracle_text().iter().map(|t| t.parse().expect("oracle invalide : terme non décimal")).collect()
}

/// fib(n, algo) égale l'oracle pour n = 0..2000.
fn sweep(algo: Algo) {
    for (n, want) in oracle().iter().enumerate() {
        assert_eq!(fib(n as u64, algo), *want, "n = {n}");
    }
}

// BIB-01
#[test]
fn bib_01_fib_0_iterative_vaut_0() {
    assert_eq!(fib(0, Algo::Iterative), oracle()[0]);
}

// BIB-02
#[test]
fn bib_02_fib_1_iterative_vaut_1() {
    assert_eq!(fib(1, Algo::Iterative), oracle()[1]);
}

// BIB-03
#[test]
fn bib_03_fib_2_iterative_vaut_1() {
    assert_eq!(fib(2, Algo::Iterative), oracle()[2]);
}

// BIB-04
#[test]
fn bib_04_fib_10_iterative_vaut_55() {
    assert_eq!(fib(10, Algo::Iterative), oracle()[10]);
}

// BIB-05
#[test]
fn bib_05_iterative_egale_l_oracle_de_0_a_2000_au_dela_d_int64_et_d_uint64() {
    sweep(Algo::Iterative);
}

// BIB-06
#[test]
fn bib_06_fast_egale_l_oracle_de_0_a_2000() {
    sweep(Algo::Fast);
}

// BIB-07
#[test]
fn bib_07_matrix_egale_l_oracle_de_0_a_2000() {
    sweep(Algo::Matrix);
}

// BIB-08
#[test]
fn bib_08_fib_100000_rend_le_meme_terme_de_20899_chiffres_pour_les_trois_algorithmes() {
    // Pas d'assert_eq! : il imprimerait deux fois 20 899 chiffres. Le message nomme l'algorithme fautif.
    let terms = Algo::ALL.map(|a| fib(100_000, a));
    for (algo, term) in Algo::ALL.into_iter().zip(&terms).skip(1) {
        assert!(*term == terms[0], "{algo:?} rend un autre terme que {:?} pour n = 100000", Algo::ALL[0]);
    }
    assert_eq!(terms[0].to_string().len(), 20_899);
}

// BIB-09
#[test]
fn bib_09_cassini_a_n_100000_pour_chaque_algorithme() {
    for algo in Algo::ALL {
        let [before, at, after] = [99_999, 100_000, 100_001].map(|n| fib(n, algo));
        // BigUint est non signé : on compare before·after à at² + 1 plutôt que de soustraire.
        assert_eq!(before * after, at.pow(2) + 1u8, "{algo:?}");
    }
}

// BIB-10
#[test]
fn bib_10_fib_list_0_vaut_0() {
    assert_eq!(fib_list(0), oracle_text()[..1]);
}

// BIB-11
#[test]
fn bib_11_fib_list_1_vaut_0_1() {
    assert_eq!(fib_list(1), oracle_text()[..2]);
}

// BIB-12
#[test]
fn bib_12_fib_list_2000_rend_les_2001_termes_de_l_oracle_dans_l_ordre() {
    assert_eq!(fib_list(2000), oracle_text());
}

// BIB-13
#[test]
fn bib_13_un_nom_d_algorithme_inconnu_est_refuse() {
    for name in ["all", "bogus", "", "FAST"] {
        assert!(name.parse::<Algo>().is_err(), "{name:?} devrait être refusé");
    }
}

// BIB-14 ne s'applique pas : l'indice est un u64 (docs/cas-de-test.md § 4).

// PERF-RS-1, hors catalogue : garde de l'optimisation « liste en limbes décimaux » (au-delà de l'oracle, contre fast).
#[test]
fn perf_rs_1_fib_list_5000_egale_fast_en_decimal_terme_par_terme() {
    for (k, got) in fib_list(5000).iter().enumerate() {
        assert_eq!(*got, fib(k as u64, Algo::Fast).to_string(), "k = {k}");
    }
}

/// algo rend les termes d'iterative aux puissances de 2 jusqu'à 2^14 et à leurs voisins, tous bits à 1 compris.
fn assert_agrees_around_powers_of_2(algo: Algo) {
    for j in 1..=14u32 {
        for n in [(1u64 << j) - 1, 1 << j, (1 << j) + 1] {
            assert_eq!(fib(n, algo), fib(n, Algo::Iterative), "n = {n}");
        }
    }
}

// PERF-RS-2, hors catalogue : garde de l'optimisation de matrix (dernière étape réduite à F(n)).
#[test]
fn perf_rs_2_matrix_egale_iterative_autour_des_puissances_de_2_jusqu_a_2_puissance_14() {
    assert_agrees_around_powers_of_2(Algo::Matrix);
}

// PERF-RS-3, hors catalogue : garde de l'optimisation de fast (dernière étape réduite à F(n), parité de k).
#[test]
fn perf_rs_3_fast_egale_iterative_autour_des_puissances_de_2_jusqu_a_2_puissance_14() {
    assert_agrees_around_powers_of_2(Algo::Fast);
}

//! Gardes d'allocation (PERF-RS-9 et 10), hors seams (catalogue § 6). Crate de test à part : son allocateur de
//! comptage n'agit pas sur `bib.rs`.
use fib::{Algo, fib, fib_list};
use std::{
    alloc::{GlobalAlloc, Layout, System},
    cell::Cell,
    hint::black_box,
};

thread_local! {
    // Un compteur par fil, car libtest lance les tests en parallèle; initialisation `const`, donc sans allocation.
    static ALLOCS: Cell<usize> = const { Cell::new(0) };
}

/// Délègue à `System` et compte `alloc` et `realloc`. `alloc_zeroed` passe par `alloc`, donc par le compteur.
struct Counting;

fn count() {
    // `try_with` plutôt que `with` : il ne panique pas si le fil détruit déjà ses variables locales.
    let _ = ALLOCS.try_with(|n| n.set(n.get() + 1));
}

// SAFETY : chaque méthode passe ses arguments tels quels à `System`, qui tient le contrat de `GlobalAlloc`.
unsafe impl GlobalAlloc for Counting {
    unsafe fn alloc(&self, layout: Layout) -> *mut u8 {
        count();
        unsafe { System.alloc(layout) }
    }

    unsafe fn dealloc(&self, ptr: *mut u8, layout: Layout) {
        unsafe { System.dealloc(ptr, layout) }
    }

    unsafe fn realloc(&self, ptr: *mut u8, layout: Layout, new_size: usize) -> *mut u8 {
        count();
        unsafe { System.realloc(ptr, layout, new_size) }
    }
}

#[global_allocator]
static ALLOCATOR: Counting = Counting;

/// Les allocations que `f` fait sur ce fil.
fn allocations<T>(f: impl FnOnce() -> T) -> usize {
    let before = ALLOCS.get();
    black_box(f());
    ALLOCS.get() - before
}

/// Plafond de PERF-RS-9. Observé : 16; la marge absorbe les changements de croissance de `BigUint` et de la chaîne
/// d'outils.
const ITERATIVE_MAX_ALLOCS: usize = 32;

/// Marge de PERF-RS-10 au-dessus d'une allocation par terme. Observé : 28 de plus, pour le vecteur des termes et les
/// deux additionneurs, qui grandissent par doublement.
const LIST_SLACK: usize = 64;

// PERF-RS-9, hors catalogue : garde de l'optimisation d'iterative (`a += &b` sur place) : pas d'allocation par pas.
// Elle compte les allocations au lieu de chronométrer : un chronomètre serait instable, ce compte est exact.
// Observé : 20 123 avec `a = &a + &b`.
#[test]
fn perf_rs_9_fib_20000_iterative_n_alloue_pas_par_pas() {
    let got = allocations(|| fib(20_000, Algo::Iterative));
    assert!(
        got <= ITERATIVE_MAX_ALLOCS,
        "fib(20000, iterative) alloue {got} fois, attendu {ITERATIVE_MAX_ALLOCS} au plus"
    );
}

// PERF-RS-10, hors catalogue : garde de l'optimisation de `fib_list` (`add` sur place) : l'addition n'alloue pas.
// La chaîne de chaque terme est inévitable : une allocation par terme est permise, pas davantage. Elle compte les
// allocations au lieu de chronométrer : un chronomètre serait instable, ce compte est exact. Observé : 20 249 avec
// un clone de `a` à chaque pas.
#[test]
fn perf_rs_10_fib_list_10000_n_alloue_que_la_chaine_de_chaque_terme() {
    let n = 10_000;
    let max = n as usize + 1 + LIST_SLACK; // F(0) … F(n) : n + 1 termes
    let got = allocations(|| fib_list(n));
    assert!(got <= max, "fib_list({n}) alloue {got} fois, attendu {max} au plus");
}

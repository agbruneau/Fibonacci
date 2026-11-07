"""Fibonacci académique : bibliothèque (fib, fib_list), puis CLI."""
import sys
import time

# Sans cela, int() d'un indice de plus de 4 300 chiffres (zéros de tête) et str() d'un terme de plus de 4 300 chiffres
# lèvent ValueError.
sys.set_int_max_str_digits(0)


def _iterative(n):
    """Applique la définition indice par indice, avec deux termes consécutifs en mémoire."""
    a, b = 0, 1
    # Une addition par tour : deux (a += b; b += a) ne gagnent rien (mesuré : 50 ms dans les deux cas à n = 100 000).
    for _ in range(n):
        a, b = b, a + b
    return a


def _matrix(n, kind=int):
    """[[1,1],[1,0]]^n = [[F(n+1), F(n)], [F(n), F(n-1)]], par carrés successifs, du poids fort au poids faible de n.
    La puissance est symétrique, donc trois coefficients (a, b, c) suffisent : [[a, b], [b, c]]."""
    # kind : le même algorithme tourne sur int ou sur Decimal (voir _fib_decimal)
    a, b, c = kind(1), kind(0), kind(1)
    bits = format(n, "b")
    for bit in bits[:-1]:
        # Trois carrés, aucun produit (un carré coûte moins) : le nouveau b = a² − c² = (a − c)(a + c) = b(a + c).
        square_a, square_b, square_c = a * a, b * b, c * c
        a, c = square_a + square_b, square_b + square_c
        b = a - c
        if bit == "1":
            # fois [[1,1],[1,0]] : que des additions
            a, b, c = a + b, a, b
    # Dernier carré : seule l'entrée F(n) sert, donc un produit au plus (le bit à 1 décale (0,0) en (0,1)).
    return a * a + b * b if bits[-1] == "1" else b * (a + c)


def _fast(n, kind=int):
    """Doublement rapide : (F(k), F(k−1)) devient celui de 2k, ou de 2k+1 si le bit vaut 1, en suivant les bits de n.
    Formules de GMP, deux carrés par bit; le dernier bit ne calcule que F(n), en un seul produit."""
    if n < 2:
        return kind(n)
    f, g = kind(1), kind(0)
    # le bit de tête est déjà lu : k = 1
    *middle, last = format(n, "b")[1:]
    sign = -2  # 2·(−1)^k
    for bit in middle:
        square_f, square_g = f * f, g * g
        before = square_f + square_g  # F(2k−1)
        odd = 4 * square_f - square_g + sign  # F(2k+1)
        f, g = (odd, odd - before) if bit == "1" else (odd - before, before)  # F(2k) = F(2k+1) − F(2k−1)
        sign = -2 if bit == "1" else 2
    # La différence de deux carrés vaut un produit : F(2k+1) = (2f − g)(2f + g) + sign, et F(2k) = f(f + 2g).
    return (2 * f - g) * (2 * f + g) + sign if last == "1" else f * (f + 2 * g)


# Seul aiguillage des algorithmes, en int comme en Decimal : PERF-PY-26, 27 et 28 y substituent un algorithme qui
# discorde ou qui espionne, et PERF-PY-30 en vérifie les couples nom-fonction par identité.
_ALGOS = {"iterative": _iterative, "matrix": _matrix, "fast": _fast}


def _check_index(n):
    # ni float ni bool : sans lui, fast rendrait 1 pour 1.5 et 0 pour 0.0, et True passerait pour 1
    if type(n) is not int:
        raise TypeError(f"indice invalide : {n!r}")
    # Avant tout calcul : sur n < 0, les algorithmes rendraient un terme faux au lieu d'une erreur.
    if n < 0:
        raise ValueError(f"indice invalide : {n}")


def fib(n, algo):
    """Rend F(n) par l'algorithme algo.
    Lève ValueError pour un algorithme inconnu ou un indice négatif, TypeError pour un indice qui n'est pas un int."""
    if algo not in _ALGOS:
        raise ValueError(f"algorithme inconnu : {algo!r}")
    _check_index(n)
    return _ALGOS[algo](n)


def _exact_decimal():
    """Rend (Decimal, contexte) : le contexte exact et local que les chemins en Decimal ouvrent par with."""
    # Importé ici : environ 5 ms que le calcul d'un terme en int n'a pas à payer. Contexte exact : la précision par
    # défaut arrondirait. Local : celui de l'appelant reste intact.
    from decimal import MAX_EMAX, MAX_PREC, Context, Decimal, localcontext

    return Decimal, localcontext(Context(prec=MAX_PREC, Emax=MAX_EMAX))


def _fib_decimal(n, algo):
    """F(n) en Decimal par fast ou matrix, les seuls algorithmes qui acceptent kind; mêmes refus que fib() pour n."""
    # Decimal : son str() est linéaire, et évite la conversion décimale, quadratique, d'un grand int (0,5 ms contre 54).
    if algo not in ("fast", "matrix"):
        raise ValueError(f"pas de chemin Decimal pour l'algorithme {algo!r}")
    _check_index(n)
    Decimal, exact = _exact_decimal()
    with exact:
        return _ALGOS[algo](n, Decimal)


def fib_list(n):
    """F(0) … F(n) en texte décimal, toujours par l'algorithme iterative."""
    _check_index(n)
    Decimal, exact = _exact_decimal()
    # Calculé en décimal : le texte sort sans conversion binaire → décimal, bien plus lente (int : 180 ms, ici 25 ms
    # pour n = 10 000).
    with exact:
        terms = []
        a, b = Decimal(0), Decimal(1)
        for _ in range(n):
            terms.append(str(a))
            a, b = b, a + b
        terms.append(str(a))
    return terms


# --- CLI : adaptateur mince (PRD § 4, ADR-0002) ---

USAGE = """usage: fib [--algo iterative|matrix|fast|all] [--time] N
       fib --list [--time] N
       fib --help
"""

MAX_TERM, MAX_LIST = 1_000_000, 10_000

COMPARISON = ("fast", "iterative", "matrix")


class CliError(Exception):
    """Code 2 : erreur d'usage. Code 3 : discordance entre algorithmes."""

    def __init__(self, message, code=2):
        super().__init__(message)
        self.code = code


def parse(args):
    """Applique la grammaire du PRD § 4."""
    algos = None
    as_list = show_time = False
    positional = []
    # le for et next() avancent le même itérateur
    rest = iter(args)
    for a in rest:
        # --algo X : X est l'argument suivant, quel qu'il soit; chaque occurrence doit être valide, la dernière gagne.
        if a == "--algo" or a.startswith("--algo="):
            if a == "--algo":
                value = next(rest, None)
            else:
                value = a.removeprefix("--algo=")
            if value is None:
                raise CliError("--algo demande une valeur")
            if value == "all":
                algos = list(COMPARISON)
            elif value in COMPARISON:
                algos = [value]
            else:
                # l'argument fautif est cité, en ASCII
                raise CliError(f"algorithme inconnu : {ascii(value)}")
        elif a == "--list":
            as_list = True
        elif a == "--time":
            show_time = True
        elif a.startswith("-"):
            raise CliError(f"option inconnue : {ascii(a)}")
        else:
            positional.append(a)
    if as_list and algos:
        raise CliError("--list et --algo s'excluent")
    if len(positional) != 1:
        raise CliError("exactement un indice attendu")
    # avant int() : il accepte +5, 1_000, ' 5 ' et des chiffres non ASCII.
    # isascii() et isdigit() valent [0-9]+, sans payer l'import de re (8 ms).
    if not (positional[0].isascii() and positional[0].isdigit()):
        raise CliError("indice invalide : chiffres ASCII seulement")
    n = int(positional[0])
    bound = MAX_LIST if as_list else MAX_TERM
    if n > bound:
        raise CliError(f"indice hors borne ({bound} au plus)")
    return algos or ["fast"], as_list, show_time, n


# Chiffres par tranche de _text : le divmod d'une tranche reste sous le seuil de _pylong.
CHUNK_DIGITS = 1000

# Mesuré : au-delà (environ 60 000 chiffres), str() avec _pylong (import de 10 ms) bat les tranches, quadratiques.
STR_FROM_BITS = 200_000


def _text(x):
    """Texte décimal d'un int ou d'un Decimal. Un Decimal, ou un int de plus de STR_FROM_BITS bits, passe par str(); en
    deçà, des tranches de CHUNK_DIGITS chiffres : dès 9 000 chiffres str() importe _pylong (re, decimal : 10 ms)."""
    if type(x) is not int or x.bit_length() > STR_FROM_BITS:
        return str(x)
    base = 10**CHUNK_DIGITS
    parts = []
    while x >= base:
        x, rest = divmod(x, base)
        parts.append(str(rest).zfill(CHUNK_DIGITS))
    parts.append(str(x))
    return "".join(reversed(parts))


# Mesuré : à cet indice, l'import de decimal (5 ms) coûte autant que l'écriture par tranches, quadratique au-delà.
# Couplage : F(169 999) compte environ 118 000 bits, donc STR_FROM_BITS ne joue qu'avec iterative et all, dès F(288 086)
DECIMAL_FROM = 170_000


def compute(algos, as_list, n):
    """Rend les lignes à écrire et les lignes de durée; avec all, les trois algorithmes doivent concorder.
    Rien n'est écrit ici : une discordance (F3) laisse stdout vide et sans durée."""
    durations = []

    def timed(name, calc):
        """Une ligne « <algo> <durée> ms » par calcul : le chronomètre ne couvre que le calcul,
        et le format ignore la locale."""
        start = time.perf_counter()
        result = calc()
        durations.append(f"{name} {(time.perf_counter() - start) * 1000:.3f} ms\n")
        return result

    # Dès DECIMAL_FROM, fast et matrix seuls calculent en Decimal, sans conversion à l'écriture. Avec iterative (int),
    # == Decimal/int convertit l'int, quadratique (420 ms à 1 000 000), et le temps est déjà celui d'iterative.
    use_decimal = n >= DECIMAL_FROM and "iterative" not in algos
    if as_list or use_decimal:
        # Import pour son effet : hors du chronomètre (5 ms qui ne sont pas du calcul), fib_list et _fib_decimal le
        # trouvent déjà chargé.
        import decimal
    if as_list:
        return timed("iterative", lambda: fib_list(n)), durations
    terms = [timed(a, lambda: _fib_decimal(n, a) if use_decimal else fib(n, a)) for a in algos]
    if any(t != terms[0] for t in terms):
        raise CliError(f"discordance entre les algorithmes pour l'indice {n}", 3)
    return [_text(terms[0])], durations


# Un seul bloc de 10 Mo (--list 10000) coûte des fautes de page et vide les caches : 7 ms de plus que par 50 lignes.
LINES_PER_WRITE = 50


def main(args):
    # L'aide l'emporte sur tout le reste, y compris à la place de la valeur de --algo.
    if "--help" in args or "-h" in args:
        sys.stdout.write(USAGE)
        return 0
    try:
        algos, as_list, show_time, n = parse(args)
        lines, durations = compute(algos, as_list, n)
    except CliError as e:
        sys.stderr.write(f"fib: {e}\n")
        return e.code
    for i in range(0, len(lines), LINES_PER_WRITE):
        sys.stdout.write("\n".join(lines[i : i + LINES_PER_WRITE]) + "\n")
    # Les termes avant les durées, même sur un pipe qui fusionne stdout et stderr.
    sys.stdout.flush()
    if show_time:
        sys.stderr.write("".join(durations))
    return 0


if __name__ == "__main__":
    # F10 : ASCII et \n quel que soit l'environnement (PYTHONIOENCODING, Windows).
    for name in ("stdout", "stderr"):
        stream = getattr(sys, name)
        if stream is None:
            # Un fd 1 ou 2 fermé au lancement met le flux à None : /dev/null le remplace, et le code ne change pas, comme en Go, Rust et TS.
            import os  # ici seulement : 2,5 ms de démarrage sous -S (mesuré), que le chemin normal ne doit pas payer

            stream = open(os.devnull, "w")
            setattr(sys, name, stream)
        stream.reconfigure(encoding="ascii", newline="\n")
    sys.exit(main(sys.argv[1:]))

"""Génère l'oracle F(0) … F(2000) par la formule de Binet.

La formule F(n) = round(φⁿ / √5) est indépendante des trois algorithmes testés (iterative,
matrix, fast). Le test de contrat vérifie ensuite la table par la définition seule (ORA-04,
ORA-05) : toute table qui passe ce contrôle est la suite de Fibonacci.
Le format est celui du b-file OEIS A000045 : « n F(n) », une ligne par terme, fins de ligne LF.
"""

from decimal import Decimal, getcontext
from pathlib import Path

# F(2000) compte 418 chiffres; 600 chiffres de précision laissent une large marge à l'arrondi.
getcontext().prec = 600
SQRT5 = Decimal(5).sqrt()
PHI = (1 + SQRT5) / 2

lines = []
power = Decimal(1)  # φⁿ
for n in range(2001):
    # |F(n) − φⁿ/√5| < 0,45 pour tout n ≥ 0 : l'arrondi au plus proche donne F(n).
    lines.append(f"{n} {int((power / SQRT5).to_integral_value())}\n")
    power *= PHI

Path(__file__).with_name("b000045.txt").write_text("".join(lines), encoding="ascii", newline="\n")

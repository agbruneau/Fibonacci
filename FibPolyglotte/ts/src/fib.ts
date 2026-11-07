// Calcule les termes de la suite de Fibonacci : BigInt pour un terme, addition décimale à la main pour la liste.
export type Algo = 'iterative' | 'matrix' | 'fast';

// Applique la définition indice par indice, avec deux termes consécutifs en mémoire.
function iterative(n: number): bigint {
  let [a, b] = [0n, 1n];
  // V8 n'alloue pas le tableau du pas : la variable temporaire ou deux additions par tour ne gagnent rien (mesuré).
  for (let i = 0; i < n; i++) [a, b] = [b, a + b];
  return a;
}

// [[1,1],[1,0]]^k = [[F(k+1), F(k)], [F(k), F(k-1)]] est symétrique : trois coefficients (a, b, c) suffisent.
// Du poids fort au poids faible : un carré par bit, puis un bit à 1 multiplie par Q, ce qui ne coûte que des additions.
function matrix(n: number): bigint {
  let [a, b, c] = [1n, 0n, 1n]; // k = 0 : la matrice identité
  const bits = n.toString(2);
  for (const bit of bits.slice(0, -1)) {
    // Trois carrés, aucun produit : le nouveau b = a' − c' = a² − c² = b(a + c), avec les anciens a, b, c et a − c = b.
    const [aa, bb, cc] = [a * a, b * b, c * c];
    [a, c] = [aa + bb, bb + cc];
    b = a - c;
    if (bit === '1') [a, b, c] = [a + b, a, b]; // fois [[1,1],[1,0]]
  }
  // Dernier bit : seul F(n) est utile. À 0, c'est b du carré, b(a + c); à 1, c'est a du carré, a² + b².
  return bits.endsWith('0') ? b * (a + c) : a * a + b * b;
}

// Doublement rapide par la suite de Lucas L : F(2k) = F(k)·L(k) et L(2k) = L(k)² − 2(−1)^k, du poids fort au faible.
// Un bit à 1 avance d'un cran : F(m+1) = (F(m) + L(m)) / 2 et L(m+1) = (5F(m) + L(m)) / 2, des divisions exactes.
function fast(n: number): bigint {
  const bits = n.toString(2);
  let f = 0n; // F(k)
  let l = 2n; // L(k)
  let sign = 1n; // (-1)^k
  for (const bit of bits.slice(0, -1)) {
    const even = f * l; // F(2k)
    const lucas = l * l - 2n * sign; // L(2k)
    if (bit === '0') [f, l, sign] = [even, lucas, 1n];
    else [f, l, sign] = [(even + lucas) >> 1n, (5n * even + lucas) >> 1n, -1n];
  }
  // Dernier bit : seul F(n) est utile, donc ni L(n) ni, pour un bit à 0, le carré.
  const even = f * l;
  return bits.endsWith('0') ? even : (even + l * l - 2n * sign) >> 1n;
}

function checkIndex(n: number): void {
  if (!Number.isInteger(n) || n < 0) throw new RangeError(`indice invalide : ${n}`);
}

const BASE = 10_000; // limbes de 4 chiffres : une somme de deux limbes et d'une retenue tient dans un int32

// Écriture ASCII de 0000 à 9999, 4 octets chacune : un limbe indexe la table, sans division ni padStart.
function asciiTable(): Uint32Array {
  const table = new Uint32Array(BASE);
  const bytes = new Uint8Array(table.buffer);
  let p = 0;
  for (let d3 = 0x30; d3 <= 0x39; d3++)
    for (let d2 = 0x30; d2 <= 0x39; d2++)
      for (let d1 = 0x30; d1 <= 0x39; d1++)
        for (let d0 = 0x30; d0 <= 0x39; d0++) {
          bytes[p++] = d3;
          bytes[p++] = d2;
          bytes[p++] = d1;
          bytes[p++] = d0;
        }
  return table;
}

/** F(0) … F(n) en décimal, une chaîne par terme, toujours par l'algorithme iterative (deux termes en mémoire).
 * L'addition décimale s'écrit à la main sur des limbes de 4 chiffres : aucune conversion binaire → décimal. */
export function fibList(n: number): string[] {
  checkIndex(n);
  // log₁₀ φ < 0,209, donc ⌈0,209·n/4⌉ limbes suffisent à F(n). Le « + 1 » est un limbe d'assurance : un Int32Array
  // ignore sans erreur une écriture hors bornes, donc un tampon trop court corromprait la sortie en silence.
  const size = Math.ceil((n * 0.209) / 4) + 1;
  const a = new Int32Array(size); // F(i), limbes du poids faible au poids fort
  const b = new Int32Array(size); // F(i+1)
  b[0] = 1;
  const ascii = asciiTable();
  const words = new Uint32Array(size); // le dernier terme calculé en ASCII, aligné à droite
  const bytes = Buffer.from(words.buffer);
  let len = 1; // limbes du plus grand des deux termes, donc du dernier calculé
  // x += y en place, limbe par limbe. Les deux termes ont len limbes au plus; le reste est à zéro.
  // Écrit aussi la somme en ASCII dans words (même boucle) et fait passer len à len + 1 si la retenue sort.
  const addInPlaceAndEncode = (x: Int32Array, y: Int32Array): void => {
    let m = -1; // -1 sans retenue, 0 avec : sans branche, car la retenue est imprévisible (une fois sur deux)
    for (let k = 0; k < len; k++) {
      const u = x[k] + y[k] - (BASE - 1) + m; // somme + retenue - BASE
      m = u >> 31;
      x[k] = u + (BASE & m); // si la somme n'atteint pas BASE (m = -1), on la rend
      words[size - 1 - k] = ascii[x[k]];
    }
    if (m === 0) {
      x[len] = 1;
      words[size - 1 - len] = ascii[1];
      len++;
    }
  };
  // Le dernier terme encodé, en texte, sans les zéros de tête de son limbe de poids fort.
  const lastTermText = (): string => {
    let start = (size - len) * 4;
    while (bytes[start] === 0x30) start++;
    return bytes.toString('latin1', start, size * 4);
  };
  const terms = ['0', '1'].slice(0, n + 1);
  for (let i = 2; i <= n; i += 2) {
    addInPlaceAndEncode(a, b); // F(i)
    terms.push(lastTermText());
    if (i < n) {
      addInPlaceAndEncode(b, a); // F(i+1)
      terms.push(lastTermText());
    }
  }
  return terms;
}

// Exportée pour les seules gardes (PERF-TS-10, PERF-TS-12) : les tests du nom lisent la table, les espions de ces deux gardes y écrivent.
// Lecture seule par le type seulement (aucun coût à l'exécution) : une affectation `algos.matrix = …` est une erreur de
// `tsc`, et seuls ces espions la convertissent en `Record` pour écrire. Pas de `Object.freeze` : il ferait rougir les
// espions à tort. La seam du PRD § 5 reste fib et fibList.
export const algos: Readonly<Record<Algo, (n: number) => bigint>> = { iterative, matrix, fast };

/** Rend F(n) par l'algorithme algo.
 * Lève TypeError pour un algorithme inconnu, RangeError pour un indice qui n'est pas un entier >= 0. */
export function fib(n: number, algo: Algo): bigint {
  if (!Object.hasOwn(algos, algo)) throw new TypeError(`algorithme inconnu : ${algo}`);
  checkIndex(n);
  return algos[algo](n);
}

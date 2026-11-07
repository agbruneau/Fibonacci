// Lecture et vérification de l'oracle (b-file OEIS A000045). Aucun terme n'est calculé ici.
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const file = fileURLToPath(new URL('./b000045.txt', import.meta.url));

/** Lignes non vides du fichier, telles quelles. Lève si le fichier manque. */
export function lines(): string[] {
  if (!existsSync(file)) throw new Error('oracle absent : contracts/b000045.txt');
  return readFileSync(file, 'utf8').split('\n').filter((l) => l !== '');
}

const digits = (line: string) => line.split(' ')[1];

/** ORA-01 à ORA-06 : [identifiant, comportement, vérification]. */
export const CHECKS: [id: string, title: string, run: (l: string[]) => void][] = [
  ['ORA-01', 'le fichier contient 2 001 lignes non vides', (l) => assert.equal(l.length, 2001)],
  ['ORA-02', 'chaque ligne est « n F(n) » en chiffres ASCII, sans zéro de tête ni \\r', (l) =>
    l.forEach((x) => assert.match(x, /^[0-9]+ (0|[1-9][0-9]*)$/))],
  ['ORA-03', 'les indices vont de 0 à 2000, consécutifs et dans l’ordre', (l) =>
    l.forEach((x, i) => assert.equal(x.split(' ')[0], String(i)))],
  ['ORA-04', 'F(0) = 0 et F(1) = 1', (l) => assert.deepEqual(l.slice(0, 2).map(digits), ['0', '1'])],
  ['ORA-05', 'chaque terme est la somme des deux précédents', (l) => {
    const t = l.map((x) => BigInt(digits(x)));
    for (let i = 2; i < t.length; i++) assert.equal(t[i], t[i - 1] + t[i - 2], `n = ${i}`);
  }],
  ['ORA-06', 'F(2000) compte 418 chiffres', (l) => assert.equal(digits(l[2000]).length, 418)],
];

let cache: string[] | undefined;

/** F(0) … F(2000) en décimal. Un oracle absent ou corrompu lève : il ne sert jamais d'attendu. */
export function terms(): string[] {
  if (cache) return cache;
  const l = lines();
  for (const [id, , run] of CHECKS) {
    try {
      run(l);
    } catch (cause) {
      throw new Error(`oracle invalide : ${id} échoue`, { cause });
    }
  }
  return (cache = l.map(digits));
}

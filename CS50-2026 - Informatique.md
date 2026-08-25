# L'informatique, de la représentation au raisonnement

### Essai sur les fondements d'une discipline, d'après CS50 (Harvard, édition 2026)

**2026-08-25** · **André-Guy Bruneau**

---

## Avertissement de méthode

Cet essai est adossé à deux sources, et à deux seulement. La première est la compilation vidéo *Harvard CS50 (2026) – Full Computer Science University Course*, vingt-cinq heures publiées par freeCodeCamp, qui rassemble bout à bout les leçons magistrales de l'édition 2026 du cours. La seconde est le site officiel du cours, `cs50.harvard.edu/x`, qui en donne la structure canonique : onze semaines numérotées de 0 à 10, une semaine supplémentaire consacrée à l'intelligence artificielle, dix séries de problèmes et un projet final. L'instructeur est David J. Malan, qui enseigne CS50 depuis plus de vingt ans et dont le cours est devenu, par sa diffusion libre, la porte d'entrée en informatique la plus fréquentée du monde.

Une divergence mineure entre les deux sources mérite d'être signalée d'emblée, parce qu'elle éclaire la nature du matériel. Le site officiel de l'édition 2026 intitule la semaine 10 « The End » et la résume par deux mots, « Fun. Games. ». La compilation vidéo, elle, découpe son dernier tiers en une leçon nommée « Emoji » et une leçon nommée « Cybersecurity ». Ces titres proviennent d'éditions antérieures dont le montage a conservé les segments encore valides. Le fait est instructif : CS50 n'est pas un texte figé mais un cours refondu chaque année, où certaines leçons persistent, d'autres migrent, d'autres disparaissent. L'ajout d'une semaine entière sur l'intelligence artificielle dans l'édition 2026 en est le témoignage le plus net. Je traite donc les deux sources comme complémentaires plutôt que concurrentes, et je signale au passage les endroits où elles ne se recouvrent pas exactement.

Ce texte n'est pas un résumé du cours. Un résumé serait inutile : le cours existe, il est gratuit, il est mieux fait que n'importe quelle paraphrase. Ce que je propose est autre chose : suivre l'arc intellectuel que CS50 dessine — de la représentation d'un nombre en base deux jusqu'au raisonnement d'un modèle de langage — et l'exposer comme ce qu'il est réellement, à savoir un argument sur la nature de l'informatique. Car CS50 défend une thèse, et cette thèse n'est pas triviale. Elle tient en ceci : l'informatique n'est pas l'étude des ordinateurs, et la programmation n'en est qu'un instrument. L'informatique est la discipline qui prend au sérieux la question de savoir comment un problème peut être résolu par une procédure mécanique, ce que cette procédure coûte, et à quel prix on peut oublier comment elle fonctionne.

---

## I. Une boîte noire et deux flèches

Le cours s'ouvre sur un dessin d'une simplicité provocante : un rectangle, une flèche qui entre, une flèche qui sort. Entrée, boîte noire, sortie. Toute l'informatique, dit en substance ce dessin, consiste à remplir ce rectangle.

Il faut résister à la tentation de trouver cela naïf. Le schéma contient déjà trois décisions lourdes de conséquences.

La première est que le problème doit être formulé avant d'être résolu. Une entrée est une chose définie : elle a une forme, un domaine, des cas limites. Une sortie l'est aussi. Un problème dont on ne sait pas dire ce qui entre ni ce qui doit sortir n'est pas un problème d'informatique, c'est encore une intuition. Une part considérable du travail réel — et CS50 y insiste par la structure même de ses séries de problèmes, qui spécifient chaque comportement attendu jusqu'aux cas dégénérés — consiste à transformer un désir vague en une spécification. Cette transformation n'est pas un préliminaire ennuyeux à la vraie tâche; c'est souvent la vraie tâche.

La deuxième décision est que la boîte est noire. On peut l'utiliser sans l'ouvrir. C'est le principe d'abstraction, qui traverse le cours de bout en bout et que rien n'illustre mieux que la trajectoire pédagogique elle-même : on commence par assembler des blocs colorés dans Scratch sans savoir ce qu'est un octet, on descend ensuite jusqu'au pointeur et à l'allocation manuelle en C, puis on remonte vers Python où l'on redevient libre d'ignorer la mémoire — mais en sachant désormais ce qu'on ignore. L'abstraction n'est pas de l'ignorance; c'est de l'ignorance choisie, délimitée et réversible. La différence est capitale. Le programmeur qui ne sait pas ce qu'il y a sous une insertion en fin de liste, en Python, n'est pas dans la même situation que celui qui le sait et décide de ne pas y penser aujourd'hui. Le second peut descendre quand la performance s'effondre; le premier ne peut que changer de bibliothèque en espérant.

La troisième décision est que le contenu de la boîte est, en dernière analyse, mécanique. Ce qui s'y passe doit pouvoir être décrit sans appel à l'intelligence, à la bonne volonté ou au jugement de qui l'exécute. C'est la définition même de l'algorithme, et c'est ce qui rend la discipline possible : si la procédure doit être suivie sans discernement, alors on peut la faire suivre par une machine, et l'on peut aussi raisonner sur elle indépendamment de tout exécutant. La théorie de la complexité, dont nous parlerons, ne serait pas concevable autrement.

De ce schéma découle la définition de travail que le cours retient pour l'informatique : c'est l'étude de la résolution de problèmes. Non pas l'étude des machines. La machine est contingente. Le problème et la procédure ne le sont pas.

---

## II. Représenter : le premier acte

Avant de calculer quoi que ce soit, il faut décider comment les choses seront écrites. Cette décision précède tout le reste, et elle est plus arbitraire qu'on ne le croit.

### Compter avec ce qu'on a

Un être humain qui compte sur ses doigts utilise un système unaire : un doigt levé vaut une unité, et la quantité représentée est le nombre de doigts levés. Le système est immédiat et sans convention, mais il plafonne. Avec cinq doigts, on compte jusqu'à cinq.

Or la même main, exploitée différemment, compte jusqu'à trente et un. Il suffit de cesser de compter les doigts levés et de considérer plutôt *quels* doigts sont levés. Chaque doigt devient alors un chiffre binaire, zéro s'il est baissé, un s'il est levé, et la main entière représente un nombre de cinq bits. Rien n'a changé dans le matériel; tout a changé dans la convention. C'est la démonstration la plus économique du principe qui gouverne toute la représentation numérique : la puissance ne vient pas du support, elle vient de la façon dont on l'interprète.

Cette bascule de l'unaire au positionnel n'est pas propre au binaire. Le système décimal que nous utilisons quotidiennement obéit à la même logique : dans le nombre 123, le chiffre 1 ne vaut pas un, il vaut cent, parce qu'il occupe la colonne des centaines. Chaque colonne vaut la base élevée à la puissance de son rang. En base dix, les colonnes valent 1, 10, 100, 1000. En base deux, elles valent 1, 2, 4, 8, 16, 32, et ainsi de suite en doublant. Le nombre binaire 101 se lit donc quatre plus zéro plus un, soit cinq. Le nombre binaire 11111111, huit bits tous levés, vaut 255 — d'où la présence obsédante de ce chiffre partout en informatique.

Pourquoi le binaire plutôt qu'autre chose? Parce que la machine est électrique, et que la distinction la plus robuste qu'un circuit électrique puisse offrir est celle entre courant et absence de courant. On pourrait imaginer des machines ternaires, et l'on en a construit; elles ont perdu. Deux états se distinguent avec une marge de bruit confortable, trois beaucoup moins. Le binaire n'est donc pas une propriété mystique de l'information, c'est un choix d'ingénierie qui a gagné parce qu'il tolérait mieux l'imperfection du monde physique. Ce point mérite d'être retenu : plusieurs des « évidences » de l'informatique sont des accidents historiques stabilisés.

Le bit est l'unité; l'octet, huit bits, en est le groupement conventionnel, capable de représenter 256 valeurs distinctes. Le reste — kilo-octet, méga-octet, giga-octet, téra-octet — n'est qu'une échelle, avec la confusion durable entre les puissances de mille et les puissances de 1024 dont l'industrie n'est jamais sortie proprement.

### Les lettres n'existent pas

Voici l'affirmation qui, correctement comprise, désoriente le plus les débutants : l'ordinateur ne contient pas de lettres. Il ne contient que des nombres, et les lettres sont une convention plaquée sur ces nombres.

Cette convention porte un nom, ASCII, et elle est d'une banalité assumée : quelqu'un a décidé que 65 signifierait A, que 66 signifierait B, et ainsi de suite jusqu'à 90 pour Z; puis que 97 signifierait a, jusqu'à 122 pour z. Rien dans le nombre 65 ne le prédisposait à représenter la première lettre de l'alphabet latin. Cette décision est le résultat d'un comité, dans les années soixante, aux États-Unis.

Deux conséquences pratiques en découlent immédiatement, et le cours les exploite abondamment. D'abord, l'écart constant de 32 entre une majuscule et sa minuscule permet de convertir la casse par une simple addition ou soustraction — un fait qui n'a rien d'universel et qui ne survit pas au passage à d'autres alphabets. Ensuite, le fait que les chiffres eux-mêmes aient des codes ASCII — le caractère `0` vaut 48 — explique le piège classique où le caractère `5` et l'entier 5 sont deux choses différentes, et où les additionner naïvement donne un résultat absurde. Le caractère est un nombre qui désigne un symbole; l'entier est un nombre qui désigne une quantité. La machine ne distingue les deux que parce que le programme lui a dit comment interpréter la case mémoire.

ASCII, avec ses sept bits utiles, couvrait 128 symboles. C'était suffisant pour l'anglais et pour à peu près rien d'autre. Le français y perdait déjà ses accents, le grec son alphabet, le chinois toute chance. La réponse fut Unicode, qui n'assigne plus des nombres sur sept bits mais sur un espace assez vaste pour contenir, en principe, tous les systèmes d'écriture de l'humanité, vivants et morts, plus les symboles mathématiques, plus les émojis. Le sous-ensemble d'Unicode qui correspond à ASCII a conservé les mêmes valeurs, ce qui a assuré la compatibilité ascendante et scellé pour longtemps le privilège de l'alphabet latin non accentué.

Il faut mesurer ce que cela signifie. Le fait qu'un émoji s'affiche identiquement sur un téléphone à Montréal et sur un serveur à Séoul repose sur un registre partagé où quelqu'un a attribué un numéro à chaque pictogramme, et sur des tables de police qui savent dessiner ce numéro. Quand un caractère apparaît sous la forme d'un carré vide, c'est que la chaîne s'est rompue quelque part entre le numéro et le dessin. Les émojis à teinte de peau variable, que le cours utilise comme exemple, sont même des séquences : un pictogramme de base suivi d'un modificateur, deux points de code combinés à l'affichage. Ce qui apparaît à l'écran comme un signe unique est, en mémoire, une suite. C'est l'abstraction qui fonctionne exactement comme prévu — jusqu'au jour où l'on compte les caractères d'une chaîne et où le compte ne tombe pas juste.

### Voir en nombres

Les couleurs suivent la même logique. Le modèle RGB décompose chaque couleur en trois quantités : rouge, vert, bleu, chacune codée sur un octet, donc de 0 à 255. Le triplet (255, 0, 0) est un rouge saturé, (255, 255, 255) est du blanc, (0, 0, 0) du noir. Trois octets suffisent à désigner un peu plus de seize millions de teintes, ce qui dépasse confortablement la capacité de discrimination de l'œil humain.

Un pixel est un tel triplet. Une image est une grille de pixels. Une image est donc une grille de nombres, et rien d'autre. Cette réduction n'est pas une métaphore : c'est littéralement ce qu'on trouve dans un fichier bitmap, après un en-tête qui précise les dimensions et le format. C'est aussi ce qui rend possible tout le traitement d'image, y compris les filtres qu'une des séries de problèmes du cours fait implémenter. Passer une photo en niveaux de gris consiste à remplacer chaque triplet par trois valeurs identiques, généralement leur moyenne. La renverser horizontalement consiste à inverser l'ordre des pixels de chaque rangée. La flouter consiste à remplacer chaque pixel par la moyenne de ses voisins. Détecter des contours consiste à appliquer un opérateur — Sobel, par exemple — qui mesure la variation locale d'intensité et s'allume là où elle est brusque.

Ce sont là des manipulations arithmétiques élémentaires sur des tableaux de nombres. Elles ne deviennent impressionnantes que par le volume : une image de douze mégapixels contient trente-six millions d'octets à traiter, et une vidéo en contient autant par image, trente fois par seconde. La différence entre un filtre instantané et un filtre insupportable ne tient pas à l'ingéniosité de l'idée mais au nombre d'opérations par pixel — ce qui nous amène directement à la question du coût.

Et une vidéo, précisément, n'est qu'une succession d'images. Un son n'est qu'une suite de mesures d'amplitude prises à intervalles réguliers. La généralisation est complète : **tout ce qu'un ordinateur manipule est un nombre, et le sens de ce nombre est entièrement extérieur à lui**. Il réside dans la convention d'interprétation, c'est-à-dire dans le programme. Le même octet vaut 65, ou la lettre A, ou une nuance de gris, ou l'intensité d'un échantillon sonore, selon le code qui le lit. L'informatique commence à cette prise de conscience : il n'y a pas de données brutes, il n'y a que des données lues selon une convention.

---

## III. L'algorithme et son prix

Un algorithme est une suite finie d'instructions non ambiguës qui, appliquée à une entrée valide, produit la sortie voulue en un nombre fini d'étapes. La définition tient en une ligne et chacun de ses mots porte.

*Finie* : une procédure qui ne s'arrête jamais ne résout rien. *Non ambiguës* : chaque étape doit être exécutable sans jugement. *Entrée valide* : un algorithme a un domaine de définition, et ce qui arrive en dehors relève de la gestion d'erreur, pas de l'algorithme. *En un nombre fini d'étapes* : la terminaison n'est pas donnée, elle se démontre.

Mais un algorithme correct ne suffit pas. La question qui structure toute la discipline est la suivante : **combien coûte-t-il?** Deux algorithmes peuvent donner la même réponse, l'un en une seconde et l'autre en un siècle. La correction est une condition nécessaire; l'efficacité décide de l'utilisabilité.

### L'annuaire téléphonique

L'exemple que CS50 emploie depuis toujours est l'annuaire papier, et il reste imbattable parce que le lecteur peut faire l'expérience physiquement.

On cherche un nom dans un annuaire de mille pages, trié alphabétiquement. Trois stratégies.

La première tourne les pages une à une, de la première à la dernière. Dans le pire cas, mille tours de page. C'est la **recherche linéaire**.

La deuxième tourne les pages deux par deux, avec un recul d'une page si l'on a dépassé. Cinq cents tours de page dans le pire cas. Deux fois plus rapide.

La troisième ouvre l'annuaire au milieu, regarde la lettre, jette la moitié qui ne peut pas contenir le nom, recommence sur la moitié restante. C'est la **recherche binaire**, et elle traite mille pages en dix étapes.

L'écart mérite qu'on s'y arrête, parce qu'il ne s'agit pas d'une amélioration mais d'un changement de régime. Doubler l'annuaire ajoute mille tours de page à la première stratégie et *une seule* étape à la troisième. Un annuaire d'un million de pages : un million d'étapes contre vingt. Un annuaire d'un milliard : un milliard contre trente. La recherche binaire ne devient pas seulement meilleure quand le problème grossit; elle devient incommensurablement meilleure. C'est la première leçon durable de l'analyse algorithmique : ce qui compte n'est pas la vitesse sur un cas, c'est la façon dont le coût réagit à la taille.

Il faut noter au passage ce que la recherche binaire exige : que l'annuaire soit trié. Cette exigence n'est pas gratuite. Elle sera payée plus loin, au moment des algorithmes de tri, et c'est un motif récurrent — un coût de préparation qu'on amortit sur de nombreuses interrogations. Trier une fois pour chercher un million de fois est rentable; trier une fois pour chercher une fois ne l'est pas.

### Écrire avant de coder

Entre l'idée et le code, CS50 place systématiquement le **pseudocode** : la procédure écrite en langue naturelle, mais structurée comme du code. Chercher un nom devient une suite d'instructions numérotées, avec des conditions, des boucles et des points d'arrêt.

Ce détour n'est pas une politesse pédagogique. Il sépare deux difficultés que les débutants confondent : *savoir quoi faire* et *savoir comment le dire à la machine*. La quasi-totalité des blocages de débutant sont du premier type, mais se manifestent comme des erreurs du second. Un programmeur qui ne sait pas écrire son pseudocode ne sait pas ce qu'il veut faire, et aucune maîtrise syntaxique ne rattrapera cela.

Le pseudocode révèle en outre la structure logique universelle qu'on retrouvera dans tous les langages du cours, de Scratch à Python : des **fonctions** (des actions nommées), des **conditions** (des embranchements), des **expressions booléennes** (des questions à réponse binaire), des **boucles** (des répétitions), des **variables** (des valeurs mémorisées). Cinq constructions. Tout le reste en est un raffinement.

### La notation asymptotique

Pour raisonner sur le coût sans se noyer dans les détails, l'informatique a inventé un langage : la notation en grand O.

L'idée est de décrire comment le nombre d'opérations croît avec la taille de l'entrée, en négligeant deux choses : les constantes multiplicatives et les termes d'ordre inférieur. Un algorithme qui exécute `3n + 17` opérations est dit en O(n), exactement comme un algorithme qui en exécute `n`. Cela peut sembler cavalier — un facteur trois compte, dans la vraie vie. Mais l'abstraction est justifiée : quand n devient grand, le facteur trois est écrasé par la différence entre n et n². Et la machine sur laquelle on exécute change les constantes sans changer la forme. On abstrait la machine pour ne garder que l'algorithme.

Les classes que le cours retient sont les suivantes.

| Notation | Nom | Comportement | Exemple |
|---|---|---|---|
| O(1) | constante | insensible à la taille | accéder au i-ième élément d'un tableau |
| O(log n) | logarithmique | double la taille, ajoute une étape | recherche binaire |
| O(n) | linéaire | double la taille, double le coût | recherche linéaire |
| O(n log n) | linéarithmique | optimum du tri par comparaison | tri par fusion |
| O(n²) | quadratique | double la taille, quadruple le coût | tri à bulles, tri par sélection |
| O(2ⁿ) | exponentielle | inutilisable au-delà de quelques dizaines | énumération de tous les sous-ensembles |
| O(n!) | factorielle | inutilisable au-delà de dix ou douze | énumération de toutes les permutations |

Trois notations coexistent, et le cours prend soin de les distinguer. **O** majore : c'est la borne supérieure, le pire cas. **Ω** (oméga) minore : c'est la borne inférieure, le meilleur cas. **Θ** (thêta) s'emploie quand les deux coïncident, c'est-à-dire quand le comportement est le même quoi qu'il arrive.

La distinction n'est pas scolastique. La recherche linéaire est en O(n) — le nom cherché peut être le dernier — mais en Ω(1), car il peut être le premier. Le tri par sélection, lui, est en Θ(n²) : il compare tout avec tout, que le tableau soit déjà trié ou dans le désordre le plus complet. Cette insensibilité est un défaut, et c'est précisément ce qui distingue un algorithme rigide d'un algorithme adaptatif. Le tri à bulles, doté d'un simple compteur d'échanges qui permet de s'arrêter tôt, est en O(n²) mais en Ω(n) : sur un tableau déjà trié, il fait une passe et s'arrête. La même famille d'algorithmes, une amélioration d'une ligne, un comportement radicalement différent sur les entrées faciles.

### Trois façons de trier

Le tri est le terrain d'exercice canonique de l'analyse algorithmique, parce qu'il est simple à énoncer, utile en pratique et qu'il admet des solutions de qualités très inégales.

**Le tri par sélection** procède ainsi : parcourir tout le tableau pour trouver le plus petit élément, l'échanger avec le premier; recommencer sur ce qui reste. C'est intuitif et c'est mauvais. Le nombre de comparaisons est n−1, puis n−2, puis n−3, et ainsi de suite jusqu'à 1 — soit n(n−1)/2 au total, donc de l'ordre de n². Il n'a aucune chance de bien se comporter, jamais : Θ(n²).

**Le tri à bulles** procède autrement : comparer chaque paire d'éléments voisins et les échanger s'ils sont dans le mauvais ordre; recommencer jusqu'à ce qu'une passe complète n'ait produit aucun échange. Les grandes valeurs remontent vers la fin comme des bulles, d'où le nom. Le coût est aussi de l'ordre de n², mais la condition d'arrêt lui donne son Ω(n) sur les entrées déjà triées.

**Le tri par fusion** change de stratégie. Au lieu d'améliorer le balayage, il divise. Trier un tableau, c'est trier sa moitié gauche, trier sa moitié droite, puis fusionner les deux moitiés triées. La fusion est facile : on compare les deux premiers éléments restants, on prend le plus petit, on avance. Le coût de la fusion est linéaire — chaque élément est touché une fois. Et le nombre de niveaux de division est logarithmique, puisqu'on divise par deux à chaque fois. D'où le O(n log n).

L'écart pratique est spectaculaire. Sur un million d'éléments, n² vaut mille milliards d'opérations; n log n en vaut vingt millions. Le rapport est de cinquante mille. Une opération qui prend une seconde avec le tri par fusion prend quatorze heures avec le tri par sélection.

Le tri par fusion paie ce gain d'une façon qu'il faut noter : il a besoin de mémoire supplémentaire pour la fusion, contrairement aux deux autres qui trient sur place. C'est la première apparition explicite du **compromis temps-espace**, qui reviendra à chaque structure de données du cours. On échange presque toujours de la mémoire contre de la vitesse, ou l'inverse. Il n'existe pas de repas gratuit; il n'existe que des dépenses qu'on choisit.

Un dernier point théorique mérite d'être posé, même si le cours ne le démontre pas : n log n est une **borne inférieure** pour tout tri fondé sur des comparaisons. Aucun algorithme de cette famille ne fera mieux, jamais. Ce n'est pas un constat empirique, c'est un théorème — un tri par comparaisons doit distinguer n! ordres possibles, chaque comparaison ne fournit qu'un bit, il faut donc au moins log₂(n!) comparaisons, ce qui vaut de l'ordre de n log n. Voir apparaître ce genre d'énoncé est important : l'informatique ne dit pas seulement ce qu'on sait faire, elle démontre parfois ce qu'on ne pourra jamais faire.

### La récursivité et la pile

Le tri par fusion introduit une idée qui dépasse largement le tri : une fonction qui s'appelle elle-même.

L'idée choque d'abord — comment une chose peut-elle se contenir? — puis devient évidente quand on voit qu'elle a besoin de deux composantes. Un **cas de base**, qui se résout sans récursion : un tableau d'un seul élément est déjà trié. Et un **cas récursif**, qui réduit strictement le problème avant de se rappeler. Sans cas de base, on obtient une descente infinie. Sans réduction stricte, aussi.

La récursivité est un gain d'expression considérable. Un parcours d'arbre s'écrit en trois lignes récursives et en trente lignes itératives, et les trois lignes sont plus faciles à vérifier. Mais elle a un coût matériel, et CS50 tient à le montrer plutôt qu'à le taire : chaque appel de fonction consomme un **cadre de pile**, un bloc de mémoire qui contient les paramètres, les variables locales et l'adresse de retour. Les cadres s'empilent. Quand la fonction retourne, son cadre est dépilé et l'exécution reprend là où elle s'était arrêtée.

Cette pile est finie. Une récursion trop profonde la déborde, et le programme meurt d'un *stack overflow* — le débordement de pile qui a donné son nom au site où les programmeurs vont chercher de l'aide. La leçon est constante dans tout le cours : **l'élégance conceptuelle a une contrepartie physique**, et il faut savoir laquelle avant de choisir.

---

## IV. Scratch, ou l'art de retirer la syntaxe

La semaine 0 de CS50 se programme en Scratch, un langage visuel conçu au MIT où l'on assemble des blocs colorés qui s'emboîtent comme des pièces de casse-tête. Le choix surprend dans un cours universitaire, et il est délibéré.

### Ce que Scratch supprime

Scratch supprime la syntaxe. Il n'y a ni point-virgule oublié, ni accolade mal fermée, ni faute de frappe dans un nom de fonction. Les blocs ne s'emboîtent que s'ils sont compatibles : un bloc qui attend un nombre refuse un bloc qui produit un booléen. Les erreurs de forme sont rendues impossibles par la géométrie.

L'effet est immédiat et il est mesurable dans la pédagogie du cours : deux tiers des étudiants de CS50 n'ont jamais fait d'informatique auparavant, et le taux d'abandon dans les premières heures d'un cours de programmation est classiquement dominé par la frustration syntaxique. En retirant cette frustration, Scratch permet de consacrer la première semaine entière aux idées plutôt qu'à la ponctuation.

### Ce que Scratch conserve

Or ce qui reste, une fois la syntaxe retirée, est exactement l'ensemble des concepts du cours entier. Le site officiel les énumère pour la semaine 0 : fonctions, arguments, valeurs de retour, variables, expressions booléennes, conditions, boucles, événements et fils d'exécution.

Ce sont là les primitives universelles. Un programme Scratch qui fait dire bonjour à un chat quand on clique sur le drapeau vert contient, en miniature, tout ce qu'un serveur web contient : un gestionnaire d'événement, une condition, un appel de fonction avec argument. La différence entre les deux n'est pas de nature, elle est d'échelle et de rigueur.

Les **événements** et les **fils d'exécution** méritent une mention particulière, parce qu'ils sont rarement enseignés si tôt. Dans Scratch, plusieurs lutins s'exécutent simultanément, chacun avec son propre script, et ils communiquent par diffusion de messages. C'est un modèle concurrent, exposé sans le vocabulaire ni les pièges — mais exposé. Quand la semaine 7 parlera de conditions de course sur une base de données, l'intuition aura déjà été installée : des choses peuvent se produire en même temps, et l'ordre n'est pas garanti.

### L'abstraction comme geste

Le geste central que Scratch enseigne est la création de blocs personnalisés. Quand la même séquence de dix blocs revient trois fois dans un projet, on la remplace par un bloc unique auquel on donne un nom. Le programme rétrécit et devient lisible.

C'est la définition opérationnelle de l'abstraction : nommer une procédure pour cesser d'avoir à penser à son contenu. Toute l'histoire des langages de programmation est la répétition de ce geste à des échelles croissantes. L'assembleur nomme des séquences d'instructions machine. C nomme des séquences d'assembleur. Une bibliothèque nomme des séquences de C. Un service web nomme des séquences d'appels de bibliothèque. À chaque étage, on gagne en expressivité et on perd en contrôle. C'est le marché fondamental de l'informatique, et il n'a pas de terme optimal : il a des points d'équilibre, différents selon ce qu'on construit.

---

## V. Descendre au métal : C

Les semaines 1 à 5 se passent en C, un langage de 1972 dont la syntaxe a essaimé partout — Java, C++, C#, JavaScript, PHP, Go, Rust en portent tous la marque. Le choix, là encore, est pédagogique avant d'être pratique.

C est un langage **transparent**. Il ne cache presque rien. Il n'y a ni ramasse-miettes, ni vérification de bornes de tableau, ni gestion automatique des chaînes de caractères. Ce qu'on écrit correspond de près à ce que la machine exécutera. Apprendre C, c'est apprendre à voir la machine, et cette vision ne se désapprend pas : elle continue d'informer le jugement quand on écrit du Python dix ans plus tard.

### Du texte au courant électrique

Un ordinateur n'exécute pas du texte. Il exécute des instructions binaires, propres à son architecture de processeur. Le passage de l'un à l'autre est le travail du **compilateur**, et il comporte quatre étapes que CS50 détaille en semaine 2.

**Le préprocesseur** traite d'abord les lignes qui commencent par un dièse. Il remplace chaque directive d'inclusion par le contenu du fichier d'en-tête désigné, substitue les constantes symboliques, résout les compilations conditionnelles. Le résultat est encore du C, mais un C dilaté où toutes les déclarations nécessaires sont présentes. Il faut comprendre qu'un fichier d'en-tête ne contient pas le code des fonctions : il contient leurs *signatures*, c'est-à-dire leur nom, le type de leurs paramètres et le type de leur valeur de retour. C'est ce dont le compilateur a besoin pour vérifier que l'appel est bien formé; le code proprement dit viendra plus tard.

**La compilation** proprement dite traduit ensuite ce C dilaté en langage d'assemblage, une représentation textuelle des instructions du processeur. C'est ici que se joue l'essentiel du travail intellectuel du compilateur : analyse lexicale, analyse syntaxique, construction de l'arbre de syntaxe abstraite, vérification des types, puis génération de code et optimisation. Un compilateur moderne réorganise les boucles, élimine le code mort, place les variables dans des registres plutôt qu'en mémoire, déroule les itérations courtes. Le programme qui sort n'est plus une traduction ligne à ligne du programme qui est entré.

**L'assemblage** convertit ce langage d'assemblage en code machine binaire. La correspondance est ici presque bijective — chaque instruction d'assemblage devient une instruction binaire. Le produit est un fichier objet.

**L'édition de liens** rassemble enfin les fichiers objets. Le programme utilise une fonction mathématique? Le code de cette fonction vit dans une bibliothèque compilée séparément. L'éditeur de liens la trouve, la copie ou en enregistre la référence, résout toutes les adresses, et produit l'exécutable final.

Savoir cela n'est pas une curiosité. C'est ce qui permet de lire un message d'erreur. Une erreur du préprocesseur signale un en-tête introuvable. Une erreur du compilateur signale une faute de syntaxe ou de type. Une erreur de l'éditeur de liens — la fameuse *undefined reference* — signale qu'on a déclaré une fonction sans jamais fournir son code, ou qu'on a oublié de lier la bibliothèque qui la contient. Le débutant qui ne connaît pas ces quatre étapes voit quatre familles d'erreurs comme une seule masse hostile. Celui qui les connaît sait immédiatement où chercher.

### Les types, ou la taille des boîtes

En C, chaque variable a un type déclaré, et ce type détermine combien d'octets elle occupe et comment ces octets sont interprétés. Un entier occupe typiquement quatre octets, un caractère un seul, un flottant en double précision huit.

Cette rigidité, que les langages modernes ont largement abandonnée, a une vertu pédagogique décisive : elle rend visible le fait que **la mémoire est finie et que les nombres qu'on y range le sont aussi**.

Un entier signé de quatre octets couvre l'intervalle allant d'environ moins deux milliards à plus deux milliards. Que se passe-t-il si on dépasse? Rien de spectaculaire : le compteur repasse par l'autre extrémité, comme un odomètre de voiture qui revient à zéro. Un très grand nombre positif, augmenté de un, devient un très grand nombre négatif. C'est le **débordement d'entier**, et il est silencieux.

CS50 cite deux exemples pour ancrer le concept. Le bogue de l'an 2000, d'abord, où des décennies de programmes avaient stocké l'année sur deux chiffres et où le passage de 99 à 00 menaçait de faire reculer les systèmes d'un siècle. Et un défaut découvert sur le Boeing 787, où un compteur interne débordait après un peu plus de deux cent quarante-huit jours de fonctionnement continu, ce qui pouvait provoquer une perte de courant alternatif en vol; le correctif provisoire consistait à redémarrer périodiquement les systèmes. Ces exemples ne sont pas des anecdotes : ils montrent qu'une limite arithmétique invisible dans le code source peut avoir des conséquences physiques.

Le problème n'a d'ailleurs pas disparu avec le siècle. Les systèmes Unix comptent le temps en secondes écoulées depuis le 1er janvier 1970, et un compteur signé de trente-deux bits déborde le 19 janvier 2038. Le correctif — passer à soixante-quatre bits — est simple et connu; le déployer partout où du code embarqué persiste ne l'est pas.

### Ce que 0,1 n'est pas

L'imprécision en virgule flottante est le second piège numérique, et il est plus insidieux parce qu'il ne concerne pas les cas extrêmes mais l'usage ordinaire.

Un nombre à virgule flottante est stocké sur un nombre fini de bits, répartis entre un signe, un exposant et une mantisse, selon la norme IEEE 754. Cette représentation est binaire, et un nombre décimal simple n'a pas nécessairement d'écriture binaire finie. Un tiers n'a pas d'écriture décimale finie; un dixième n'a pas d'écriture binaire finie. La machine stocke donc une approximation, la plus proche valeur représentable.

D'où le résultat qui déconcerte tout le monde une fois : additionner un dixième et deux dixièmes ne donne pas exactement trois dixièmes, mais une valeur qui en diffère au seizième chiffre après la virgule. Afficher le résultat avec vingt décimales rend l'écart visible. Le cours fait précisément cette démonstration, et elle est salutaire.

Les conséquences pratiques sont de trois ordres. D'abord, **on ne compare jamais deux flottants par égalité stricte** : on vérifie que leur écart est inférieur à une tolérance. Ensuite, **on n'accumule pas des flottants sans y penser** : additionner un million de petites erreurs produit une grande erreur, et l'ordre des opérations change le résultat, ce qui viole l'associativité qu'on croyait acquise depuis l'école primaire. Enfin, et surtout, **on ne stocke jamais de l'argent en flottant** : on utilise des entiers de cents, ou un type décimal exact. Une application financière qui perd un centième de cent par transaction en perd beaucoup à la fin de l'année, et l'audit sera difficile.

### Correction, conception, style

CS50 note les travaux sur trois axes, et cette grille est plus qu'un barème : c'est une thèse sur ce qu'est un bon programme.

**La correction** demande si le programme fait ce qu'il doit faire, sur toutes les entrées prévues, y compris les cas limites — le tableau vide, la chaîne vide, la valeur zéro, la valeur maximale, l'entrée mal formée. C'est le critère non négociable. Un programme élégant et faux ne vaut rien.

**La conception** demande si le programme est bien construit. Y a-t-il de la duplication qui aurait dû devenir une fonction? Un algorithme quadratique là où un linéaire suffisait? Une structure de données mal choisie? Une boucle qui recalcule à chaque tour ce qui ne change pas? La conception est le critère le plus difficile à enseigner, parce qu'il n'admet pas de règle mécanique : c'est du jugement, et le jugement s'acquiert en voyant beaucoup de code.

**Le style** demande si le programme est lisible. Indentation cohérente, noms significatifs, commentaires là où l'intention n'est pas évidente, longueur de fonction raisonnable. Le style paraît cosmétique et ne l'est pas : le code est lu bien plus souvent qu'il n'est écrit, et la personne qui le lira dans six mois sera probablement soi-même, sans aucun souvenir du contexte.

L'ordre de ces trois critères est significatif. La correction d'abord, parce qu'elle est binaire et qu'elle prime. La conception ensuite, parce qu'elle détermine ce que coûtera la prochaine modification. Le style enfin, parce qu'il détermine si la prochaine modification sera seulement possible.

---

## VI. Tableaux, chaînes et le premier secret

### Le tableau, structure primitive

Un tableau est une suite d'éléments de même type rangés **contigus** en mémoire. Cette contiguïté n'est pas un détail d'implémentation : c'est la propriété qui donne au tableau son unique superpouvoir.

Puisque tous les éléments ont la même taille et qu'ils se suivent, l'adresse du i-ième élément se calcule par une multiplication et une addition à partir de l'adresse du premier. L'accès est donc en O(1), quelle que soit la position et quelle que soit la taille du tableau. Accéder au millionième élément coûte exactement ce que coûte l'accès au premier.

Ce superpouvoir se paie de deux façons. D'abord, la taille est fixée à la création : agrandir un tableau signifie en allouer un plus grand ailleurs et y recopier tout le contenu, opération en O(n). Ensuite, l'insertion au milieu exige de décaler tout ce qui suit, également en O(n). Le tableau est excellent pour lire n'importe où et mauvais pour se transformer. Toute la semaine 5 du cours consistera à construire des structures qui font le contraire.

C, par ailleurs, ne vérifie pas les bornes. Écrire à l'indice cinquante d'un tableau de dix éléments ne produit aucune erreur à la compilation et souvent aucune erreur à l'exécution : on écrit simplement dans la mémoire qui suit, laquelle appartient à autre chose. Le programme continue, corrompu. Ce comportement, indéfendable du point de vue de la sûreté, est la source directe d'une des familles de vulnérabilités les plus exploitées de l'histoire de l'informatique, et nous y reviendrons.

### La chaîne de caractères n'existe pas non plus

En C, il n'y a pas de type chaîne. Une chaîne est un tableau de caractères terminé par un octet de valeur zéro, appelé le caractère nul.

Cette convention explique tout le comportement des chaînes en C. La longueur d'une chaîne n'est stockée nulle part : la calculer exige de parcourir les octets jusqu'au premier zéro, opération en O(n). D'où le piège classique de la boucle qui appelle la fonction de longueur à chaque itération et transforme un parcours linéaire en parcours quadratique — une des erreurs de conception que CS50 signale explicitement. D'où aussi le fait qu'une chaîne de cinq caractères occupe six octets, l'oubli du sixième étant une source inépuisable de corruption mémoire.

Le cours introduit d'abord les chaînes derrière un type simplifié fourni par sa propre bibliothèque, puis retire ce voile en semaine 4 pour révéler qu'il ne s'agissait que d'un pointeur vers le premier caractère. Le geste pédagogique est remarquable : on donne une abstraction, on l'utilise, puis on la démonte. C'est exactement le rapport qu'un professionnel entretient avec les abstractions dont il dépend.

### Les arguments de la ligne de commande

Un programme reçoit ses paramètres par la fonction principale, qui peut déclarer deux arguments : le nombre de mots tapés sur la ligne de commande, et le tableau de ces mots. Le premier mot est toujours le nom du programme lui-même.

Ce mécanisme, d'apparence anodine, est la première rencontre de l'étudiant avec l'idée qu'un programme est un objet du système d'exploitation, qui reçoit des entrées, produit des sorties, et renvoie un **code de retour** — zéro pour le succès, autre chose pour l'échec. C'est la convention sur laquelle repose l'ensemble de l'outillage Unix : la possibilité d'enchaîner des programmes, de les arrêter au premier échec, de les scripter. Le cours fait travailler ses étudiants en ligne de commande précisément pour installer cette culture, où un ordinateur se pilote par composition de petits outils plutôt que par clics dans une interface.

### Cryptographie élémentaire

La semaine 2 introduit le chiffrement, et elle le fait par les deux systèmes historiques les plus simples.

**Le chiffre de César** décale chaque lettre d'un nombre fixe de positions dans l'alphabet. La clé est ce nombre. Le déchiffrement est le décalage inverse. Sa faiblesse est totale : il n'y a que vingt-cinq clés possibles, on les essaie toutes en un instant. Même sans cela, l'analyse des fréquences le brise immédiatement, puisqu'il préserve la distribution statistique des lettres — la lettre la plus fréquente du texte chiffré correspond à la lettre la plus fréquente de la langue.

**Le chiffre de Vigenère** utilise un mot-clé et fait varier le décalage lettre par lettre. Il résiste mieux, mais pas longtemps : dès qu'on devine la longueur du mot-clé, on décompose le texte en autant de chiffres de César indépendants, et chacun tombe par analyse des fréquences.

L'intérêt pédagogique de ces deux systèmes est double. Ils installent d'abord la structure conceptuelle de tout chiffrement : un texte clair, une clé, un algorithme, un texte chiffré, et la propriété que la connaissance de l'algorithme sans la clé ne doit rien donner — c'est le principe de Kerckhoffs, qui veut que la sécurité repose sur le secret de la clé et jamais sur le secret de la méthode. Ils démontrent ensuite, par leur faiblesse même, que l'intuition est un mauvais guide en cryptographie. Ces systèmes paraissaient solides à leurs inventeurs. Ils étaient brisés.

C'est la raison pour laquelle la règle professionnelle, en cryptographie, est de **ne jamais concevoir son propre algorithme**. On utilise des primitives standardisées, publiées, attaquées pendant des décennies par des spécialistes, et implémentées par des bibliothèques auditées. La créativité est ici un défaut.

---

## VII. La mémoire, ou ce qui se passe vraiment

La semaine 4 est le pivot du cours. Elle retire les voiles posés jusque-là et expose la mémoire telle qu'elle est.

### Les adresses

La mémoire vive est une immense suite d'octets, chacun repéré par un numéro appelé son **adresse**. Ces adresses s'écrivent traditionnellement en hexadécimal, base seize, avec les chiffres de 0 à 9 puis les lettres de A à F — une notation compacte où chaque chiffre représente exactement quatre bits, donc chaque paire de chiffres exactement un octet. La correspondance étant simple, la conversion se fait de tête après un peu de pratique, ce qui n'est pas le cas du décimal.

Une variable est un nom donné à un emplacement. Ce nom n'existe que dans le code source; à l'exécution, il n'y a que des adresses. C, contrairement à presque tous les langages qui l'ont suivi, permet de manipuler ces adresses directement.

### Les pointeurs

Un **pointeur** est une variable dont la valeur est une adresse. C'est tout. La difficulté n'est pas conceptuelle, elle est notationnelle : deux opérateurs, l'un qui prend l'adresse d'une variable, l'autre qui accède au contenu d'une adresse, et une syntaxe de déclaration qui utilise le même symbole que le second dans un sens différent.

Une fois passé cet obstacle, les pointeurs répondent à des besoins précis.

Ils permettent d'abord de **modifier un argument**. En C, les arguments sont passés par copie : une fonction qui reçoit un entier reçoit un duplicata, et le modifier ne change rien chez l'appelant. C'est pourquoi la fonction d'échange de deux valeurs, écrite naïvement, ne fonctionne pas — l'exemple que CS50 utilise pour introduire le sujet. Passer les adresses permet à la fonction d'atteindre les originaux.

Ils permettent ensuite d'**éviter les copies coûteuses**. Passer une grande structure par valeur signifie la recopier entièrement à chaque appel. Passer son adresse coûte huit octets.

Ils permettent enfin de **construire des structures liées** — et c'est le besoin décisif, celui qui ouvre la semaine 5. Sans pointeur, on ne peut pas relier un bloc de mémoire à un autre, et donc on ne peut construire ni liste chaînée, ni arbre, ni graphe.

Le prix de cette puissance est le **défaut de segmentation** : déréférencer un pointeur qui ne pointe pas sur une zone valide fait tuer le programme par le système d'exploitation. Les causes habituelles sont le pointeur non initialisé, qui contient des ordures; le pointeur nul, qu'on a oublié de tester; et le pointeur pendant, qui désigne une zone déjà libérée. Cette dernière est la pire, parce que le programme ne meurt pas forcément : il lit ou écrit dans une zone qui a pu être réattribuée à autre chose, et le désordre se manifeste ailleurs, plus tard, sans rapport apparent avec sa cause.

### Pile et tas

La mémoire d'un programme en cours d'exécution est organisée en régions, dont deux nous intéressent.

**La pile** contient les cadres d'appel de fonction : paramètres, variables locales, adresse de retour. Elle est gérée automatiquement — un cadre est empilé à l'appel, dépilé au retour. Elle est rapide, parce qu'allouer revient à déplacer un pointeur. Elle est petite, typiquement quelques mégaoctets. Et surtout, **ce qu'elle contient disparaît au retour de la fonction**, ce qui interdit de renvoyer l'adresse d'une variable locale : l'appelant recevrait un pointeur vers un cadre déjà dépilé.

**Le tas** contient la mémoire allouée explicitement à l'exécution. Il est grand, limité par la mémoire disponible. Il est plus lent, parce que l'allocateur doit chercher un bloc libre de taille suffisante et tenir sa comptabilité. Et il est **entièrement à la charge du programmeur** : ce qui est alloué doit être libéré, exactement une fois.

Les deux régions croissent l'une vers l'autre depuis les extrémités opposées de l'espace d'adressage. Quand elles se rencontrent, le programme meurt.

L'allocation dynamique répond à une nécessité que le tableau à taille fixe ne peut satisfaire : **on ne connaît pas toujours la taille à l'avance**. Combien de lignes dans le fichier que l'utilisateur va fournir? Combien de mots dans le dictionnaire? Combien de connexions simultanées? Aucune constante écrite dans le code ne répond correctement à ces questions : trop petite, elle plante; trop grande, elle gaspille.

### Les trois péchés de la gestion mémoire

La gestion manuelle engendre trois catégories de fautes, et elles sont responsables d'une proportion écrasante des vulnérabilités logicielles.

**La fuite** : on alloue et on ne libère jamais. Le programme grossit indéfiniment. Un utilitaire qui s'exécute en une seconde s'en tire; un serveur qui tourne pendant des mois finit par épuiser la machine. La fuite est sournoise parce qu'elle ne produit aucun symptôme jusqu'à ce qu'elle en produise un catastrophique.

**La double libération** : on libère deux fois le même bloc. Les structures internes de l'allocateur sont corrompues, et le désordre se manifeste à un moment arbitraire, ailleurs dans le programme.

**L'usage après libération** : on continue d'utiliser un pointeur vers une zone libérée. C'est la faute la plus dangereuse, car un attaquant qui contrôle ce qui sera alloué dans l'intervalle contrôle ce que le programme lira.

CS50 fait installer un outil d'analyse dynamique — Valgrind — qui instrumente l'exécution et rapporte ces trois catégories de fautes. C'est la première rencontre de l'étudiant avec l'idée qu'**on ne juge pas de la correction d'un programme en le regardant**, mais en l'instrumentant. Cette idée fondera plus tard toute la culture des tests, du profilage et de l'analyse statique.

### Le débordement de tampon

Puisque C ne vérifie pas les bornes, écrire au-delà de la fin d'un tableau alloué sur la pile écrase ce qui suit dans le cadre d'appel. Or ce qui suit comprend, dans les dispositions classiques, l'adresse de retour de la fonction.

Un attaquant qui contrôle les données écrites contrôle donc l'adresse vers laquelle le programme sautera au retour de la fonction. En plaçant du code de son choix dans le tampon et en faisant pointer l'adresse de retour vers ce code, il obtient l'exécution arbitraire. C'est le **débordement de tampon**, décrit publiquement dans les années quatre-vingt-dix et exploité massivement depuis.

Les défenses modernes en ont considérablement réduit la portée : des valeurs sentinelles placées avant l'adresse de retour et vérifiées à la sortie, la randomisation de la disposition de l'espace d'adressage, le marquage de la pile comme non exécutable. Mais aucune de ces défenses ne supprime la cause, qui est qu'un tableau n'a pas de borne vérifiée. C'est pourquoi les langages conçus depuis vérifient les bornes par défaut, et pourquoi les organisations qui écrivent du logiciel critique migrent vers des langages à sûreté mémoire.

Il faut retenir la structure de l'argument, parce qu'elle est générale : **une décision de conception prise en 1972 pour des raisons de performance a produit, cinquante ans plus tard, une classe entière de vulnérabilités**. Les choix d'architecture ont une durée de vie qui dépasse largement l'horizon de ceux qui les font.

### Fichiers et images

La semaine 4 se termine sur les entrées-sorties fichier, et le pont avec la semaine 0 se referme.

Un fichier est une suite d'octets sur un support persistant. On l'ouvre en obtenant un pointeur de fichier, on lit ou on écrit par blocs, on ferme. Le format du fichier n'est rien d'autre qu'une convention sur la signification de ces octets — exactement comme pour les caractères et les couleurs.

Un fichier bitmap commence par des en-têtes qui donnent le type, la taille, les dimensions et la profondeur de couleur, puis contient les pixels rangés par rangées, avec un remplissage pour aligner chaque rangée sur une frontière de quatre octets. Connaître ce format suffit pour écrire un programme qui applique des filtres — et c'est ce que la série de problèmes demande.

Une autre série demande de récupérer des images JPEG effacées sur une carte mémoire. L'exercice est plus subtil qu'il n'y paraît, et il enseigne une vérité que le grand public ignore : **effacer un fichier ne l'efface pas**. Le système d'exploitation retire l'entrée du répertoire et marque les blocs comme réutilisables, mais les octets restent en place jusqu'à ce que quelque chose les écrase. Un programme qui parcourt le support brut, reconnaît les signatures de début de fichier JPEG et reconstitue les images n'a besoin d'aucun privilège particulier.

Les conséquences pratiques sont considérables. Un disque revendu, un téléphone donné, une clé USB jetée contiennent probablement encore ce qu'on croyait avoir supprimé. La seule suppression fiable est le chiffrement du support dès l'origine, suivi de la destruction de la clé — ou la destruction physique.

---

## VIII. Structures de données : organiser pour aller vite

La semaine 5 rassemble tout ce qui précède. Elle utilise les pointeurs de la semaine 4 et l'analyse de complexité de la semaine 3 pour construire des façons d'organiser les données, chacune avec un profil de coût différent.

### Type abstrait et implémentation

La distinction préalable est celle entre le **type abstrait de données** et son **implémentation**.

Un type abstrait est défini par ses opérations et par leur sémantique, sans rien dire de la façon dont elles sont réalisées. Une pile est ce qui offre l'empilement et le dépilement avec la discipline « dernier entré, premier sorti ». Une file est ce qui offre l'enfilement et le défilement avec la discipline « premier entré, premier sorti ». Ces définitions ne mentionnent aucune mémoire.

Une implémentation choisit une réalisation concrète. Une pile peut être un tableau avec un indice de sommet, ou une liste chaînée où l'on insère toujours en tête. Les deux respectent le contrat; leurs coûts diffèrent.

Cette séparation est l'abstraction de la semaine 0 appliquée aux données plutôt qu'aux procédures. Elle permet de raisonner sur un programme en termes de ce qu'il fait, puis de changer l'implémentation sans toucher au reste. C'est le principe qui fonde l'ensemble du génie logiciel : programmer contre une interface, pas contre une réalisation.

### Piles et files

**La pile** est la structure de la récursion, de l'annulation, du retour en arrière. Le navigateur qui revient à la page précédente dépile. L'éditeur qui défait la dernière action dépile. Le processeur qui retourne d'une fonction dépile. La discipline « dernier entré, premier sorti » correspond à tout ce qui a une structure d'imbrication.

**La file** est la structure de l'équité et de l'ordonnancement. La file d'impression, la file d'attente d'un service, le tampon d'un flux réseau. La discipline « premier entré, premier sorti » correspond à tout ce qui doit respecter l'ordre d'arrivée.

Les deux se réalisent en O(1) par opération si l'implémentation est bien choisie. Pour la file, un tableau naïf pose problème : défiler en tête exige de décaler tout le reste, en O(n). La solution classique est le **tampon circulaire**, où deux indices tournent modulo la taille du tableau et où rien n'est jamais décalé. C'est le genre d'astuce qui distingue une implémentation utilisable d'une implémentation correcte mais lente.

### La liste chaînée

Une liste chaînée renonce à la contiguïté. Chaque élément — chaque **nœud** — contient une valeur et un pointeur vers le nœud suivant. Le dernier pointe sur rien. On ne conserve que l'adresse du premier.

Le compromis est exactement inverse de celui du tableau.

| Opération | Tableau | Liste chaînée |
|---|---|---|
| Accès au i-ième | O(1) | O(n) |
| Insertion en tête | O(n) | O(1) |
| Insertion en queue | O(1) amorti | O(n), ou O(1) avec pointeur de queue |
| Suppression connue | O(n) | O(1) si on a le prédécesseur |
| Croissance | réallocation et copie | illimitée, un nœud à la fois |
| Surcoût mémoire | nul | un pointeur par élément |
| Localité de cache | excellente | mauvaise |

Deux lignes de ce tableau méritent d'être soulignées, parce qu'on les oublie souvent.

Le **surcoût mémoire** n'est pas négligeable : sur une machine à adresses de soixante-quatre bits, chaque nœud coûte huit octets de pointeur. Une liste chaînée d'entiers de quatre octets consacre donc les deux tiers de sa mémoire à sa propre structure, avant même de compter l'alignement.

La **localité de cache** est le facteur le plus sous-estimé de la performance moderne. Un processeur ne lit pas la mémoire octet par octet mais par lignes de cache, typiquement soixante-quatre octets. Parcourir un tableau charge des éléments par paquets et chaque lecture sert plusieurs fois. Parcourir une liste chaînée saute d'une adresse à une autre, sans motif, et chaque saut risque un défaut de cache coûtant plusieurs centaines de cycles. En pratique, un parcours de tableau bat souvent un parcours de liste chaînée d'un facteur dix, alors que les deux sont en O(n). L'analyse asymptotique dit ce qui se passe quand n tend vers l'infini; elle ne dit rien de ce qui se passe sur la machine qu'on a.

La **liste doublement chaînée** ajoute à chaque nœud un pointeur vers le précédent. Elle permet le parcours dans les deux sens et la suppression d'un nœud dont on ne connaît que l'adresse, au prix d'un pointeur supplémentaire et d'une discipline de mise à jour plus délicate.

### Les arbres

Un arbre binaire de recherche généralise la recherche binaire à une structure chaînée. Chaque nœud contient une valeur et deux pointeurs, gauche et droit, avec l'invariant que tout ce qui est à gauche est plus petit et tout ce qui est à droite est plus grand.

La recherche descend depuis la racine en comparant, exactement comme dans l'annuaire. Chaque comparaison élimine un sous-arbre entier. Si l'arbre est **équilibré**, la hauteur est logarithmique et la recherche est en O(log n). On obtient ainsi ce que ni le tableau ni la liste ne donnaient : recherche, insertion et suppression toutes trois en O(log n), avec une taille qui croît librement.

Le mot « si » porte tout le poids. Un arbre binaire de recherche n'est pas équilibré par construction. Insérer des valeurs déjà triées produit une chaîne : chaque nœud n'a qu'un enfant, la hauteur est n, et toutes les opérations retombent en O(n). L'arbre a dégénéré en liste chaînée coûteuse.

C'est pourquoi les implémentations réelles utilisent des arbres **auto-équilibrants** — arbres AVL, arbres rouge-noir, arbres B — qui effectuent des rotations à l'insertion pour maintenir la hauteur logarithmique. CS50 ne les enseigne pas en détail, mais pose la question qui y mène, et c'est l'essentiel : l'étudiant sait désormais qu'il existe un problème d'équilibre, ce qui suffit pour savoir quoi chercher.

Les arbres dépassent d'ailleurs largement le stockage ordonné. Un système de fichiers est un arbre. Un document HTML est un arbre. Un programme, une fois analysé par un compilateur, est un arbre. La structure hiérarchique est si commune que la maîtriser rend lisibles des domaines apparemment sans rapport.

### Les tables de hachage

La table de hachage est la structure la plus utilisée de l'informatique pratique, et son idée est audacieuse : **calculer où une donnée doit se trouver, plutôt que la chercher**.

Une fonction de hachage prend une clé — une chaîne, par exemple — et produit un entier, qu'on ramène par un reste de division à un indice dans un tableau de seaux. Pour ranger une donnée, on calcule son hachage et on la place dans le seau correspondant. Pour la retrouver, on recalcule le même hachage et on regarde dans le même seau. Aucune comparaison avec les autres éléments, aucun parcours : un calcul, un accès. C'est du O(1).

Le problème est la **collision** : deux clés distinctes peuvent produire le même indice. C'est inévitable, par un simple argument de dénombrement — il y a plus de clés possibles que de seaux. La solution la plus courante, le **chaînage séparé**, place dans chaque seau une liste chaînée de toutes les entrées qui y aboutissent. La recherche devient alors : calculer le hachage, puis parcourir la petite liste du seau.

Le comportement dépend donc entièrement de la qualité de la fonction de hachage. Une bonne fonction distribue les clés uniformément, les listes restent courtes, et le coût moyen est constant. Une mauvaise fonction concentre les clés dans quelques seaux, les listes s'allongent, et le coût dérive vers O(n). Le pire cas théorique d'une table de hachage est donc linéaire; son cas moyen, avec une fonction correcte et un facteur de charge raisonnable, est constant. C'est le compromis qu'on accepte, et il est excellent — au point que les tableaux associatifs de Python, les objets de JavaScript, les cartes de Java et de Go sont tous, sous le capot, des tables de hachage.

Ce pire cas linéaire a d'ailleurs une portée en sécurité. Si un attaquant connaît la fonction de hachage, il peut fabriquer des milliers de clés qui entrent toutes dans le même seau, et transformer un service en O(1) en un service en O(n) — c'est l'attaque par collisions de hachage, dont la parade est une fonction de hachage randomisée à chaque démarrage.

Le redimensionnement mérite aussi une mention : quand le facteur de charge dépasse un seuil, on alloue un tableau plus grand et on **réinsère tout**, puisque les indices dépendent de la taille. L'opération est en O(n), mais elle est rare, et son coût amorti sur les insertions reste constant. La notion de **coût amorti** est ici essentielle : une opération occasionnellement coûteuse peut rester bon marché en moyenne, et raisonner opération par opération induit en erreur.

### Les tries

Le trie — de *retrieval* — pousse la logique à l'extrême. C'est un arbre où le chemin depuis la racine *est* la clé. Pour un dictionnaire de mots, chaque nœud a autant d'enfants qu'il y a de lettres possibles, et l'on descend lettre par lettre.

La conséquence est remarquable : le temps de recherche ne dépend **pas du tout** du nombre de mots stockés, mais seulement de la longueur du mot cherché. Chercher un mot de cinq lettres dans un trie de cent mots ou de dix millions de mots coûte exactement cinq descentes. C'est du O(k) où k est la longueur de la clé, ce qui, pour des clés bornées, revient à du O(1) véritable — sans cas moyen, sans hypothèse sur la distribution, sans collision.

Le prix est la mémoire, et il est brutal. Chaque nœud porte un tableau de pointeurs vers ses enfants possibles, dont la plupart sont vides. Un trie naïf pour un dictionnaire anglais consomme facilement plusieurs centaines de mégaoctets là où une table de hachage se contente de quelques dizaines. C'est le compromis temps-espace dans sa forme la plus nue.

La série de problèmes de la semaine 5 — un correcteur orthographique dont on mesure le temps d'exécution et l'usage mémoire — est construite pour faire éprouver ce compromis plutôt que pour l'énoncer. L'étudiant choisit sa structure, l'implémente, mesure, et constate.

### Le tableau récapitulatif

| Structure | Recherche | Insertion | Suppression | Mémoire | Ordre maintenu |
|---|---|---|---|---|---|
| Tableau non trié | O(n) | O(1) en fin | O(n) | minimale | non |
| Tableau trié | O(log n) | O(n) | O(n) | minimale | oui |
| Liste chaînée | O(n) | O(1) en tête | O(1) si localisée | + un pointeur/élément | non |
| ABR équilibré | O(log n) | O(log n) | O(log n) | + deux pointeurs/élément | oui |
| ABR dégénéré | O(n) | O(n) | O(n) | idem | oui |
| Table de hachage | O(1) moyen, O(n) pire | O(1) moyen | O(1) moyen | + seaux et chaînes | non |
| Trie | O(k) | O(k) | O(k) | très élevée | oui (lexicographique) |

Ce tableau est le cœur de la semaine 5, et il énonce la leçon générale : **il n'existe pas de meilleure structure de données**. Il existe des structures adaptées à des profils d'usage. Beaucoup de lectures et peu d'écritures? Un tableau trié. Beaucoup d'insertions en tête? Une liste. Des recherches par clé exacte? Une table de hachage. Des recherches par préfixe ou un parcours ordonné? Un trie ou un arbre.

Choisir suppose de savoir ce que le programme fera, ce qui ramène au point de départ : **spécifier avant de résoudre**.

---

## IX. Remonter : Python et le coût de l'abstraction

La semaine 6 quitte C pour Python, et le contraste est calculé.

### Ce qui disparaît

Le programme qui, en C, occupait une trentaine de lignes — inclusion d'en-têtes, déclaration de la fonction principale, déclaration de types, boucle indicée, gestion manuelle de la mémoire, code de retour — s'écrit en Python en trois ou quatre lignes. Toute la cérémonie a disparu.

Ce qui a disparu précisément :

**La déclaration de type.** Une variable Python n'a pas de type déclaré; l'objet qu'elle référence en a un, et il est déterminé à l'exécution. On gagne en concision et en souplesse; on perd la détection à la compilation des erreurs de type, qui se manifesteront à l'exécution, éventuellement en production, éventuellement dans une branche rarement empruntée.

**La gestion de la mémoire.** Il n'y a ni allocation ni libération explicites. Un ramasse-miettes récupère automatiquement les objets devenus inatteignables. Les trois péchés de la semaine 4 — fuite, double libération, usage après libération — deviennent structurellement impossibles, ce qui élimine d'un coup une classe entière de vulnérabilités.

**Les limites arithmétiques.** Les entiers Python n'ont pas de taille fixe : ils croissent selon le besoin, et le débordement d'entier n'existe pas. En revanche, l'imprécision en virgule flottante subsiste intégralement, parce qu'elle ne vient pas du langage mais de la représentation binaire des nombres réels. La démonstration de la semaine 1 se refait à l'identique en Python. C'est un point que le cours prend soin de faire : certaines abstractions protègent, d'autres non, et il faut savoir lesquelles.

**Le débordement de tampon.** Les séquences vérifient leurs bornes; dépasser lève une exception au lieu de corrompre la mémoire voisine.

### Ce qui apparaît

En échange, Python fournit d'emblée les structures de données de la semaine 5 : la liste, le dictionnaire — une table de hachage — et l'ensemble. Ce qui a demandé une semaine d'implémentation en C s'obtient par une paire d'accolades.

Ce raccourci est précisément ce qui justifie l'ordre du cours. Un étudiant qui rencontrerait le dictionnaire Python sans avoir implémenté une table de hachage l'utiliserait comme une boîte magique. Celui qui vient de la semaine 5 sait ce qu'il y a dedans : une fonction de hachage, des seaux, des collisions, un redimensionnement, un cas moyen constant et un pire cas linéaire. Il sait donc qu'un dictionnaire est excellent pour la recherche par clé exacte et inutile pour trouver toutes les clés commençant par « pro ». Il sait qu'une clé doit être immuable, et *pourquoi* — modifier une clé après insertion changerait son hachage et rendrait l'entrée introuvable.

Python apporte aussi son écosystème : les **modules**, fichiers de code réutilisables; les **paquets**, collections de modules; et l'index public où des centaines de milliers de bibliothèques sont disponibles. C'est l'abstraction portée à l'échelle de l'industrie — et avec elle, un problème de sécurité que le cours ne développe pas mais qu'il faut nommer : installer un paquet, c'est exécuter du code écrit par un inconnu avec les privilèges de son compte. La chaîne d'approvisionnement logicielle est devenue une surface d'attaque majeure, et la facilité d'installation en est la cause directe.

### Le prix

Python est nettement plus lent que C. Le facteur varie selon la nature du calcul, mais un ordre de grandeur de dix à cent est courant pour du code de calcul pur. La raison est structurelle : Python est interprété, chaque opération passe par une machinerie de résolution de types et d'appel dynamique, et chaque entier est un objet alloué sur le tas plutôt qu'une valeur dans un registre.

Ce coût est-il rédhibitoire? Presque jamais, et le raisonnement mérite d'être explicité. Le temps du programmeur coûte plus cher que le temps de la machine. Un programme dix fois plus lent mais écrit en un dixième du temps est généralement un meilleur investissement. Et surtout, la plupart des programmes ne sont pas limités par le processeur : ils attendent le réseau, le disque, la base de données ou l'utilisateur. Optimiser le langage d'un programme qui passe quatre-vingt-quinze pour cent de son temps à attendre une réponse HTTP ne rapporte rien.

Quand la vitesse compte vraiment, la pratique dominante n'est d'ailleurs pas d'abandonner Python mais de **descendre localement** : les bibliothèques de calcul numérique sont écrites en C ou en Fortran et exposées à Python. On garde la commodité en surface et la performance dans le noyau chaud. C'est l'abstraction utilisée intelligemment : haute par défaut, basse là où c'est mesuré nécessaire.

D'où la règle que le cours enseigne implicitement en faisant mesurer plutôt que deviner : **on n'optimise pas ce qu'on n'a pas mesuré**. L'intuition des programmeurs sur l'endroit où leur programme passe son temps est notoirement mauvaise. Le profileur existe pour cela.

---

## X. Les données : SQL et le modèle relationnel

La semaine 7 introduit les bases de données relationnelles, et l'on change de nature de problème. Jusqu'ici, les données vivaient dans la mémoire d'un programme et disparaissaient avec lui. Désormais elles persistent, elles sont partagées, elles sont volumineuses.

### Pourquoi une base plutôt qu'un fichier

Un fichier texte peut stocker des données. Ce qu'il ne peut pas faire :

Il ne peut pas **interroger efficacement**. Trouver toutes les lignes qui satisfont un critère exige de lire tout le fichier. Une base de données maintient des index et répond sans tout lire.

Il ne peut pas **garantir la cohérence**. Rien n'empêche d'écrire une date invalide, un identifiant en double, une référence vers une ligne inexistante. Une base impose des contraintes de type, d'unicité et d'intégrité référentielle, et rejette ce qui les viole.

Il ne peut pas **gérer l'accès concurrent**. Deux programmes qui écrivent simultanément dans le même fichier produisent un résultat imprévisible. Une base sérialise les accès et offre des transactions.

Il ne peut pas **survivre à une panne au milieu d'une écriture**. Une base journalise ses modifications et peut restaurer un état cohérent après un arrêt brutal.

Ces quatre besoins sont ce qui justifie l'existence d'un système de gestion de base de données. Chacun peut être réimplémenté à la main; aucun n'est facile à bien réimplémenter.

### Le modèle relationnel

Les données sont organisées en **tables** : des lignes qui sont des enregistrements, des colonnes qui sont des attributs typés. Une table a une **clé primaire**, colonne dont la valeur identifie uniquement chaque ligne. Une table peut avoir des **clés étrangères**, colonnes dont la valeur doit correspondre à une clé primaire d'une autre table.

Le principe directeur est la **normalisation** : chaque fait n'est stocké qu'une fois, à un seul endroit. Plutôt qu'une grande table où le nom d'un auteur serait répété sur chacune de ses œuvres, on crée une table d'auteurs, une table d'œuvres, et un lien entre elles. Corriger l'orthographe d'un nom devient une modification unique. Sans normalisation, la même information existe en plusieurs exemplaires et finit par diverger — c'est l'anomalie de mise à jour, qui est la maladie chronique des feuilles de calcul utilisées comme bases de données.

La normalisation a un coût, elle aussi : reconstituer une information complète exige de **joindre** plusieurs tables, ce qui coûte du temps. Les systèmes analytiques dénormalisent souvent délibérément pour accélérer la lecture, acceptant la redondance en échange. Le compromis temps-espace, encore, sous un autre visage.

### Le langage

SQL est déclaratif, et c'est sa particularité la plus intéressante. On y décrit **ce qu'on veut**, pas comment l'obtenir. La requête énonce les colonnes voulues, les tables concernées, les conditions de filtrage, le regroupement, l'ordre. L'optimiseur du système décide ensuite du plan d'exécution : quel index utiliser, dans quel ordre joindre les tables, quel algorithme de jointure employer.

C'est un renversement par rapport à tout ce qui précède dans le cours. En C, en Python, on décrit une procédure. En SQL, on décrit un résultat. Le programmeur cède le contrôle de l'exécution à un composant qui en sait davantage que lui sur la distribution réelle des données — et qui a souvent raison, mais pas toujours, ce qui explique l'existence du métier consistant à lire des plans d'exécution.

Les opérations fondamentales sont au nombre de quatre, l'acronyme CRUD : créer, lire, mettre à jour, supprimer. Autour d'elles s'organisent le filtrage, le tri, les fonctions d'agrégation qui comptent, somment ou moyennent, et le regroupement qui applique ces agrégats par catégorie.

Les **jointures** relient les tables. La jointure interne ne conserve que les lignes appariées; les jointures externes conservent en plus les lignes non appariées d'un côté ou de l'autre, en comblant avec des valeurs nulles. Comprendre la différence est l'une des sources d'erreur les plus fréquentes en analyse de données : une jointure interne mal choisie fait disparaître silencieusement les enregistrements orphelins, et le rapport produit est faux sans qu'aucune erreur ne soit signalée.

### Les index

Un index est une structure auxiliaire — typiquement un arbre B, cousin équilibré de l'arbre binaire de recherche de la semaine 5 — qui permet de trouver rapidement les lignes correspondant à une valeur.

Sans index, une recherche exige un balayage complet de la table, en O(n). Avec index, elle est en O(log n). Sur une table de dix millions de lignes, l'écart est celui entre plusieurs secondes et quelques millisecondes.

Les index ne sont pas gratuits. Ils occupent de l'espace disque. Et surtout, ils doivent être mis à jour à chaque insertion, modification ou suppression, ce qui ralentit les écritures. Une table sur-indexée écrit lentement. La règle pratique consiste à indexer les colonnes fréquemment utilisées en filtrage ou en jointure, et à ne pas indexer le reste. C'est, une dernière fois, le compromis temps-espace — et cette fois aussi un compromis lecture-écriture.

### Les transactions et les conditions de course

Le cours consacre une part notable de la semaine 7 à un problème que rien n'avait préparé : **plusieurs programmes accèdent à la même donnée en même temps**.

L'exemple canonique est le compte bancaire. Retirer de l'argent suppose de lire le solde, de vérifier qu'il est suffisant, puis de le décrémenter. Si deux retraits s'exécutent simultanément, les deux peuvent lire le même solde initial, tous deux le juger suffisant, et tous deux décrémenter — le compte devient négatif alors que le contrôle avait été fait. C'est une **condition de course**, et elle est d'autant plus redoutable qu'elle ne se manifeste que sous charge, de façon intermittente, et qu'elle disparaît quand on ajoute des messages de traçage qui modifient la synchronisation.

Le même schéma frappe le vote en double, la réservation du même siège par deux personnes, l'incrémentation d'un compteur par deux processus. La structure est toujours la même : une séquence lire-décider-écrire qu'un autre acteur peut interrompre entre la lecture et l'écriture.

La réponse des bases de données est la **transaction** : un groupe d'opérations traité comme une unité indivisible. Soit tout s'applique, soit rien. Les propriétés recherchées portent l'acronyme ACID — atomicité, cohérence, isolation, durabilité. L'isolation est celle qui traite les conditions de course : elle donne à chaque transaction l'illusion d'être seule.

Cette illusion coûte cher, et les systèmes offrent plusieurs niveaux d'isolation, du plus permissif au plus strict, avec des débits inversement proportionnels à la rigueur. Le verrouillage nécessaire introduit par ailleurs un nouveau risque, l'**interblocage** : deux transactions attendent chacune une ressource que l'autre détient, et aucune n'avance. Les systèmes le détectent et sacrifient l'une des deux, qui doit recommencer.

Il faut retenir ceci : la concurrence est le domaine où l'intuition humaine échoue le plus complètement. Nous raisonnons séquentiellement. Un programme concurrent a un nombre d'entrelacements possibles qui croît de façon combinatoire, et il suffit qu'un seul soit fautif pour que le système échoue une fois sur dix mille — c'est-à-dire tous les jours, à l'échelle d'un service réel.

### L'injection SQL

Si les requêtes sont construites par concaténation de chaînes, un utilisateur qui contrôle une partie du texte contrôle une partie de la requête. Il peut fermer la chaîne littérale, ajouter une condition toujours vraie, terminer la requête et en ajouter une autre. On appelle cela l'**injection SQL**, et elle figure au sommet des classements de vulnérabilités depuis plus de vingt ans.

Les conséquences vont du contournement d'authentification — une condition toujours vraie fait accepter n'importe quel mot de passe — à l'exfiltration complète de la base, voire à sa destruction.

La parade est connue, simple et absolue : les **requêtes paramétrées**. On envoie au système la structure de la requête avec des emplacements réservés, puis les valeurs séparément. Le système ne compile la requête qu'une fois, à partir de la structure seule, et traite les valeurs comme des données pures, jamais comme du code. Aucune ponctuation contenue dans une valeur ne peut modifier la requête, parce que la requête est déjà compilée quand la valeur arrive.

L'échappement manuel des caractères spéciaux, la validation par listes noires, le filtrage de mots-clés sont autant de contournements imparfaits, et l'histoire de la sécurité applicative est jonchée de leurs échecs. La seule défense correcte est structurelle : **séparer le code des données**.

Ce principe dépasse largement SQL. L'injection de commandes système, l'injection de scripts dans une page web, l'injection de modèles côté serveur reposent toutes sur la même faute — un texte fourni par un tiers est interprété comme instruction plutôt que comme donnée. Reconnaître ce motif est l'une des compétences les plus transférables que le cours transmette.

---

## XI. L'intelligence artificielle

L'édition 2026 insère, entre la semaine 7 et la semaine 8, une semaine entière consacrée à l'intelligence artificielle. Le site officiel en énumère le contenu : ingénierie de prompts, prompt système et prompt utilisateur, arbres de décision, algorithme minimax, apprentissage automatique, apprentissage par renforcement avec le dilemme entre exploration et exploitation, réseaux de neurones profonds, grands modèles de langage, architecture des transformeurs, et hallucinations.

Ce n'est pas un supplément d'actualité collé à un programme existant. C'est un chapitre construit pour montrer une continuité — et une rupture.

### L'IA sans apprentissage

L'intelligence artificielle ne commence pas avec l'apprentissage automatique. Pendant un demi-siècle, elle a consisté à écrire des règles.

Un **arbre de décision** en est la forme la plus directe : une cascade de conditions qui aboutit à une action. Le personnage d'un jeu vidéo qui poursuit le joueur s'il est proche, se soigne s'il est blessé et patrouille sinon exécute un arbre de décision. C'est de la logique conditionnelle de la semaine 0, appliquée à un comportement.

L'algorithme **minimax** franchit un cran. Il s'applique aux jeux à deux joueurs, à somme nulle et à information complète — le tic-tac-toe, les échecs. L'idée : un joueur cherche à maximiser un score, l'autre à le minimiser. On explore l'arbre des coups possibles jusqu'aux positions terminales, on leur attribue une valeur, puis on remonte en supposant que chaque joueur joue son meilleur coup. Le coup à jouer est celui qui maximise le résultat garanti dans le pire cas.

Un programme de tic-tac-toe écrit ainsi est **imbattable**. Il n'a rien appris, il ne s'améliore pas, il n'a aucune représentation de ce qu'est un jeu. Il énumère et il compare. C'est une leçon utile contre l'anthropomorphisme : un comportement qui ressemble à une stratégie experte peut n'être qu'une recherche exhaustive.

La limite est le nombre de positions. Le tic-tac-toe en compte quelques centaines de milliers, on peut tout explorer. Les échecs en comptent davantage qu'il n'y a d'atomes dans l'univers observable. Il faut donc couper l'arbre à une profondeur donnée et estimer la valeur des positions non terminales par une **fonction d'évaluation** heuristique, et élaguer les branches dont on peut prouver qu'elles ne changeront pas le résultat — l'élagage alpha-bêta, qui ne change pas la réponse mais réduit énormément le travail. On retrouve exactement les préoccupations de la semaine 3 : la correction et le coût, séparément.

### Quand les règles ne suffisent plus

Certaines tâches résistent à toute écriture de règles. Distinguer un chat d'un chien sur une photographie en est l'exemple canonique. Aucune liste de conditions sur les pixels ne le fait correctement, et ce n'est pas faute d'avoir essayé pendant vingt ans.

L'**apprentissage automatique** renverse la démarche. Au lieu d'écrire la règle, on fournit des exemples étiquetés et on laisse un algorithme ajuster ses paramètres jusqu'à ce qu'il classe correctement. C'est l'apprentissage **supervisé**, et il exige une quantité d'exemples annotés qui est souvent le véritable goulot d'étranglement — la partie chère d'un projet d'apprentissage automatique n'est presque jamais le modèle, c'est la donnée.

L'**apprentissage par renforcement** procède autrement : pas d'exemples étiquetés, mais un environnement, des actions possibles et un signal de récompense. L'agent essaie, observe le résultat, et ajuste sa politique pour favoriser ce qui rapporte. C'est ainsi qu'on entraîne un programme à jouer sans lui montrer de parties, ou un robot à marcher sans lui décrire la marche.

Le dilemme central, que le cours nomme explicitement, est celui de l'**exploration contre l'exploitation**. Exploiter, c'est refaire ce qui a fonctionné. Explorer, c'est essayer autre chose au risque de perdre. Un agent qui n'exploite jamais n'accumule rien; un agent qui n'explore jamais se fige sur la première stratégie médiocre qu'il a trouvée et ne découvrira jamais la bonne. L'équilibre habituel consiste à choisir l'action la mieux notée la plupart du temps et une action au hasard occasionnellement, avec une part d'aléatoire qui décroît à mesure que l'agent apprend.

Ce dilemme n'a rien de spécifiquement informatique, et c'est ce qui le rend intéressant. Il structure le choix d'un restaurant, d'une carrière, d'une stratégie de recherche. L'informatique lui donne une formulation précise et des solutions démontrables.

### Les réseaux de neurones

Un neurone artificiel est une fonction élémentaire : il reçoit des entrées numériques, les multiplie par des poids, additionne, ajoute un biais, et passe le résultat dans une fonction non linéaire. C'est tout. L'analogie biologique est lointaine au point d'être trompeuse.

La puissance vient de l'assemblage. On dispose ces neurones en couches, la sortie de l'une alimentant l'entrée de la suivante. Une couche d'entrée reçoit les données brutes, une couche de sortie produit le résultat, et entre les deux s'empilent des couches dites cachées. Un réseau **profond** est simplement un réseau à nombreuses couches cachées — d'où « apprentissage profond ».

L'apprentissage consiste à ajuster les poids. On présente un exemple, on compare la sortie produite à la sortie attendue, on mesure l'écart par une fonction de perte, puis on propage cet écart à rebours à travers le réseau pour calculer la contribution de chaque poids à l'erreur, et on corrige chaque poids d'une petite quantité dans la direction qui réduit la perte. C'est la rétropropagation, couplée à la descente de gradient. Répété des millions de fois, ce processus produit un réseau qui généralise.

Deux observations s'imposent.

La première est qu'**il n'y a aucune magie**, seulement de l'arithmétique en très grand nombre. Les opérations sont des multiplications de matrices, et c'est précisément pourquoi les processeurs graphiques, conçus pour cela, ont rendu le domaine praticable.

La seconde est que **le résultat n'est pas interprétable**. Un réseau entraîné est un tableau de milliards de nombres. On peut vérifier qu'il classe bien; on ne peut pas, en général, dire pourquoi il a classé ainsi. C'est une rupture avec tout ce qui précède dans le cours. Un programme en C peut être lu, raisonné, prouvé. Un réseau ne peut être qu'évalué statistiquement. Cette opacité n'est pas un défaut d'implémentation qu'on corrigerait avec plus de soin; elle est constitutive de la méthode, et elle pose des problèmes réels dès qu'une décision engage — un refus de crédit, un diagnostic, une décision judiciaire.

### Les transformeurs et les grands modèles de langage

L'architecture qui a rendu possibles les modèles de langage actuels est le **transformeur**, introduit en 2017. Son apport tient au mécanisme d'**attention**, qui permet à chaque élément d'une séquence de pondérer sa relation avec tous les autres, quelle que soit la distance qui les sépare.

C'est ce qui manquait aux architectures antérieures, qui traitaient les séquences par récurrence et perdaient progressivement le contexte lointain. Avec l'attention, le mot qui résout un pronom peut se trouver mille mots plus tôt sans que cela pose de difficulté structurelle. Un avantage secondaire, mais décisif en pratique, est que le calcul se parallélise : contrairement à une récurrence, l'attention traite tous les éléments simultanément, ce qui permet d'entraîner sur des corpus d'une taille auparavant inaccessible.

Un **grand modèle de langage** est un transformeur entraîné à prédire le prochain fragment de texte, sur des corpus de plusieurs milliers de milliards de mots. C'est là toute la tâche d'entraînement : la prédiction du token suivant. Rien de plus.

Il faut s'arrêter sur ce point, car il est la clé de tout le reste. **Un modèle de langage produit du texte plausible, et la plausibilité n'est pas la vérité.** Le modèle n'a pas de base de faits qu'il consulterait, pas de mécanisme de vérification, pas de représentation de sa propre incertitude qui serait séparable de la génération. Quand il énonce une chose vraie, c'est parce que cette chose était bien représentée dans son corpus et que la continuation vraie était aussi la plus probable. Quand il énonce une chose fausse avec la même assurance, le mécanisme est identique.

C'est ce qu'on nomme **hallucination**, et le terme est doublement malheureux. Il suggère une défaillance occasionnelle d'un système par ailleurs fiable, alors qu'il s'agit du fonctionnement normal appliqué à une région où le corpus ne contraignait pas la réponse. Et il suggère une pathologie, alors qu'aucun mécanisme interne ne distingue les deux cas. Le modèle ne ment pas — mentir suppose de connaître la vérité.

Les conséquences pratiques sont directes et le cours ne les esquive pas : **tout ce qui est vérifiable doit être vérifié**. Une référence bibliographique produite par un modèle peut être entièrement inventée tout en ayant la forme exacte d'une vraie référence — auteurs plausibles, revue plausible, année plausible. Un extrait de code peut appeler une fonction qui n'existe pas dans la bibliothèque. Un raisonnement peut être fluide et faux à la troisième étape.

### L'ingénierie de prompts

Le cours distingue deux niveaux d'instruction.

Le **prompt système** est fourni par le développeur de l'application. Il définit le rôle, le ton, les contraintes, ce que l'assistant doit refuser. Il est invisible à l'utilisateur final et persiste sur toute la conversation.

Le **prompt utilisateur** est ce que la personne tape. Il varie à chaque tour.

Cette séparation est un choix d'architecture, et il faut voir ce qu'il n'est pas : ce n'est pas une frontière de sécurité au sens où l'entend un système d'exploitation. Les deux niveaux arrivent au modèle comme du texte dans la même fenêtre de contexte, avec des priorités apprises plutôt que des privilèges appliqués. D'où l'**injection de prompt**, où un texte tiers — une page web, un document, un courriel que le modèle est chargé de traiter — contient des instructions que le modèle risque de suivre comme si elles venaient de l'utilisateur légitime.

Le lecteur attentif reconnaîtra le motif de la semaine 7. C'est exactement l'injection SQL, transposée : du contenu qui devrait être traité comme donnée est interprété comme instruction. Mais la parade de la semaine 7 ne se transpose pas. Contre l'injection SQL, on sépare structurellement le code des données par une requête précompilée. Contre l'injection de prompt, aucune séparation structurelle équivalente n'existe, parce qu'un modèle de langage n'a qu'un seul canal — le texte — et que sa capacité même consiste à interpréter ce texte. Le problème est ouvert, et les défenses actuelles sont des mitigations statistiques, non des garanties.

Quant à l'ingénierie de prompts proprement dite, elle relève moins d'une technique ésotérique que de ce que le cours enseigne depuis la semaine 0 : **spécifier clairement**. Donner le contexte, préciser le format attendu, décomposer une tâche complexe, fournir des exemples. Ce sont les mêmes exigences que celles d'un bon énoncé de problème, adressées à un exécutant qui ne pose pas de question.

### Ce que cela change pour l'apprentissage

Il y a une tension évidente entre l'existence d'outils capables d'écrire du code correct et un cours qui demande à ses étudiants d'écrire du code.

CS50 y répond de façon nuancée, en autorisant l'usage d'outils d'IA pédagogiques conçus pour orienter sans donner la solution, tout en interdisant de soumettre du code qu'on n'a pas compris. La politique dit en substance : l'outil peut expliquer, il ne peut pas remplacer.

Le raisonnement de fond est solide et dépasse la question académique. Un modèle produit du code plausible. Évaluer si ce code est correct, s'il gère les cas limites, s'il a la bonne complexité, s'il introduit une vulnérabilité, s'il conviendra dans six mois — cela suppose exactement les connaissances que le cours transmet. **L'existence d'un générateur de code augmente la valeur du jugement plutôt qu'elle ne la diminue**, parce qu'elle déplace le goulot d'étranglement de la production vers l'évaluation. Celui qui ne sait pas lire un algorithme quadratique ne saura pas que le code qu'on lui a donné mettra quatorze heures au lieu d'une seconde.

---

## XII. Le réseau, ou l'abstraction la plus réussie

La semaine 8 sort de la machine unique. Elle est, à mon sens, la démonstration la plus achevée de la thèse du cours sur l'abstraction, parce qu'elle expose un empilement de conventions dont chaque étage ignore délibérément ce que fait celui du dessous.

### Les couches

Taper une adresse dans un navigateur déclenche une cascade que l'on peut décrire à cinq niveaux.

**Le niveau physique** transporte des signaux — impulsions électriques dans le cuivre, impulsions lumineuses dans la fibre, ondes dans l'air. C'est le seul niveau qui existe matériellement.

**Le protocole IP** attribue à chaque machine une adresse et se charge d'acheminer des paquets d'une adresse à une autre à travers des **routeurs**, chacun ne connaissant que ses voisins et une table lui indiquant vers qui transmettre. Aucun routeur ne connaît le trajet complet. Cette conception décentralisée est la raison pour laquelle le réseau survit à la panne d'un de ses nœuds : le trafic se réachemine.

IP ne garantit rien. Les paquets peuvent se perdre, arriver dans le désordre, être dupliqués. C'est un service de remise au mieux, et cette modestie est délibérée : elle a permis au réseau de croître sans que les routeurs aient à maintenir d'état par connexion.

**Le protocole TCP** ajoute par-dessus les garanties manquantes. Il découpe les données en segments numérotés, exige un accusé de réception pour chacun, retransmet ce qui n'est pas confirmé, réordonne à l'arrivée, et ajuste son débit pour ne pas saturer le réseau. Deux machines obtiennent ainsi un canal fiable et ordonné construit sur un service qui ne l'est pas. C'est une abstraction au sens plein : elle fournit une propriété que le substrat n'a pas.

TCP introduit aussi le **numéro de port**, qui distingue les services sur une même machine — le trafic web sur un port, le courriel sur un autre. Une adresse IP désigne une machine; un couple adresse-port désigne un service.

**Le système DNS** traduit les noms lisibles en adresses IP. C'est un annuaire réparti et hiérarchique, où la requête remonte de serveur en serveur jusqu'à trouver l'autorité compétente, avec une mise en cache à chaque étage. Son existence tient à un fait simple : les humains retiennent des noms, les machines routent des nombres.

**Le protocole HTTP**, enfin, définit ce que les deux extrémités se disent une fois le canal établi. Un client envoie une requête composée d'une méthode, d'un chemin et d'en-têtes; un serveur répond par un code d'état, des en-têtes et un corps.

Les méthodes principales sont GET, qui demande une ressource et dont les paramètres sont visibles dans l'adresse, et POST, qui soumet des données dans le corps de la requête. La distinction a une conséquence pratique immédiate : les paramètres d'un GET apparaissent dans l'historique du navigateur, dans les journaux des serveurs intermédiaires et dans l'en-tête de provenance transmis au site suivant. On n'y met jamais un mot de passe ni une donnée personnelle.

Les codes d'état se lisent par leur première décimale : 2xx pour le succès, 3xx pour la redirection, 4xx pour une faute du client, 5xx pour une faute du serveur. Le 404 est entré dans la culture générale; le 500 est celui qui réveille les équipes d'astreinte.

### Ce que l'empilement démontre

Le point à retenir n'est pas la liste des couches mais leur **indépendance**. HTTP ne sait pas si les octets voyagent par fibre ou par satellite. TCP ne sait pas s'il transporte une page web ou un courriel. IP ne sait pas ce que contiennent ses paquets. Chaque couche traite celle du dessous comme une boîte noire.

C'est ce qui a permis au réseau d'absorber, sans réécriture, l'apparition du sans-fil, de la fibre, de la téléphonie mobile, du chiffrement généralisé et du vidéo en continu. Les couches basses ont été remplacées plusieurs fois sans que les couches hautes s'en aperçoivent. Une architecture en couches bien conçue est ce qui permet à un système de survivre à ses propres composants — et c'est peut-être le seul enseignement du cours qui vaille pour tout système complexe, informatique ou non.

### Les trois langages du client

**HTML** décrit la structure d'un document : un arbre d'éléments délimités par des balises et qualifiés par des attributs. Ce n'est pas un langage de programmation — il n'a ni condition, ni boucle, ni variable. C'est un langage de balisage, et cette limitation est ce qui en fait un format d'échange durable.

**CSS** décrit la présentation : des sélecteurs qui désignent des éléments, des propriétés qui leur assignent une apparence. La séparation entre structure et présentation permet de changer entièrement l'aspect d'un site sans toucher à son contenu.

**JavaScript** ajoute le comportement. C'est un langage de programmation complet, exécuté dans le navigateur, qui manipule le document à travers le **DOM** — la représentation en arbre de la page, en mémoire. Modifier cet arbre modifie l'affichage instantanément.

Le modèle de programmation est **événementiel** : le code enregistre des gestionnaires qui seront appelés quand quelque chose survient — un clic, une frappe, une réponse réseau. C'est un renversement du flux de contrôle. Le programme ne déroule pas une séquence; il attend, et il réagit.

L'étudiant qui a fait la semaine 0 reconnaît la structure : Scratch fonctionnait déjà ainsi, avec ses blocs déclenchés par le drapeau vert ou par une touche. La boucle s'est refermée, et elle démontre ce que le cours voulait démontrer — que ce qu'on apprend n'est pas un langage mais un ensemble de concepts qui se réincarnent.

### Les expressions régulières

La semaine 8 introduit aussi les expressions régulières, un mini-langage pour décrire des motifs dans du texte. On y déclare des classes de caractères, des quantificateurs, des ancrages et des groupes de capture, et l'on obtient un outil de reconnaissance d'une densité redoutable.

Leur utilité est réelle — validation de format, extraction, substitution — et leur danger l'est aussi. Une expression régulière un peu longue devient illisible, y compris pour celui qui l'a écrite. Et certaines constructions, notamment les quantificateurs imbriqués, produisent sur des entrées adverses un retour sur trace exponentiel : c'est le déni de service par expression régulière, où une chaîne de quelques dizaines de caractères bloque un serveur pendant des minutes.

Il faut mentionner enfin la limite théorique, parce qu'elle est instructive. Les expressions régulières reconnaissent les langages réguliers, ce qui exclut structurellement les langages imbriqués. On ne peut pas analyser du HTML avec une expression régulière — non pas parce que c'est difficile, mais parce que c'est impossible : HTML admet une imbrication arbitraire, et aucun automate fini ne compte jusqu'à l'infini. C'est le genre de résultat que l'informatique théorique fournit et qui épargne des semaines d'effort à qui le connaît.

---

## XIII. Construire une application : Flask

La semaine 9 assemble tout. Elle utilise Python de la semaine 6, SQL de la semaine 7, HTML et JavaScript de la semaine 8, et produit une application web complète.

### Le modèle client-serveur

Le navigateur envoie une requête, le serveur la traite, la réponse revient. Ce qui distingue une page statique d'une application, c'est que le serveur **construit** la réponse plutôt que de la lire sur disque : il consulte une base de données, applique une logique, remplit un gabarit.

Flask est un micro-cadriciel : il fournit le nécessaire — routage, requêtes, réponses, gabarits, sessions — et rien de plus. Le choix pédagogique est cohérent avec celui de C : un outil minimal où l'on voit ce qui se passe, plutôt qu'un cadriciel complet qui déciderait à la place de l'étudiant.

### Routes et décorateurs

Une **route** associe un chemin d'URL à une fonction Python. La syntaxe utilise un **décorateur**, c'est-à-dire une fonction qui en enveloppe une autre pour lui ajouter un comportement sans modifier son code. C'est une notion de programmation fonctionnelle introduite ici de façon presque indolore, mais qui mérite d'être reconnue pour ce qu'elle est : la fonction est traitée comme une valeur qu'on passe et qu'on transforme.

Chaque fonction de route reçoit implicitement la requête et retourne une réponse. Le serveur maintient une table de correspondance et appelle la bonne fonction. C'est simple, et cette simplicité est la raison du succès du modèle.

### Le problème de l'état

HTTP est **sans état**. Chaque requête est indépendante; le serveur n'a aucun moyen intrinsèque de savoir que deux requêtes viennent du même utilisateur. Cette propriété est ce qui permet à un serveur de traiter des millions de clients sans mémoriser chacun.

Mais une application a besoin d'état : qui est connecté, ce qu'il y a dans le panier, quelle langue afficher.

La solution est le **cookie** : le serveur envoie une petite valeur que le navigateur conserve et renvoie à chaque requête suivante vers le même domaine. Le serveur reconnaît alors le client.

La **session** est la construction qui s'appuie dessus. Le cookie ne contient qu'un identifiant opaque et aléatoire; les données réelles restent côté serveur, indexées par cet identifiant. La séparation est essentielle. Un cookie qui contiendrait directement l'identité de l'utilisateur serait modifiable par lui — il suffirait de changer un nom pour devenir quelqu'un d'autre.

Les conséquences en sécurité sont directes et le cours les aborde. L'identifiant de session doit être **imprévisible**, faute de quoi on devine celui d'autrui. Il doit être transmis exclusivement sur une connexion chiffrée, sans quoi on l'intercepte. Il doit être marqué inaccessible au JavaScript de la page, ce qui limite les dégâts d'une injection de script. Il doit être **renouvelé à la connexion**, pour empêcher un attaquant de fixer d'avance la valeur qui servira. Et il doit expirer.

Le vol de session mérite d'être compris pour ce qu'il est : celui qui détient le cookie **est** l'utilisateur, du point de vue du serveur. Le mot de passe n'est vérifié qu'une fois, à l'ouverture. Toute la sécurité de la suite repose sur un jeton.

### Séparer les responsabilités

Le cours introduit enfin, sans en faire un dogme, la séparation en trois responsabilités : les **données** et leur accès, la **présentation** confiée à des gabarits, la **logique** qui relie les deux. C'est le patron modèle-vue-contrôleur, sous une forme ou une autre.

L'intérêt n'est pas la conformité à un schéma mais ce qu'il rend possible : changer l'apparence sans toucher à la logique, changer la base de données sans réécrire les pages, tester la logique sans navigateur. Une modification bien localisée est une modification bon marché, et le coût du changement est la seule mesure honnête de la qualité d'une architecture.

C'est le même principe que le type abstrait de la semaine 5, à une échelle supérieure. Le cours n'a cessé de le répéter sous des formes différentes : **définir des frontières, et faire en sorte que ce qui change d'un côté ne traverse pas**.

---

## XIV. La sécurité, qui n'est pas un chapitre

La compilation vidéo se termine sur une leçon consacrée à la cybersécurité. Le placement est un peu trompeur, car le sujet n'a jamais cessé d'être présent : la sécurité n'est pas un module qu'on ajoute, c'est une propriété qui se perd à chaque étage.

### Le fil rouge du cours

Reprenons ce que les semaines précédentes ont posé, sans le présenter comme de la sécurité.

Le **débordement de tampon** de la semaine 4 vient de ce que C ne vérifie pas les bornes d'un tableau. L'**injection SQL** de la semaine 7 vient de ce qu'une requête construite par concaténation confond code et données. La **condition de course** de la semaine 7 vient de ce qu'une séquence lire-décider-écrire n'est pas atomique. Le **vol de session** de la semaine 9 vient de ce qu'un jeton confère une identité. L'**injection de prompt** de la semaine sur l'IA vient de ce qu'un modèle de langage ne distingue pas structurellement l'instruction de la donnée.

Cinq vulnérabilités, cinq étages différents, et deux causes seulement. La première : **une frontière de confiance a été franchie sans vérification**. La seconde : **quelque chose qui devait être traité comme donnée a été traité comme instruction**.

Qui a compris ces deux motifs a compris la structure de la majorité des failles applicatives, et sait où regarder dans un système qu'il n'a jamais vu.

### Les mots de passe

Un serveur ne doit jamais stocker les mots de passe. Il stocke leur **empreinte**, produite par une fonction de hachage à sens unique : facile à calculer, infaisable à inverser. À la connexion, on hache ce qui est fourni et on compare les empreintes.

Cela ne suffit pas. Deux utilisateurs avec le même mot de passe auraient la même empreinte, et un attaquant qui a précalculé les empreintes des mots de passe courants les reconnaît d'un coup d'œil. On ajoute donc un **sel** : une valeur aléatoire, différente pour chaque compte, concaténée au mot de passe avant hachage et stockée à côté de l'empreinte. Les tables précalculées deviennent inutiles, et deux comptes identiques ont des empreintes différentes.

Cela ne suffit toujours pas. Une fonction de hachage rapide se teste des milliards de fois par seconde sur du matériel spécialisé. On utilise donc des fonctions **délibérément lentes et gourmandes en mémoire**, conçues pour l'usage précis du stockage de mots de passe. Le coût d'une vérification légitime est de quelques dizaines de millisecondes, imperceptible; le coût d'une attaque par force brute est multiplié dans les mêmes proportions.

La règle qui en découle vaut d'être répétée : **on n'écrit pas son propre schéma de stockage de mots de passe**. On utilise une fonction éprouvée, avec ses paramètres recommandés.

Quant à la politique de mots de passe, l'évolution des recommandations mérite d'être notée, car elle contredit ce que beaucoup d'organisations appliquent encore. Les exigences de complexité — une majuscule, un chiffre, un symbole — produisent des mots de passe courts, prévisibles et difficiles à retenir, donc réutilisés ou notés. L'expiration forcée produit des variations incrémentales triviales. La recommandation actuelle privilégie la **longueur**, le contrôle contre les listes de mots de passe déjà compromis, et l'absence d'expiration sans indice de compromission.

### Chiffrement symétrique et asymétrique

Le **chiffrement symétrique** utilise une même clé pour chiffrer et déchiffrer. Il est rapide et convient au volume. Son problème est la distribution : comment transmettre la clé à son correspondant sans qu'un tiers l'intercepte?

Le **chiffrement asymétrique** résout ce problème par une paire de clés mathématiquement liées. Ce que l'une chiffre, seule l'autre le déchiffre. On publie la première — la clé publique — et l'on garde la seconde. N'importe qui peut alors chiffrer un message que seul le détenteur de la clé privée pourra lire. L'opération inverse produit une **signature** : ce que la clé privée a chiffré, la clé publique le vérifie, ce qui prouve l'origine.

L'asymétrique est lent. En pratique, on combine : on utilise l'asymétrique pour négocier une clé de session symétrique, puis on chiffre tout le trafic avec celle-ci. C'est ce que fait HTTPS.

Il reste un problème que la cryptographie seule ne règle pas : comment savoir que la clé publique reçue est bien celle du destinataire voulu, et non celle d'un intercepteur? La réponse est institutionnelle plutôt que mathématique — un système d'autorités de certification qui attestent l'appartenance d'une clé à un domaine, et que les navigateurs choisissent de croire. Le cadenas affiché par le navigateur signifie que la connexion est chiffrée et que le certificat est valide pour ce domaine. **Il ne signifie pas que le site est honnête.** Un site frauduleux obtient un certificat valide en quelques minutes. La confusion entre les deux est activement exploitée.

### L'authentification à deux facteurs

Un mot de passe est un facteur unique : ce qu'on sait. S'il fuit, tout est perdu. Un second facteur — ce qu'on possède ou ce qu'on est — exige qu'un attaquant compromette deux choses de natures différentes.

Tous les seconds facteurs ne se valent pas. Le code envoyé par message texte est le plus répandu et le plus faible, car il est vulnérable au détournement de carte SIM. Le code à usage unique généré par une application est nettement meilleur. La clé matérielle ou le dispositif compatible avec l'authentification par clé publique est le seul qui résiste à l'hameçonnage, parce qu'il vérifie le domaine avant de signer : un site frauduleux, même parfaitement imité, ne reçoit rien d'utilisable.

### La leçon générale

La sécurité est **asymétrique**. Le défenseur doit couvrir toutes les entrées; l'attaquant n'en a besoin que d'une. Elle est **composite** : un système fait de composants sûrs peut être vulnérable dans leur assemblage. Elle est **temporelle** : ce qui était sûr en 2005 ne l'est plus, parce que le matériel a progressé et que des attaques ont été découvertes.

Et surtout, elle est **économique**. La sécurité absolue n'existe pas. Ce qui existe, c'est un coût d'attaque qu'on élève au-dessus de la valeur de ce qui est protégé. Poser correctement la question suppose de savoir contre qui l'on se défend — un curieux, un fraudeur opportuniste, un concurrent, un État — car les moyens diffèrent de plusieurs ordres de grandeur.

---

## XV. La forme du cours comme argument

Il faut dire un mot de la pédagogie de CS50, parce qu'elle est elle-même un objet d'étude et qu'elle porte une part de la thèse.

### La progression

L'ordre des semaines n'est pas chronologique ni historique. Il est **argumentatif**.

On commence par le plus haut niveau — des blocs qu'on assemble sans syntaxe — pour installer les concepts. On descend ensuite au plus bas — pointeurs, allocation, adresses — pour installer la compréhension du substrat. Puis on remonte — Python, SQL, le web — en réutilisant à chaque étage ce qui a été construit plus bas.

Cette forme en V a une conséquence précise : à la fin, l'étudiant utilise des abstractions **qu'il a lui-même implémentées**. Le dictionnaire Python n'est pas un mystère, c'est la table de hachage de la semaine 5. La chaîne de caractères n'est pas un mystère, c'est le tableau terminé par un zéro de la semaine 2. La session web n'est pas un mystère, c'est un identifiant dans un cookie.

C'est la différence entre savoir se servir d'un outil et savoir quand il va casser.

### Les séries de problèmes

Dix séries, numérotées de 0 à 9, chacune adossée à une semaine. Leur conception obéit à trois principes qui expliquent une bonne part de l'efficacité du cours.

D'abord, elles sont **spécifiées jusqu'à l'obsession**, y compris les cas limites et le format exact des sorties. L'étudiant apprend, sans qu'on le lui dise, à lire une spécification — compétence que les cours théoriques transmettent rarement et que le métier exige quotidiennement.

Ensuite, elles sont **vérifiées automatiquement**. Un outil exécute les tests de correction, un autre analyse le style. La rétroaction est immédiate et impersonnelle, ce qui permet d'itérer sans attendre et sans embarras.

Enfin, elles sont souvent proposées en deux niveaux de difficulté sur le même sujet, ce qui permet à un cours unique de servir des étudiants dont l'expérience préalable varie du néant à plusieurs années.

Le projet final, lui, ne prescrit rien : il demande de concevoir et de réaliser quelque chose d'utile. C'est le seul moment du cours où la spécification est à écrire soi-même, et c'est probablement le plus difficile — pour la raison exposée au tout début de cet essai.

### L'accès

CS50 est intégralement gratuit et librement accessible. Les vidéos, les notes, les diapositives, le code source, les séries de problèmes et l'outillage de correction sont publics. La certification est payante, mais l'apprentissage ne l'est pas. Le matériel est sous une licence qui autorise les enseignants à l'adopter ou à l'adapter.

Il faut mesurer ce que cela représente. Un cours d'introduction d'une des universités les plus sélectives du monde, avec son infrastructure d'évaluation, est disponible pour quiconque a une connexion. La compilation de vingt-cinq heures publiée par freeCodeCamp étend encore cette portée, en supprimant jusqu'au besoin de s'inscrire.

C'est une position sur la nature de la connaissance informatique : elle ne tire pas sa valeur de sa rareté. Ce que l'université vend n'est pas l'information — elle est publique — mais l'encadrement, l'évaluation et l'attestation. La distinction est saine, et l'informatique est l'une des rares disciplines où elle a été poussée jusqu'au bout.

---

## XVI. Ce qui reste

### Ce que l'informatique n'est pas

Elle n'est pas l'étude des ordinateurs. Le matériel de 1972, quand C a été écrit, n'a plus aucun rapport avec celui d'aujourd'hui. Les algorithmes de tri, eux, sont identiques, et leurs démonstrations aussi.

Elle n'est pas la programmation. Programmer est à l'informatique ce qu'écrire est à la littérature : l'instrument indispensable, jamais l'objet.

Elle n'est pas la maîtrise d'un langage. CS50 en traverse six — Scratch, C, Python, SQL, HTML avec CSS, JavaScript — précisément pour rendre visible que ce qui se transporte d'un langage à l'autre est ce qui compte.

Elle n'est pas un ensemble de recettes. Les recettes se périment. Un cadriciel dominant est remplacé en cinq ans. Une notation asymptotique ne se périme pas.

### Ce qu'elle est

C'est une discipline du **jugement sous contrainte**.

Chaque décision technique est un arbitrage, et le cours ne cesse d'en présenter : temps contre espace, lisibilité contre performance, abstraction contre contrôle, sécurité contre commodité, généralité contre simplicité, coût de développement contre coût d'exécution. Aucun de ces arbitrages n'a de solution universelle. Chacun a une bonne réponse *pour un contexte donné*, et l'expertise consiste à identifier le contexte avant de choisir.

C'est aussi une discipline de la **décomposition**. Un problème qu'on ne sait pas résoudre se découpe jusqu'à ce que les morceaux soient résolubles. Le tri par fusion, la recherche binaire, la récursivité, l'architecture en couches du réseau, la séparation des responsabilités d'une application web : ce sont cinq formes d'un même geste.

C'est enfin une discipline de la **rigueur sur la représentation**. Tout ce que fait une machine, elle le fait sur des nombres qui n'ont d'autre sens que celui d'une convention. Se tromper de convention — confondre un caractère et un entier, dépasser la capacité d'un type, croire qu'un flottant vaut exactement un dixième, traiter une donnée comme une instruction — est la source d'une part écrasante des défaillances, des plus bénignes aux plus coûteuses.

### Ce qui subsiste après l'oubli

La plupart de ceux qui suivent CS50 auront oublié, dans cinq ans, la syntaxe exacte d'une déclaration de pointeur et l'ordre des arguments d'une fonction de la bibliothèque standard. Cela n'a aucune importance : ces choses se retrouvent en trente secondes.

Ce qui subsiste est d'un autre ordre.

Savoir qu'un problème doit être spécifié avant d'être résolu. Savoir qu'une solution correcte peut être inutilisable, et pourquoi. Savoir que doubler la taille d'une entrée n'a pas le même effet selon l'algorithme, et savoir estimer lequel. Savoir qu'une abstraction cache un mécanisme, que ce mécanisme a un coût, et qu'on peut descendre le regarder. Savoir qu'un système fait de couches indépendantes survit à ses composants. Savoir que la donnée fournie par un tiers n'est jamais une instruction. Savoir qu'on ne mesure pas la qualité d'un programme à son élégance mais au coût de sa prochaine modification. Savoir, enfin, qu'un texte plausible n'est pas un texte vrai, et que la vérification reste au programmeur.

Ce sont des dispositions du jugement. Elles ne dépendent d'aucun langage, d'aucune machine et d'aucune époque. C'est cela, l'informatique — et c'est ce que vingt-cinq heures de cours, correctement suivies, peuvent réellement transmettre.

---

## Sources

- **Vidéo** — *Harvard CS50 (2026) – Full Computer Science University Course*, freeCodeCamp.org, 25 heures. Compilation intégrale des leçons magistrales de l'édition 2026, incluant la nouvelle section sur l'intelligence artificielle. `https://www.youtube.com/watch?v=gmuTjeQUbTM`
- **Article de présentation** — freeCodeCamp, *Harvard CS50 2026 – Free Computer Science University Course*. Découpage de la compilation en treize segments : leçons 0 à 9, *Artificial Intelligence*, *Emoji*, *Cybersecurity*. `https://www.freecodecamp.org/news/harvard-cs50-2026-free-computer-science-university-course/`
- **Site officiel du cours** — *CS50x 2026 : Introduction to Computer Science*, Harvard University, David J. Malan. Structure canonique en onze semaines plus une semaine sur l'intelligence artificielle, dix séries de problèmes, projet final. `https://cs50.harvard.edu/x/`
- **Pages de semaine consultées** — semaines 0 (Scratch), 1 (C), 2 (Arrays), 3 (Algorithms), 4 (Memory), 5 (Data Structures), 6 (Python), 7 (SQL), *AI* (Artificial Intelligence), 8 (HTML, CSS, JavaScript), 9 (Flask), 10 (The End), ainsi que l'index des séries de problèmes.

**Note sur le traitement des sources.** L'ossature de cet essai — l'ordre des sujets, le choix des exemples canoniques (l'annuaire téléphonique, la main qui compte en binaire, l'échec de la fonction d'échange sans pointeurs, le compte bancaire concurrent), la grille d'évaluation en correction-conception-style, la liste des concepts par semaine — provient directement des deux sources. Les développements qui les prolongent, notamment sur la localité de cache, la borne inférieure du tri par comparaison, les coûts amortis, les limites théoriques des expressions régulières, l'analogie entre injection SQL et injection de prompt, et l'évolution des recommandations en matière de mots de passe, sont des extensions assumées : ils ne figurent pas explicitement dans le cours mais découlent de ce qu'il pose. Ils sont signalés comme tels dans le texte chaque fois que la distinction importe.

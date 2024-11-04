// DoublingParallel-Fibonacci Fast Calculation in Go
// Ce programme est une version traduite et adaptée d'un code en C# pour calculer un nombre de Fibonacci en utilisant la méthode de "Doubling".
// Il exploite une pile de tuples pour générer les valeurs et utilise sync.Pool pour optimiser la gestion des ressources.

package main

import (
	"fmt"
	"math/big"
	"time"
)

// Déclaration des structures utilisées
type Tuple struct {
	val1 int
	val2 int
}

type StackElement struct {
	index  int
	values Tuple
}

// Déclaration des variables globales
var fiboKnown []TupleBig
var stack []StackElement
var number int

// TupleBig encapsule l'indice et la valeur du nombre de Fibonacci
type TupleBig struct {
	index int
	value *big.Int
}

// FibCalculator encapsule les variables big.Int réutilisables
type FibCalculator struct {
	a, b, c, temp *big.Int
}

// NewFibCalculator crée une nouvelle instance de FibCalculator
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		a:    big.NewInt(0),
		b:    big.NewInt(1),
		c:    new(big.Int),
		temp: new(big.Int),
	}
}

// ReadNumber permet de lire l'entrée de l'utilisateur
func ReadNumber() {
	fmt.Println("Note: F(0) = 0, F(1) = 1, F(2) = 1, n >= 2")
	fmt.Println("Entrez n, où n est le n-ième nombre de Fibonacci: ")
	fmt.Scan(&number)
	if number < 2 {
		ReadNumber()
	}
}

// CreateStack construit la pile d'éléments nécessaires au calcul
func CreateStack(stack *[]StackElement) {
	for i := len(*stack) - 1; i >= 0; i-- {
		element := (*stack)[i]
		if contains(*stack, element.values.val1) {
			continue
		}
		*stack = append(*stack, StackElement{index: element.values.val1, values: Tuple{val1: element.values.val1 / 2, val2: element.values.val1/2 + 1}})
		if contains(*stack, element.values.val2) {
			continue
		}
		*stack = append(*stack, StackElement{index: element.values.val2, values: Tuple{val1: element.values.val2 / 2, val2: element.values.val2/2 + 1}})
		CreateStack(stack)
	}
}

// contains vérifie si la pile contient un élément avec l'indice donné
func contains(stack []StackElement, index int) bool {
	for _, el := range stack {
		if el.index == index {
			return true
		}
	}
	return false
}

// CreateFibonacci génère les valeurs de Fibonacci en utilisant la méthode "Doubling"
func CreateFibonacci() {
	for i := len(stack) - 1; i >= 0; i-- {
		el := stack[i]
		val1 := findValue(el.values.val1)
		val2 := findValue(el.values.val2)
		var value *big.Int
		if el.index%2 == 0 {
			value = new(big.Int).Mul(val1, new(big.Int).Sub(new(big.Int).Mul(big.NewInt(2), val2), val1))
		} else {
			temp1 := new(big.Int).Mul(val1, val1)
			temp2 := new(big.Int).Mul(val2, val2)
			value = new(big.Int).Add(temp1, temp2)
		}
		fiboKnown = append(fiboKnown, TupleBig{index: el.index, value: value})
	}
}

// findValue recherche une valeur dans fiboKnown par l'indice
func findValue(index int) *big.Int {
	for _, el := range fiboKnown {
		if el.index == index {
			return el.value
		}
	}
	return big.NewInt(0)
}

func main() {
	// Initialiser les valeurs connues de Fibonacci
	fiboKnown = []TupleBig{
		{index: 0, value: big.NewInt(0)},
		{index: 1, value: big.NewInt(1)},
		{index: 2, value: big.NewInt(1)},
	}

	ReadNumber()
	start := time.Now()

	// Initialiser la pile de travail
	stack = []StackElement{{index: number, values: Tuple{val1: number / 2, val2: number/2 + 1}}}
	CreateStack(&stack)

	// Calculer les valeurs de Fibonacci
	CreateFibonacci()
	end := time.Since(start)

	// Afficher le résultat
	result := findValue(number)
	fmt.Printf("Le %d-ième nombre de Fibonacci est: %s. Calcul effectué en %v\n", number, result.String(), end)
}

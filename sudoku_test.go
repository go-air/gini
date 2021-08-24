package gini_test

import (
	"fmt"
	"testing"

	"github.com/go-air/gini"
	"github.com/go-air/gini/z"
)

func BenchmarkSudoku(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Example_sudoku()
	}
}

func Example_sudoku() {
	g := gini.New()
	// 9 rows, 9 cols, 9 boxes, 9 numbers
	// one variable for each triple (row, col, n)
	// indicating whether or not the number n
	// appears in position (row,col).
	var lit = func(row, col, num int) z.Lit {
		n := num
		n += col * 9
		n += row * 81
		return z.Var(n + 1).Pos()
	}

	// add a clause stating that every position on the board has a number
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			for n := 0; n < 9; n++ {
				m := lit(row, col, n)
				g.Add(m)
			}
			g.Add(0)
		}
	}

	// every row has unique numbers
	for n := 0; n < 9; n++ {
		for row := 0; row < 9; row++ {
			for colA := 0; colA < 9; colA++ {
				a := lit(row, colA, n)
				for colB := colA + 1; colB < 9; colB++ {
					b := lit(row, colB, n)
					g.Add(a.Not())
					g.Add(b.Not())
					g.Add(0)
				}
			}
		}
	}

	// every column has unique numbers
	for n := 0; n < 9; n++ {
		for col := 0; col < 9; col++ {
			for rowA := 0; rowA < 9; rowA++ {
				a := lit(rowA, col, n)
				for rowB := rowA + 1; rowB < 9; rowB++ {
					b := lit(rowB, col, n)
					g.Add(a.Not())
					g.Add(b.Not())
					g.Add(0)
				}
			}
		}
	}

	// function adding constraints stating that every box on the board
	// rooted at x, y has unique numbers
	var box = func(x, y int) {
		// all offsets w.r.t. root x,y
		offs := []struct{ x, y int }{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}}
		// all numbers
		for n := 0; n < 9; n++ {
			for i, offA := range offs {
				a := lit(x+offA.x, y+offA.y, n)
				for j := i + 1; j < len(offs); j++ {
					offB := offs[j]
					b := lit(x+offB.x, y+offB.y, n)
					g.Add(a.Not())
					g.Add(b.Not())
					g.Add(0)
				}
			}
		}
	}

	// every box has unique numbers
	for x := 0; x < 9; x += 3 {
		for y := 0; y < 9; y += 3 {
			box(x, y)
		}
	}
	if g.Solve() != 1 {
		fmt.Printf("error, unsat sudoku.\n")
		return
	}
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			for n := 0; n < 9; n++ {
				if g.Value(lit(row, col, n)) {
					fmt.Printf("%d", n+1)
					break
				}
			}
			if col != 8 {
				fmt.Printf(" ")
			}
		}
		fmt.Printf("\n")

	}
	// Output: 5 2 9 1 3 6 7 4 8
	// 4 3 1 7 8 5 2 9 6
	// 8 7 6 4 9 2 1 3 5
	// 1 6 3 2 4 8 5 7 9
	// 2 4 5 9 1 7 8 6 3
	// 7 9 8 5 6 3 4 1 2
	// 6 5 4 3 2 1 9 8 7
	// 3 1 2 8 7 9 6 5 4
	// 9 8 7 6 5 4 3 2 1
}

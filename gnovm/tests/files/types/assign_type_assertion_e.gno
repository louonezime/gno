package main

type Animal interface {
	eat()
}

type Robot struct {
}

type Dog struct{}

func (Dog) eat() {}

func main() {
	var animal Animal = Dog{}

	var r Robot
	var ok bool

	r, ok = animal.(Dog)

	println(r)
	println(ok)
}

// Error:
// main/files/types/assign_type_assertion_e.gno:20:2: cannot use main.Dog as main.Robot without explicit conversion

// TypeCheckError:
// main/files/types/assign_type_assertion_e.gno:20:10: cannot use animal.(Dog) (comma, ok expression of struct type Dog) as Robot value in assignment

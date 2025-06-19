package main

import "fmt"

// Greeter provides greeting capabilities
type Greeter struct {
	Name string
}

// NewGreeter creates a new Greeter instance
func NewGreeter(name string) *Greeter {
	return &Greeter{Name: name}
}

// Greet prints a greeting message
func (g *Greeter) Greet() string {
	return fmt.Sprintf("Hello, %s!", g.Name)
}

func main() {
	g := NewGreeter("World")
	fmt.Println(g.Greet())
}

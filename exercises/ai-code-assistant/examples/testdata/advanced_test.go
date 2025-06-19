package main

import (
	"fmt"
	"sync"
)

// Person represents a person with a name and age
type Person struct {
	Name string
	Age  int
}

// NewPerson creates a new Person instance
func NewPerson(name string, age int) *Person {
	return &Person{
		Name: name,
		Age:  age,
	}
}

// Greet returns a greeting message
func (p *Person) Greet() string {
	return fmt.Sprintf("Hello, my name is %s and I'm %d years old.", p.Name, p.Age)
}

// Counter is a simple counter with mutex protection
type Counter struct {
	mu    sync.Mutex
	count int
}

// Increment increments the counter
func (c *Counter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

// Value returns the current counter value
func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// CalculateSum calculates the sum of a slice of integers
func CalculateSum(numbers []int) int {
	sum := 0
	for _, n := range numbers {
		sum += n
	}
	return sum
}

// CalculateAverage calculates the average of a slice of integers
func CalculateAverage(numbers []int) float64 {
	if len(numbers) == 0 {
		return 0
	}
	sum := CalculateSum(numbers)
	return float64(sum) / float64(len(numbers))
}

func main() {
	person := NewPerson("Alice", 30)
	fmt.Println(person.Greet())

	counter := &Counter{}
	for i := 0; i < 5; i++ {
		counter.Increment()
	}
	fmt.Printf("Counter value: %d\n", counter.Value())

	nums := []int{1, 2, 3, 4, 5}
	fmt.Printf("Sum: %d\n", CalculateSum(nums))
	fmt.Printf("Average: %.2f\n", CalculateAverage(nums))
	multiply := func(x, y int) int {
		return x * y
	}
	fmt.Printf("2 * 3 = %d\n", multiply(2, 3))

	defer fmt.Println("This will be printed last")
	fmt.Println("This will be printed first")

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Printf("Goroutine %d is running\n", id)
		}(i)
	}
	wg.Wait()
}

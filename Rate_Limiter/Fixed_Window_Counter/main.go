package main

import (
	"fmt"
	"time"
)

func main() {
	fw := NewFixedWindow(2, 5*time.Second)
	fmt.Printf("request admitted %t\n", fw.Admit())
	fmt.Printf("request admitted %t\n", fw.Admit())
	fmt.Printf("request admitted %t\n", fw.Admit())
	fmt.Printf("request admitted %t\n", fw.Admit())
	time.Sleep(5 * time.Second)
	fmt.Printf("request admitted %t\n", fw.Admit())

}

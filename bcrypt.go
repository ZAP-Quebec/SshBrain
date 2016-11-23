package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

var (
	hash string
)

func init() {
	flag.StringVar(&hash, "hash", "", "Hash to compare")
}

func main() {
	flag.Parse()

	fmt.Print("Password: ")
	pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()

	if err != nil {
		fmt.Printf("Error reading password: %s\n", err)
		return
	}

	if hash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(hash), pass); err != nil {
			fmt.Println("Password doesn't match.")
		} else {
			fmt.Println("Password valid.")
		}
	} else {

		hash, err := bcrypt.GenerateFromPassword(pass, bcrypt.DefaultCost)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}

		fmt.Printf("Hash: %s\n", string(hash))
	}
}

package errors

import (
	"fmt"
	"os"
)

func CheckErr(e error) {
	if e != nil {
		panic(e)
	}
}

func HandleErr(e error, message string) {
	if e != nil {
		Error(message)
	}
}

func Error(message string) {
	fmt.Printf("[error] -> %s\n", message)
	os.Exit(1)
}

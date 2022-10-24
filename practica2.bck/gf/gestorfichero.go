package gf

import (
    "log"
    "os"
    "fmt"
    "time"
)

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func LeerFichero() string{
	dat, err := os.ReadFile("file.txt")
    checkError(err)
    fmt.Print(string(dat))

    if err != nil {
        log.Fatalf("unable to read file: %v", err)
    }
    return string(dat)
}

func EscribirFichero(fragmento string){
	file, err := os.OpenFile("temp.txt", os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        log.Println(err)
    }
    defer file.Close()
    time.Sleep(2 * time.Second)
    if _, err := file.WriteString(fragmento); err != nil {
        log.Fatal(err)
    }
}

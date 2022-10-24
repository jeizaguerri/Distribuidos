
package main

import (
    "fmt"
    "gf"
	"strconv"
    "os"
)



func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func main() {
	me, err := (strconv.Atoi(os.Args[1]))
	checkError(err)
	usersFile := "ms/users.txt"

	
	gestor:= gf.New(me, 1, usersFile)
	
	var i int
	fmt.Scanf("%d", i)
	for{

	//SC
	contenido := gf.LeerFichero(gestor)
	fmt.Println(me,": ", contenido)
	}
}
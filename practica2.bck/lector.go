
package main

import (
	"ra"
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

	

	ricart := ra.New(me, usersFile)
	go ra.TratarPeticiones(ricart)

	var i int
	fmt.Scanf("%d", i)
	for{
	//Pre - protocol
	ra.PreProtocol(ricart)

	//SC
	gf.EscribirFichero(strconv.Itoa(ra.OurSeqNum))
	//gestor.LeerFichero()
	fmt.Println(me," Hola")

	ra.PostProtocol(ricart)

	}
}
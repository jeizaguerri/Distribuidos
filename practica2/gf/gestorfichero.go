package gf

import (
    "log"
    "os"
    "fmt"
    "time"
    "ra"
    "os/exec"
)

type gfStruct struct{
    Ricart *ra.RASharedDB
}

func New(me int, op_type int, usersFile string) (*gfStruct) {

    ricart := ra.New(me, op_type, usersFile)
    gf := gfStruct{ricart}
    go ra.TratarPeticiones(gf.Ricart)
    
    return &gf
}


func scpCopy(user string,adress string) {
	//export PATH=$PATH:/usr/local/go/bin
	fmt.Println("antes del exec")
    cmd := exec.Command("scp", "file.txt", user+"@"+adress+":"+"$HOME/Documentos/3/dist/eina-ssdd/practica1/file.txt")
	
	fmt.Println("despues")
	err := cmd.Run()
	fmt.Println("despues run")
	checkError(err)
	fmt.Println("despues err")
}





func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func LeerFichero(gf *gfStruct) string{
    ra.PreProtocol(gf.Ricart)
    //SC
	dat, err := os.ReadFile("temp.txt")
    checkError(err)
    fmt.Print(string(dat))

    if err != nil {
        log.Fatalf("unable to read file: %v", err)
    }
    //End SC
    ra.PostProtocol(gf.Ricart)
    return string(dat)
}

func EscribirFichero(fragmento string, gf *gfStruct){
    ra.PreProtocol(gf.Ricart)
    //SC
	file, err := os.OpenFile("temp.txt", os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        log.Println(err)
    }
    defer file.Close()
    time.Sleep(2 * time.Second)
    if _, err := file.WriteString(fragmento); err != nil {
        log.Fatal(err)
    }

    //Mandar cambio al resto de nodos

    //End SC
    ra.PostProtocol(gf.Ricart)

}

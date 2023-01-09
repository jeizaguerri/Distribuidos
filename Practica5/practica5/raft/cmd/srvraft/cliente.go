package main

import (
	//"errors"
	"fmt"
	//"log"
	//"net"
	"net/rpc"
	"os"
	"raft/internal/comun/rpctimeout"
	//"strconv"
	//"strings"
	"time"
)

type TipoOperacion struct {
	Operacion string // La operaciones posibles son "leer" y "escribir"
	Clave     string
	Valor     string // en el caso de la lectura Valor = ""
}

type ResultadoRemoto struct {
	ValorADevolver string
	IndiceRegistro int
	EstadoParcial
}

type EstadoParcial struct {
	Mandato int
	EsLider bool
	IdLider int
}

func main() {

	time.Sleep(1000 * time.Millisecond)

	var nodos []rpctimeout.HostPort
	// Resto de argumento son los end points como strings
	// De todas la replicas-> pasarlos a HostPort
	for _, endPoint := range os.Args[1:] {
		nodos = append(nodos, rpctimeout.HostPort(endPoint))
	}
	var reply ResultadoRemoto
				
			var args TipoOperacion = TipoOperacion{"escribir", "1", "a"}
			for i := 0; i < len(nodos); i++ {  //envia a todos los nodos pero solo el lider responde con los datos

					fmt.Println("enviando peticion de voto al nodo ", i)
				client, err := rpc.Dial("tcp", (string)(nodos[i]))
				if err != nil {
					fmt.Println("dialing:", err)
				}
			
				err = client.Call("NodoRaft.SometerOperacionRaft", args, &reply)
				if err != nil {
					fmt.Println("Nodoraft error:", err)
				}

				if reply.ValorADevolver != "false" {
					if args.Operacion == "leer" {
						fmt.Println("Valor leido = ",reply.ValorADevolver)
					}else{
						fmt.Println("Valor escrito")
					}
				}
			}

			time.Sleep(1000 * time.Millisecond)

			args = TipoOperacion{"escribir", "2", "b"}
			for i := 0; i < len(nodos); i++ {  //envia a todos los nodos pero solo el lider responde con los datos

					fmt.Println("enviando peticion de voto al nodo ", i)
				client, err := rpc.Dial("tcp", (string)(nodos[i]))
				if err != nil {
					fmt.Println("dialing:", err)
				}
			
				err = client.Call("NodoRaft.SometerOperacionRaft", args, &reply)
				if err != nil {
					fmt.Println("Nodoraft error:", err)
				}

				if reply.ValorADevolver != "false" {
					if args.Operacion == "leer" {
						fmt.Println("Valor leido = ",reply.ValorADevolver)
					}else{
						fmt.Println("Valor escrito")
					}
				}
			}

			time.Sleep(1000 * time.Millisecond)

			args = TipoOperacion{"leer", "1", ""}
			for i := 0; i < len(nodos); i++ {

					fmt.Println("enviando peticion de voto al nodo ", i)
				client, err := rpc.Dial("tcp", (string)(nodos[i]))
				if err != nil {
					fmt.Println("dialing:", err)
				}
			
				err = client.Call("NodoRaft.SometerOperacionRaft", args, &reply)
				if err != nil {
					fmt.Println("Nodoraft error:", err)
				}

				if reply.ValorADevolver != "false" {
					if args.Operacion == "leer" {
						fmt.Println("Valor leido = ",reply.ValorADevolver)
					}else{
						fmt.Println("Valor escrito")
					}
				}
			}

			time.Sleep(1000 * time.Millisecond)

			args = TipoOperacion{"escribir", "2", ""}

			for i := 0; i < len(nodos); i++ { //envia a todos los nodos pero solo el lider responde con los datos

				fmt.Println("enviando peticion de voto al nodo ", i)
				client, err := rpc.Dial("tcp", (string)(nodos[i]))
				if err != nil {
					fmt.Println("dialing:", err)
				}
			
				err = client.Call("NodoRaft.SometerOperacionRaft", args, &reply)
				if err != nil {
					fmt.Println("Nodoraft error:", err)
				}

				if reply.ValorADevolver != "false" {
					if args.Operacion == "leer" {
						fmt.Println("Valor leido = ",reply.ValorADevolver)
					}else{
						fmt.Println("Valor escrito")
					}
				}
			}


		
}

/*
* AUTOR: Rafael Tolosana Calasanz
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: septiembre de 2021
* FICHERO: ricart-agrawala.go
* DESCRIPCIÓN: Implementación del algoritmo de Ricart-Agrawala Generalizado en Go
*/
package ra

import (
    "ms"
    "sync"
    "fmt" 
)

const N = 2


type Request struct{
    Clock   [N+1]int
    Pid     int   
    op_t    int
}

type Reply struct{}

//const Exclude = [2][2]bool {{false,true},{true,true}}

type RASharedDB struct {
    me  int//Numero unico
    op_type     int //Read 0, Write 1

    OurSeqNum   [N+1]int
    HigSeqNum   [N+1]int    //Numero de secuencia mas alto visto en un mensaje REQUEST recibido
    OutRepCnt   int     //Numero de respuestas esperadas
    ReqCS       bool    //Pidiendo entrar a la SC
    RepDefd     [N+1]bool   //RepDefd[j] = true -> j está esperando a un mensaje de respuesta
    ms          *ms.MessageSystem 
    done        chan bool
    chrep       chan bool
    Mutex       sync.Mutex // mutex para proteger concurrencia sobre las variables
    Exclude     [2][2]bool
    // TODO: completar
}
    
    

func mayor_que(clk1 [N+1]int, clk2 [N+1]int) (bool){
    for i := 1; i <= N; i++ {
        if(clk1[i] < clk2[i]){
            return false
        }
    }
    return true
}

func igual_que(clk1 [N+1]int, clk2 [N+1]int) (bool){
    for i := 1; i <= N; i++ {
        if(clk1[i] != clk2[i]){
            return false
        }
    }
    return true
}

func son_comparables(clk1 [N+1]int, clk2 [N+1]int) (bool){
    return mayor_que(clk1,clk2) || mayor_que(clk2,clk1)
}


func New(me int, op_type int, usersFile string) (*RASharedDB) {

    messageTypes := []ms.Message{Request{}, Reply{}}
    msgs := ms.New(me, usersFile, messageTypes)
    ra := RASharedDB{me, op_type, [N+1]int{}, [N+1]int{}, 0, false, [N+1]bool{}, &msgs,  make(chan bool),  make(chan bool), sync.Mutex{}, [2][2]bool{{false,true},{true,true}}}
    return &ra
}

//Pre: Verdad
//Post: Realiza  el  PreProtocol  para el  algoritmo de
//      Ricart-Agrawala Generalizado
func PreProtocol(ra *RASharedDB) (){
    //Esperamos al mutex para modificar las variables
    ra.Mutex.Lock()
    ra.ReqCS = true
    ra.OurSeqNum = ra.HigSeqNum
    ra.OurSeqNum[ra.me] = ra.OurSeqNum[ra.me] +1
    ra.Mutex.Unlock()
    fmt.Println(ra.me," en la previa")
    ra.OutRepCnt = N -1 ;

    //Enviamos mensaje a todos los demás procesos
    for j := 1; j <= N; j++ {
        if(j != ra.me){
            ra.ms.Send(j, Request{ra.OurSeqNum,ra.me, ra.op_type})
        }
    }
    fmt.Println("Peticiones enviadas")

    // Esperar a recibir respuesta de los otros nodos
    <- ra.chrep
    // Acceder a SC
}

//Pre: Verdad
//Post: Realiza  el  PostProtocol  para el  algoritmo de
//      Ricart-Agrawala Generalizado
func PostProtocol(ra *RASharedDB)(){
    //Salir de la SC
    ra.ReqCS = false
    // Enviar mensaje ack a tdoso los mensajes postergados
    for j := 1; j <= N; j++ {
        if(ra.RepDefd[j]){
            ra.RepDefd[j] = false
            ra.ms.Send(j, Reply{})
        }
    }
    fmt.Println("Postprotocol")
    fmt.Println("")
    fmt.Println("")

}

func TratarPeticiones(ra *RASharedDB) (){
    for {
        rec := ra.ms.Receive() // se bloquea aquí si no hay nada en el mailbox
        fmt.Println("Llega peticion")

        switch rec.(type){
        case Reply:
            fmt.Println("Es reply")
            //Si se recibe una respuesta
            ra.OutRepCnt --
            if(ra.OutRepCnt <= 0){
                ra.chrep <- true
            }
        default:
            fmt.Println("Es request")
            //Codigo de tratamiento de requests
            defer_it := false
            //Actualizar maximo numero de secuencia
            var req Request = rec.(Request)
            fmt.Println("ra: ", ra.OurSeqNum)
            fmt.Println("req: ", req.Clock)
            //ra.OurSeqNum[ra.me] = ra.OurSeqNum[ra.me] + 1
            //Comparar relojes
            if(son_comparables(ra.OurSeqNum, req.Clock)){
                fmt.Println("Es comparable")
                //Escoger el mayor
                if(mayor_que(req.Clock, ra.OurSeqNum)){
                    fmt.Println("Es mayor")
                    ra.HigSeqNum = req.Clock
                }
            }else{
                fmt.Println("No es comparable")
                if(req.Pid > ra.me){
                    ra.HigSeqNum = req.Clock

                }
            }
            
            //Esperar al mutex para acceder a las variables
            ra.Mutex.Lock()
            defer_it = ra.ReqCS && ((mayor_que(req.Clock, ra.OurSeqNum)) || (igual_que(req.Clock, ra.OurSeqNum) && req.Pid > ra.me)) && (ra.Exclude[ra.op_type][req.op_t])

            ra.Mutex.Unlock()
            if(defer_it){
                ra.RepDefd[req.Pid] = true
                fmt.Println("Añadido a defered")

            }else{
                fmt.Println("Respondiendo")
                ra.ms.Send(req.Pid, Reply{})
            }
        }
    }
}

func (ra *RASharedDB) Stop(){
    ra.ms.Stop()
    ra.done <- true
}





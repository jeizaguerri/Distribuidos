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
    Clock   int
    Pid     int   
}

type Reply struct{}

type RASharedDB struct {
    me  int//Numero unico

    OurSeqNum   int
    HigSeqNum   int     //Numero de secuencia mas alto visto en un mensaje REQUEST recibido
    OutRepCnt   int     //Numero de respuestas esperadas
    ReqCS       bool    //Pidiendo entrar a la SC
    RepDefd     [N+1]bool   //RepDefd[j] = true -> j está esperando a un mensaje de respuesta
    ms          *ms.MessageSystem 
    done        chan bool
    chrep       chan bool
    Mutex       sync.Mutex // mutex para proteger concurrencia sobre las variables
    
    // TODO: completar
}



func New(me int, usersFile string) (*RASharedDB) {
    messageTypes := []ms.Message{Request{}, Reply{}}
    msgs := ms.New(me, usersFile, messageTypes)
    ra := RASharedDB{me, 0, 0, 0, false, [N+1]bool{}, &msgs,  make(chan bool),  make(chan bool), sync.Mutex{}}
    return &ra
}

//Pre: Verdad
//Post: Realiza  el  PreProtocol  para el  algoritmo de
//      Ricart-Agrawala Generalizado
func PreProtocol(ra *RASharedDB) (){
    //Esperamos al mutex para modificar las variables
    ra.Mutex.Lock()
    ra.ReqCS = true
    ra.OurSeqNum = ra.HigSeqNum + 1;
    ra.Mutex.Unlock()
    fmt.Println(ra.me," en la previa")
    ra.OutRepCnt = N -1 ;

    //Enviamos mensaje a todos los demás procesos
    for j := 1; j <= N; j++ {
        if(j != ra.me){
            ra.ms.Send(j, Request{ra.OurSeqNum,ra.me})
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
            if(req.Clock > ra.HigSeqNum){
                ra.HigSeqNum = req.Clock
            }
            
            //Esperar al mutex para acceder a las variables
            ra.Mutex.Lock()
            defer_it = ra.ReqCS && ((req.Clock > ra.OurSeqNum) || (req.Clock == ra.OurSeqNum && req.Pid > ra.me))
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





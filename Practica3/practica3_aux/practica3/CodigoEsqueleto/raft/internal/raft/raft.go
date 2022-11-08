// Escribir vuestro código de funcionalidad Raft en este fichero
//

package raft

//
// API
// ===
// Este es el API que vuestra implementación debe exportar
//
// nodoRaft = NuevoNodo(...)
//   Crear un nuevo servidor del grupo de elección.
//
// nodoRaft.Para()
//   Solicitar la parado de un servidor
//
// nodo.ObtenerEstado() (yo, mandato, esLider)
//   Solicitar a un nodo de elección por "yo", su mandato en curso,
//   y si piensa que es el msmo el lider
//
// nodoRaft.SometerOperacion(operacion interface()) (indice, mandato, esLider)

// type AplicaOperacion


import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"math/rand"
	"sync"
	"time"
	"net/rpc"

	"raft/internal/comun/rpctimeout"
)


const (
	// Constante para fijar valor entero no inicializado
	IntNOINICIALIZADO = -1

	//  false deshabilita por completo los logs de depuracion
	// Aseguraros de poner kEnableDebugLogs a false antes de la entrega
	kEnableDebugLogs = true

	// Poner a true para logear a stdout en lugar de a fichero
	kLogToStdout = false

	// Cambiar esto para salida de logs en un directorio diferente
	kLogOutputDir = "./logs_raft/"

	LIDER = 0
	SEGUIDOR = 1
	CANDIDATO = 2

	T_HEARTBEAT = 900

	T_TIMEOUT_MIN = 1000
	T_TIMEOUT_MAX = 2000
)

type TipoOperacion struct {
	Operacion string  // La operaciones posibles son "leer" y "escribir"
	Clave string
	Valor string    // en el caso de la lectura Valor = ""
}


// A medida que el nodo Raft conoce las operaciones de las  entradas de registro
// comprometidas, envía un AplicaOperacion, con cada una de ellas, al canal
// "canalAplicar" (funcion NuevoNodo) de la maquina de estados 
type AplicaOperacion struct {
	Indice int  // en la entrada de registro
	Operacion TipoOperacion
}

type EntradaLog struct{
	State int		//Estado
	Term int		//Mandato
}

// Tipo de dato Go que representa un solo nodo (réplica) de raft
//
type NodoRaft struct {
	Mux   sync.Mutex       // Mutex para proteger acceso a estado compartido

	// Host:Port de todos los nodos (réplicas) Raft, en mismo orden
	Nodos []rpctimeout.HostPort
	Yo    int           // indice de este nodos en campo array "nodos"
	IdLider int
	// Utilización opcional de este logger para depuración
	// Cada nodo Raft tiene su propio registro de trazas (logs)
	Logger *log.Logger

	// Vuestros datos aqui.
	Log []EntradaLog
	// mirar figura 2 para descripción del estado que debe mantenre un nodo Raft
	
	CurrentTerm 	int		// latest term server has seen (initialized to 0 on first boot, increases monotonically)
	VotedFor 		int		//candidateId that received vote in current
	
	
	CommitIndex		int		//index of highest log entry known to be committed (initialized to 0, increases monotonically)
	LastApplied 	int   	// index of highest log entry applied to state machine (initialized to 0, increases monotonically)

	NextIndex		[]int	//for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	MatchIndex		[]int 	//for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	Estado			int

	Done			chan bool	//Acaba la votacion
	Votos			int // votos que recibe el nodo como candidato
}



// Creacion de un nuevo nodo de eleccion
//
// Tabla de <Direccion IP:puerto> de cada nodo incluido a si mismo.
//
// <Direccion IP:puerto> de este nodo esta en nodos[yo]
//
// Todos los arrays nodos[] de los nodos tienen el mismo orden

// canalAplicar es un canal donde, en la practica 5, se recogerán las
// operaciones a aplicar a la máquina de estados. Se puede asumir que
// este canal se consumira de forma continúa.
//
// NuevoNodo() debe devolver resultado rápido, por lo que se deberían
// poner en marcha Gorutinas para trabajos de larga duracion
func NuevoNodo(nodos []rpctimeout.HostPort, yo int, 
						canalAplicarOperacion chan AplicaOperacion) *NodoRaft {
	nr := &NodoRaft{}
	nr.Nodos = nodos
	nr.Yo = yo
	nr.IdLider = -1

	if kEnableDebugLogs {
		nombreNodo := nodos[yo].Host() + "_" + nodos[yo].Port()
		logPrefix := fmt.Sprintf("%s", nombreNodo)
		
		fmt.Println("LogPrefix: ", logPrefix)

		if kLogToStdout {
			nr.Logger = log.New(os.Stdout, nombreNodo + " -->> ",
								log.Lmicroseconds|log.Lshortfile)
		} else {
			err := os.MkdirAll(kLogOutputDir, os.ModePerm)
			if err != nil {
				panic(err.Error())
			}
			logOutputFile, err := os.OpenFile(fmt.Sprintf("%s/%s.txt",
			  kLogOutputDir, logPrefix), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				panic(err.Error())
			}
			nr.Logger = log.New(logOutputFile, 
						   logPrefix + " -> ", log.Lmicroseconds|log.Lshortfile)
		}
		nr.Logger.Println("logger initialized")
	} else {
		nr.Logger = log.New(ioutil.Discard, "", 0)
	}

	// Añadir codigo de inicialización
	nr.CurrentTerm = 0
	nr.VotedFor = -1
	nr.CommitIndex = 0
	nr.LastApplied = 0
	
	nr.NextIndex = []int{}
	nr.MatchIndex = []int{}

	nr.Estado = SEGUIDOR
	nr.Done = make(chan bool)
	go nr.GestionNodo()
	return nr
}


// Metodo Para() utilizado cuando no se necesita mas al nodo
//
// Quizas interesante desactivar la salida de depuracion
// de este nodo
//
func (nr *NodoRaft) para() {
	go func() {time.Sleep(5 * time.Millisecond); os.Exit(0) } ()
}

// Devuelve "yo", mandato en curso y si este nodo cree ser lider
//
// Primer valor devuelto es el indice de este  nodo Raft el el conjunto de nodos 
// la operacion si consigue comprometerse.
// El segundo valor es el mandato en curso
// El tercer valor es true si el nodo cree ser el lider
// Cuarto valor es el lider, es el indice del líder si no es él
func (nr *NodoRaft) obtenerEstado() (int, int, bool, int) {
	var yo int = nr.Yo
	var mandato int
	var esLider bool
	var idLider int = nr.IdLider
	
	nr.Mux.Lock()
	yo = nr.Yo
	mandato = nr.CurrentTerm
	esLider = nr.Estado == LIDER
	idLider = nr.IdLider
	nr.Mux.Unlock()

	return yo, mandato, esLider, idLider
}

// El servicio que utilice Raft (base de datos clave/valor, por ejemplo)
// Quiere buscar un acuerdo de posicion en registro para siguiente operacion
// solicitada por cliente.

// Si el nodo no es el lider, devolver falso
// Sino, comenzar la operacion de consenso sobre la operacion y devolver en
// cuanto se consiga
// 
// No hay garantia que esta operacion consiga comprometerse en una entrada de
// de registro, dado que el lider puede fallar y la entrada ser reemplazada
// en el futuro.
// Primer valor devuelto es el indice del registro donde se va a colocar 
// la operacion si consigue comprometerse.
// El segundo valor es el mandato en curso
// El tercer valor es true si el nodo cree ser el lider
// Cuarto valor es el lider, es el indice del líder si no es él
func (nr *NodoRaft) someterOperacion(operacion TipoOperacion) (int, int,
															bool, int, string) {
	indice := -1
	mandato := -1
	EsLider := false
	idLider := -1
	valorADevolver := ""
	

	// Vuestro codigo aqui
	

	return indice, mandato, EsLider, idLider, valorADevolver
}


// -----------------------------------------------------------------------
// LLAMADAS RPC al API
//
// Si no tenemos argumentos o respuesta estructura vacia (tamaño cero)
type Vacio struct{}

func (nr * NodoRaft) ParaNodo(args Vacio, reply *Vacio) error {
	defer nr.para()
	return nil
}

type EstadoParcial struct {
	Mandato	int
	EsLider bool
	IdLider	int
}

type EstadoRemoto struct {
	IdNodo	int
	EstadoParcial
}

func (nr *NodoRaft) ObtenerEstadoNodo(args Vacio, reply *EstadoRemoto) error {
	reply.IdNodo,reply.Mandato,reply.EsLider,reply.IdLider = nr.obtenerEstado()
	return nil
}

type ResultadoRemoto struct {
	ValorADevolver string
	IndiceRegistro int
	EstadoParcial
}

func (nr *NodoRaft) SometerOperacionRaft(operacion TipoOperacion,
												reply *ResultadoRemoto) error {
	reply.IndiceRegistro,reply.Mandato, reply.EsLider,
			reply.IdLider,reply.ValorADevolver = nr.someterOperacion(operacion)
	return nil
}

// -----------------------------------------------------------------------
// LLAMADAS RPC protocolo RAFT
//
// Structura de ejemplo de argumentos de RPC PedirVoto.
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
//
type ArgsPeticionVoto struct {
	// Vuestros datos aqui
	Term int	//	candidate’s term
	CandidateID int	//candidate requesting vote
	LastLogIndex int	//index of candidate’s last log entry (§5.4)
	LastLogTerm int	//term of candidate’s last log entry (§5.4)
}


// Structura de ejemplo de respuesta de RPC PedirVoto,
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
//
//
type RespuestaPeticionVoto struct {
	// Vuestros datos aqui
	Term			int		//currentTerm, for candidate to update itself
	VoteGranted		bool	//true means candidate received vote
}


// Metodo para RPC PedirVoto
//
func (nr *NodoRaft) PedirVoto (peticion *ArgsPeticionVoto,
										reply *RespuestaPeticionVoto) error {
	// Vuestro codigo aqui
	nr.Logger.Println("Peticion de voto recibida ", peticion, "Mandato: ", nr.CurrentTerm, "votado:", nr.VotedFor)
	if(peticion.Term < nr.CurrentTerm){
		//denegar
		nr.Logger.Println("a")
		reply.Term = nr.CurrentTerm
		reply.VoteGranted = false
		

	} else if(peticion.Term > nr.CurrentTerm){
		//aceptar
		nr.Logger.Println("b")
		nr.Mux.Lock()
		nr.Estado = SEGUIDOR
		nr.CurrentTerm = peticion.Term
		nr.VotedFor = peticion.CandidateID
		
		nr.Mux.Unlock()
		nr.Done<-true

		reply.Term = nr.CurrentTerm
		reply.VoteGranted = true

	} else if(nr.VotedFor != peticion.CandidateID && nr.VotedFor != -1){
		//denegar
		nr.Logger.Println("c")
		reply.Term = nr.CurrentTerm
		reply.VoteGranted = false

	} else{
		//aceptar
		nr.Logger.Println("d")
		nr.Mux.Lock()
		nr.Logger.Println("d1")
		nr.Estado = SEGUIDOR
		nr.VotedFor = peticion.CandidateID
		nr.Logger.Println("d3")
		
		nr.Logger.Println("d4")
		nr.Mux.Unlock()
		cosa := true
		nr.Done<-cosa
		nr.Logger.Println("d5")
		reply.Term = nr.CurrentTerm
		reply.VoteGranted = true
	}
	nr.Logger.Println("Reply: " ,reply)
	return nil
}



type ArgAppendEntries struct {
	
	Term			int		//leader’s term
	LeaderId		int		//so follower can redirect clients
	PrevLogIndex	int		//index of log entry immediately preceding new ones
	PrevLogTerm		int		//term of prevLogIndex entry
	Entries			[]EntradaLog	//log entries to store (empty for heartbeat; may send more than one for efficiency)
	LeaderCommit	int		//leader’s commitIndex
}

type Results struct {
	
	Term 		int			//currentTerm, for leader to update itself
	Success		bool		//true if follower contained entry matching prevLogIndex and prevLogTerm
}


// Metodo de tratamiento de llamadas RPC AppendEntries
func (nr *NodoRaft) AppendEntries(args *ArgAppendEntries,
													  results *Results) error {
	if(args.Term > nr.CurrentTerm){
		//Aceptar, cambiar de mandato y reset de voted for
		nr.Mux.Lock()
		nr.Estado = SEGUIDOR
		nr.CurrentTerm = args.Term
		nr.VotedFor = -1
		
		nr.Mux.Unlock()
		nr.Done<-true

		results.Term = nr.CurrentTerm
		results.Success = true
	}else if(args.Term == nr.CurrentTerm && args.LeaderId == nr.IdLider){
		//Aceptar sin cambiar de mandato
		results.Term = nr.CurrentTerm
		results.Success = true
	}else{
		//Denegar
		results.Term = nr.CurrentTerm
		results.Success = false
	}

	return nil
}


// ----- Metodos/Funciones a utilizar como clientes
//
//

// Ejemplo de código enviarPeticionVoto
//
// nodo int -- indice del servidor destino en nr.nodos[]
//
// args *RequestVoteArgs -- argumentos par la llamada RPC
//
// reply *RequestVoteReply -- respuesta RPC
//
// Los tipos de argumentos y respuesta pasados a CallTimeout deben ser
// los mismos que los argumentos declarados en el metodo de tratamiento
// de la llamada (incluido si son punteros
//
// Si en la llamada RPC, la respuesta llega en un intervalo de tiempo,
// la funcion devuelve true, sino devuelve false
//
// la llamada RPC deberia tener un timout adecuado.
//
// Un resultado falso podria ser causado por una replica caida,
// un servidor vivo que no es alcanzable (por problemas de red ?),
// una petición perdida, o una respuesta perdida
//
// Para problemas con funcionamiento de RPC, comprobar que la primera letra
// del nombre  todo los campos de la estructura (y sus subestructuras)
// pasadas como parametros en las llamadas RPC es una mayuscula,
// Y que la estructura de recuperacion de resultado sea un puntero a estructura
// y no la estructura misma.
//
func (nr *NodoRaft) enviarPeticionVoto(nodo int, args *ArgsPeticionVoto,
											reply *RespuestaPeticionVoto) bool {

	nr.Logger.Println("enviando peticion de voto al nodo ",nodo)
	client, err := rpc.Dial("tcp", (string)(nr.Nodos[nodo]))
		if err != nil {
			log.Fatal("dialing:", err)
		}
	
		err = client.Call("NodoRaft.PedirVoto", args, &reply)
	if err != nil {
		log.Fatal("Nodoraft error:", err)
	}
	
	nr.Logger.Println("aaaa")
	return reply.VoteGranted
}


func (nr *NodoRaft) enviarPeticionAppendEntries(nodo int, args *ArgAppendEntries,
	reply *Results) bool {
		
	nr.Logger.Println("enviando enviarPeticionAppendEntries al nodo ",nodo)
	client, err := rpc.Dial("tcp", (string)(nr.Nodos[nodo]))
	if err != nil {
	log.Fatal("dialing:", err)
	}

	err = client.Call("NodoRaft.AppendEntries", args, &reply)
	if err != nil {
	log.Fatal("Nodoraft error:", err)
	}

	return reply.Success
}


func (nr *NodoRaft) mandarHeartbeat(i int){
	args := &ArgAppendEntries{}
	reply := &Results{}
	nr.enviarPeticionAppendEntries(i, args, reply)
	if(reply.Success == false){
		nr.Mux.Lock()
		nr.CurrentTerm = reply.Term
		nr.Estado = SEGUIDOR
		nr.Mux.Unlock()
	}
}

func (nr *NodoRaft) MandarVotacion(i int){
	args := &ArgsPeticionVoto{}
	reply := &RespuestaPeticionVoto{}
	nr.enviarPeticionVoto(i, args, reply)
	nr.Logger.Println("respuesta recibida:", reply)
	if(reply.VoteGranted){
		nr.Mux.Lock()
		nr.Votos++
		if(nr.Votos > len(nr.Nodos)/2){
			nr.Estado = LIDER
			nr.Done <- true
		}
		nr.Mux.Unlock()
	}
}

func (nr *NodoRaft) GestionNodo(){
	time.Sleep(5 * time.Second)
	for{
		if(nr.Estado == LIDER){
			for nr.Estado == LIDER{
				nr.Logger.Println("Lider")
				time.Sleep(T_HEARTBEAT)
				for i := 0; i < len(nr.Nodos); i++ {
					go nr.mandarHeartbeat(i)
				}
			}
			
		}else if (nr.Estado == SEGUIDOR){
			for nr.Estado == SEGUIDOR {
				nr.Logger.Println("Seguidor")
				select{
				case <- nr.Done:
					//Se recibe latido, reset timeout
					nr.Logger.Println("Done recibido")
				case <- time.After((time.Duration)(rand.Intn(T_TIMEOUT_MAX - T_TIMEOUT_MIN) + T_TIMEOUT_MIN)* time.Millisecond):
					//Se termina el timeout, se pasa a candidato
					nr.Mux.Lock()
					nr.Estado = CANDIDATO
					nr.Mux.Unlock()
					nr.Logger.Println("Timeout seguidor")
				
				}
			}
		}else{
			//Candidato
			for nr.Estado == CANDIDATO{
				nr.Logger.Println("Candidato: Empezando votacion")
				nr.Mux.Lock()
				nr.CurrentTerm++
				nr.VotedFor = nr.Yo
				nr.Votos = 1
				nr.Mux.Unlock()
				for i := 0; i < len(nr.Nodos); i++ {
					if i != nr.Yo{
						go nr.MandarVotacion(i)
					}
				}
				select{
				case <- nr.Done:
					//Lider o seguidor
					nr.Logger.Println("Votacion teminada antes de que acabe el tiempo")
				case <- time.After((time.Duration)(rand.Intn(2500 - 1000) + 1000) * time.Millisecond):
						//Se acaba el tiempo de candidato, empezar nueva votacion
						nr.Logger.Println("Tiempo de votacion terminado")
						break
					
				}
			}
				
		}
	}
}




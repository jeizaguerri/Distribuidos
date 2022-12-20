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
	"math"
	"math/rand"
	"net/rpc"
	"os"
	"raft/internal/comun/rpctimeout"
	"sync"
	"time"
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

	LIDER     = 0
	SEGUIDOR  = 1
	CANDIDATO = 2

	T_HEARTBEAT = 200

	T_TIMEOUT_MIN = 2000
	T_TIMEOUT_MAX = 3000

	N_NODOS = 3
)

type TipoOperacion struct {
	Operacion string // La operaciones posibles son "leer" y "escribir"
	Clave     string
	Valor     string // en el caso de la lectura Valor = ""
}

// A medida que el nodo Raft conoce las operaciones de las  entradas de registro
// comprometidas, envía un AplicaOperacion, con cada una de ellas, al canal
// "canalAplicar" (funcion NuevoNodo) de la maquina de estados
type AplicaOperacion struct {
	Indice    int // en la entrada de registro
	Operacion TipoOperacion
}

type EntradaLog struct {
	State     int //Estado
	Term      int //Mandato
	Operacion TipoOperacion
}

// Tipo de dato Go que representa un solo nodo (réplica) de raft
type NodoRaft struct {
	Mux sync.Mutex // Mutex para proteger acceso a estado compartido

	// Host:Port de todos los nodos (réplicas) Raft, en mismo orden
	Nodos   []rpctimeout.HostPort
	Yo      int // indice de este nodos en campo array "nodos"
	IdLider int
	// Utilización opcional de este logger para depuración
	// Cada nodo Raft tiene su propio registro de trazas (logs)
	Logger *log.Logger

	// Vuestros datos aqui.
	Log []EntradaLog
	// mirar figura 2 para descripción del estado que debe mantenre un nodo Raft

	CurrentTerm int // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	VotedFor    int //candidateId that received vote in current

	CommitIndex int //index of highest log entry known to be committed (initialized to 0, increases monotonically)
	LastApplied int // index of highest log entry applied to state machine (initialized to 0, increases monotonically)

	NextIndex  [N_NODOS]int //for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	MatchIndex [N_NODOS]int //for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	Estado int

	Done      chan bool //Acaba la votacion
	heartBeat chan bool //me llega el heartbeat del lider
	Votos     int       // votos que recibe el nodo como candidato

	CanalOperacion chan AplicaOperacion
}

// funcion auxiliar para insertar en una lista y si no hay componente ahí crearlo
// 0 <= index
func insert(a []EntradaLog, index int, value EntradaLog) []EntradaLog {
	if len(a) == index { // nil or empty slice or after last element
		return append(a, value)
	}

	if len(a) < index {
		for i := len(a); i <= len(a)+(index-len(a)); i++ {
			a = append(a, value) //hacemos vario inserts hasta llegar al índice que queremos
		}
		return append(a, value)
	}

	a = append(a[:index+1], a[index:]...) // index < len(a)
	a[index] = value
	return a
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
			nr.Logger = log.New(os.Stdout, nombreNodo+" -->> ",
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
				logPrefix+" -> ", log.Lmicroseconds|log.Lshortfile)
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

	nr.NextIndex = [N_NODOS]int{}
	nr.MatchIndex = [N_NODOS]int{}

	//probablemente no hace falta hacer esto pero porsiacaso
	l := 0
	for l < len(nr.Nodos) {
		nr.NextIndex[l] = 0
		l++
	}

	nr.Estado = SEGUIDOR
	nr.Done = make(chan bool, 1)
	nr.heartBeat = make(chan bool, 1)
	nr.CanalOperacion = canalAplicarOperacion

	go nr.GestionNodo()
	return nr
}

// Metodo Para() utilizado cuando no se necesita mas al nodo
//
// Quizas interesante desactivar la salida de depuracion
// de este nodo
func (nr *NodoRaft) para() {
	go func() { time.Sleep(5 * time.Millisecond); os.Exit(0) }()
}

// Devuelve "yo", mandato en curso y si este nodo cree ser lider
//
// Primer valor devuelto es el indice de este  nodo Raft el el conjunto de nodos
// la operacion si consigue comprometerse.
// El segundo valor es el mandato en curso
// El tercer valor es true si el nodo cree ser el lider
// Cuarto valor es el lider, es el indice del líder si no es él
func (nr *NodoRaft) obtenerEstado() (int, int, bool, int) {
	var yo int
	var mandato int
	var esLider bool
	var idLider int

	nr.Mux.Lock()
	yo = nr.Yo
	mandato = nr.CurrentTerm
	esLider = nr.Estado == LIDER
	idLider = nr.IdLider
	nr.Mux.Unlock()

	return yo, mandato, esLider, idLider
}

// lo que lanzamos como go routines en el lider para someter una entrada
func (nr *NodoRaft) mandarSometer(i int, entries []EntradaLog) {
	args := &ArgAppendEntries{}
	if len(nr.Log) == 1 {
		args = &ArgAppendEntries{nr.CurrentTerm, nr.Yo, 0, 0, entries, nr.CommitIndex} //cambiar arg del append entries (el nil está bien)
	} else {
		args = &ArgAppendEntries{nr.CurrentTerm, nr.Yo, nr.NextIndex[i] - 1, nr.Log[nr.NextIndex[i]-1].Term, entries, nr.CommitIndex} //cambiar arg del append entries (el nil está bien)
	}

	//args := &ArgAppendEntries{nr.CurrentTerm, nr.Yo, nr.NextIndex[i] - 1, nr.Log[nr.NextIndex[i]-1].Term, entries, nr.CommitIndex}
	reply := &Results{}
	nr.enviarPeticionAppendEntries(i, args, reply)
	nr.Logger.Println("respuesta someterOperacion del nodo: ", i, " = ", reply, " miTerm = ", nr.CurrentTerm)
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
	nr.Mux.Lock()
	indice := nr.LastApplied
	mandato := nr.CurrentTerm
	esLider := (nr.IdLider == nr.Yo)
	idLider := nr.IdLider
	valorADevolver := "" //si no es lider devolvera "" (un string vacio)
	nr.Mux.Unlock()

	if !esLider {
		return indice, mandato, esLider, idLider, valorADevolver
	}

	nr.Mux.Lock()
	nr.Log = insert(nr.Log, indice, EntradaLog{nr.Estado, mandato, operacion}) //añadimos al log del lider e incrementamos el last aplied
	nr.LastApplied++
	nr.Logger.Printf("contenido del Log: ", nr.Log)
	nr.Mux.Unlock()

	entries := []EntradaLog{EntradaLog{nr.Estado, mandato, operacion}}
	for i := 0; i < len(nr.Nodos); i++ {
		if i != nr.Yo {
			go nr.mandarSometer(i, entries)
		}
	}

	return indice, mandato, esLider, idLider, valorADevolver
}

// -----------------------------------------------------------------------
// LLAMADAS RPC al API
//
// Si no tenemos argumentos o respuesta estructura vacia (tamaño cero)
type Vacio struct{}

func (nr *NodoRaft) ParaNodo(args Vacio, reply *Vacio) error {
	defer nr.para()
	return nil
}

type EstadoParcial struct {
	Mandato int
	EsLider bool
	IdLider int
}

type EstadoRemoto struct {
	IdNodo int
	EstadoParcial
}

func (nr *NodoRaft) ObtenerEstadoNodo(args Vacio, reply *EstadoRemoto) error {
	reply.IdNodo, reply.Mandato, reply.EsLider, reply.IdLider = nr.obtenerEstado()
	return nil
}

type ResultadoRemoto struct {
	ValorADevolver string
	IndiceRegistro int
	EstadoParcial
}

func (nr *NodoRaft) SometerOperacionRaft(operacion TipoOperacion,
	reply *ResultadoRemoto) error {
	reply.IndiceRegistro, reply.Mandato, reply.EsLider, reply.IdLider, reply.ValorADevolver = nr.someterOperacion(operacion)
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
type ArgsPeticionVoto struct {
	// Vuestros datos aqui
	Term         int //	candidate’s term
	CandidateID  int //candidate requesting vote
	LastLogIndex int //index of candidate’s last log entry (§5.4)
	LastLogTerm  int //term of candidate’s last log entry (§5.4)
}

// Structura de ejemplo de respuesta de RPC PedirVoto,
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
type RespuestaPeticionVoto struct {
	// Vuestros datos aqui
	Term        int  //currentTerm, for candidate to update itself
	VoteGranted bool //true means candidate received vote
}

// Metodo para RPC PedirVoto
func (nr *NodoRaft) PedirVoto(peticion *ArgsPeticionVoto,
	reply *RespuestaPeticionVoto) error {

	nr.Logger.Println("Peticion de voto recibida de ", peticion.CandidateID, "con term =", peticion.Term, " Mi term: ", nr.CurrentTerm, "ult votado:", nr.VotedFor)

	if peticion.Term > nr.CurrentTerm {
		//Actualizar currentTerm
		nr.Mux.Lock()
		nr.Estado = SEGUIDOR
		nr.CurrentTerm = peticion.Term
		nr.VotedFor = -1
		nr.Mux.Unlock()
	}

	if peticion.Term < nr.CurrentTerm {
		//denegar
		reply.Term = nr.CurrentTerm
		nr.Logger.Println("False en pedirVoto por term superior")
		reply.VoteGranted = false
		return nil
	}

	//condicion := false
	LastTerm := 0
	if len(nr.Log) > 0 { // no podemos usar la funcion len porque el tamaño puede dejar de ser real  AJUSTAR VARIABLES EN INICIO Y CREAR NUEVAS
		LastTerm = nr.Log[len(nr.Log)-1].Term
		nr.Logger.Println("Longitud del log > 0")

		//condicion = (nr.LastApplied == peticion.LastLogIndex && peticion.LastLogTerm != nr.Log[peticion.LastLogIndex].Term)
	}

	logOK := (peticion.LastLogTerm > LastTerm) || ((peticion.LastLogTerm == LastTerm) && (peticion.LastLogIndex >= len(nr.Log)))

	//nr.Logger.Println("lastApplied: ", nr.LastApplied, "lastLogIndex: ", peticion.LastLogIndex)
	//////////
	//if (nr.VotedFor != peticion.CandidateID && nr.VotedFor != -1) || (nr.LastApplied > peticion.LastLogIndex) || condicion {
	//	//denegar
	//	nr.Logger.Println("False en pedirVoto porque ya se ha votado o esta desactualizado")
	//	reply.VoteGranted = false
	//	return nil
	//}
	//////////

	if (peticion.Term == nr.CurrentTerm) && logOK && ((nr.VotedFor == peticion.CandidateID) || (nr.VotedFor == -1)) {
		//aceptar
		nr.Mux.Lock()
		nr.IdLider = peticion.CandidateID
		nr.Estado = SEGUIDOR
		nr.VotedFor = peticion.CandidateID

		nr.Mux.Unlock()
		nr.Logger.Println("antes del done aceptar")
		nr.Done <- true
		nr.Logger.Println("despues del done aceptar")
		reply.Term = nr.CurrentTerm
		reply.VoteGranted = true

		nr.Logger.Println("Reply al nodo: ", peticion.CandidateID, " = ", reply.VoteGranted)
		return nil
	} else {
		reply.VoteGranted = false
		return nil
	}

}

type ArgAppendEntries struct {
	Term         int          //leader’s term
	LeaderId     int          //so follower can redirect clients
	PrevLogIndex int          //index of log entry immediately preceding new ones
	PrevLogTerm  int          //term of prevLogIndex entry
	Entries      []EntradaLog //log entries to store (empty for heartbeat; may send more than one for efficiency)
	LeaderCommit int          //leader’s commitIndex
}

type Results struct {
	Term    int  //currentTerm, for leader to update itself
	Success bool //true if follower contained entry matching prevLogIndex and prevLogTerm
}

// Metodo de tratamiento de llamadas RPC AppendEntries
func (nr *NodoRaft) AppendEntries(args *ArgAppendEntries,
	results *Results) error {

	results.Term = nr.CurrentTerm
	//Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)
	if len(nr.Log) > 0 && nr.Log[args.PrevLogIndex].Term != args.PrevLogTerm {
		results.Success = false
		return nil
	}

	//igual en Pedir Voto si el lider tiene un commit index menor le respondemos false pero nos actualizamos el term si es mayor
	if nr.CommitIndex > args.LeaderCommit {
		if args.Term > nr.CurrentTerm {
			nr.Mux.Lock()
			nr.CurrentTerm = args.Term
			nr.Mux.Unlock()
		}
		results.Success = false
		return nil
	}

	if args.Entries != nil {
		// If an existing entry conflicts with a new one (same index but different terms)
		i := 0
		if args.PrevLogIndex < nr.LastApplied {
			for i < (nr.LastApplied-args.PrevLogIndex) && i < len(args.Entries) {
				if nr.Log[args.PrevLogIndex+1+i].Term != args.Entries[i].Term {
					nr.Mux.Lock()
					nr.LastApplied = args.PrevLogIndex + i
					nr.Mux.Unlock()
					results.Success = false
					return nil
				}
				i++
			}

		} else if args.PrevLogIndex > nr.LastApplied {
			results.Success = false
			return nil
		}
	}

	//Append any new entries not already in the log
	if args.Entries != nil {
		i := 0
		for i < len(args.Entries) { // corregir len of entries
			nr.Mux.Lock()
			//nr.Log[args.PrevLogIndex+1+i] = args.Entries[i] es un append y hecho así dará error
			nr.Log = insert(nr.Log, args.PrevLogIndex+1+i, args.Entries[i])
			nr.LastApplied++
			nr.Mux.Unlock()
			i++
		}
		nr.Logger.Printf("contenido del Log: ", nr.Log)
		nr.Logger.Printf("args:  tamaño len", len(args.Entries), " Entries:", args.Entries, " PrevLogIndex", args.PrevLogIndex)
	}

	// If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	if args.LeaderCommit > nr.CommitIndex {
		nr.Mux.Lock()
		nr.CommitIndex = (int)(math.Min((float64)(args.LeaderCommit), (float64)(nr.LastApplied)))
		nr.Mux.Unlock()
	}

	//1. Reply false if term < currentTerm (§5.1)
	if args.Term > nr.CurrentTerm {
		//Aceptar, cambiar de mandato y reset de voted for
		//nr.Logger.Println("lock")
		nr.Mux.Lock()
		nr.Estado = SEGUIDOR
		nr.CurrentTerm = args.Term
		nr.VotedFor = -1
		nr.IdLider = args.LeaderId

		nr.Mux.Unlock()
		//nr.Logger.Println("Unlock")
		nr.Logger.Println("antes del done appendEntries con cambio de mandato")
		nr.heartBeat <- true
		nr.Logger.Println("despues del done appendEntries sin cambio de mandato")

		results.Term = args.Term
		results.Success = true
	} else if args.Term == nr.CurrentTerm && args.LeaderId == nr.IdLider {
		//Aceptar sin cambiar de mandato
		nr.Logger.Println("Done antes de appendEntries sin cambio de mandato")
		nr.heartBeat <- true
		nr.Logger.Println("Done despues de appendEntries sin cambio de mandato")
		results.Term = nr.CurrentTerm
		results.Success = true
	} else {
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
func (nr *NodoRaft) enviarPeticionVoto(nodo int, args *ArgsPeticionVoto,
	reply *RespuestaPeticionVoto) bool {

	nr.Logger.Println("enviando peticion de voto al nodo ", nodo)
	client, err := rpc.Dial("tcp", (string)(nr.Nodos[nodo]))
	if err != nil {
		log.Fatal("dialing:", err)
	}

	err = client.Call("NodoRaft.PedirVoto", args, &reply)
	if err != nil {
		log.Fatal("Nodoraft error:", err)
	}

	//nr.Logger.Println("aaaa")
	return reply.VoteGranted
}

func (nr *NodoRaft) enviarPeticionAppendEntries(nodo int, args *ArgAppendEntries,
	reply *Results) bool {

	nr.Logger.Println("enviando enviarPeticionAppendEntries al nodo ", nodo)
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

// El valor devuelto indica si ha habido error (timeout) o no
func (nr *NodoRaft) enviarPeticionAppendEntriesTimeout(nodo int, args *ArgAppendEntries,
	reply *Results, timeout int) bool {

	nr.Logger.Println("enviando enviarPeticionAppendEntries al nodo ", nodo)
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.AppendEntries", args, &reply, (time.Duration)(timeout)*time.Millisecond)
	if err != nil {
		return false
	}

	return true
}

func (nr *NodoRaft) mandarHeartbeat(i int) {
	args := &ArgAppendEntries{}
	if len(nr.Log) > 0 {
		args = &ArgAppendEntries{nr.CurrentTerm, nr.Yo, nr.NextIndex[i] - 1, nr.Log[nr.NextIndex[i]-1].Term, nil, nr.CommitIndex} //cambiar arg del append entries (el nil está bien)
	} else {
		args = &ArgAppendEntries{nr.CurrentTerm, nr.Yo, 0, 0, nil, nr.CommitIndex} //cambiar arg del append entries (el nil está bien)
	}

	reply := &Results{}
	nr.enviarPeticionAppendEntries(i, args, reply)
	nr.Logger.Println("respuesta heartbeat del nodo: ", i, " = ", reply, " miTerm = ", nr.CurrentTerm)
	if reply.Success == false {
		// false porque el term es mayor
		if reply.Term > nr.CurrentTerm {
			nr.Mux.Lock()
			nr.CurrentTerm = reply.Term
			nr.Estado = SEGUIDOR
			nr.Mux.Unlock()
		}

		// false por desajuste
		for reply.Success == false && nr.Estado == LIDER {
			nr.NextIndex[i]--
			nr.enviarPeticionAppendEntries(i, args, reply)
		}

		//Reconstruir logs
		for nr.NextIndex[i] <= nr.CommitIndex && nr.Estado == LIDER {
			entries := []EntradaLog{nr.Log[nr.NextIndex[i]]}

			argsReconstruir := &ArgAppendEntries{nr.CurrentTerm, nr.Yo, nr.NextIndex[i] - 1, nr.Log[nr.NextIndex[i]-1].Term, entries, nr.NextIndex[i]}
			//nr.enviarPeticionAppendEntries(i, argsReconstruir, reply)
			err := nr.enviarPeticionAppendEntriesTimeout(i, argsReconstruir, reply, T_HEARTBEAT)
			//err := nr.Nodos[i].CallTimeout("enviarPeticionAppendEntries", argsReconstruir, reply, T_HEARTBEAT*time.Millisecond)
			if !err && reply.Success == true {
				nr.Mux.Lock()
				nr.NextIndex[i]++
				nr.Mux.Unlock()
			}
		}
	}
}

func (nr *NodoRaft) MandarVotacion(i int) {
	args := &ArgsPeticionVoto{nr.CurrentTerm, nr.Yo, nr.LastApplied, 0}

	LastTerm := 0
	if len(nr.Log) > 0 { // no podemos usar la funcion len porque el tamaño puede dejar de ser real  AJUSTAR VARIABLES EN INICIO Y CREAR NUEVAS
		LastTerm = nr.Log[len(nr.Log)-1].Term
		nr.Logger.Println("Longitud del log > 0")

		//condicion = (nr.LastApplied == peticion.LastLogIndex && peticion.LastLogTerm != nr.Log[peticion.LastLogIndex].Term)
	}

	args = &ArgsPeticionVoto{nr.CurrentTerm, nr.Yo, len(nr.Log), LastTerm}

	reply := &RespuestaPeticionVoto{}
	nr.enviarPeticionVoto(i, args, reply)
	nr.Logger.Println("respuesta recibida del nodo: ", i, " = ", reply, " miTerm = ", nr.CurrentTerm)
	if reply.VoteGranted {

		nr.Mux.Lock()
		nr.Votos++
		nr.Mux.Unlock()
		nr.Logger.Println("votos++")
		///nr.Logger.Println("Unlock")
		if nr.Votos > len(nr.Nodos)/2 { //nr.Estado == LIDER esto lo puse pero no se por que, si hay problemas es algo a mirar
			//nr.Logger.Println("lock")
			nr.Mux.Lock()
			nr.IdLider = nr.Yo
			nr.Estado = LIDER
			nr.Mux.Unlock()
			//nr.Logger.Println("Unlock")
			nr.Logger.Println("Done antes de vottacion")
			nr.Done <- true
			nr.Logger.Println("Done despues de vottacion")
		}

	} else if reply.Term > nr.CurrentTerm {
		nr.Mux.Lock()
		nr.CurrentTerm = reply.Term
		nr.Estado = SEGUIDOR
		nr.VotedFor = -1
		nr.Done <- true
		nr.Mux.Unlock()

	}
}

func (nr *NodoRaft) GestionNodo() {
	time.Sleep(5 * time.Second)
	for {
		if nr.CommitIndex > nr.LastApplied {
			nr.LastApplied++
			var operacionCanal AplicaOperacion = AplicaOperacion{nr.LastApplied, nr.Log[nr.LastApplied].Operacion}
			nr.CanalOperacion <- operacionCanal
		}

		if nr.Estado == LIDER {
			nr.Logger.Println("Lider")
			var escribir TipoOperacion = TipoOperacion{"escribir", "1", "a"}

			nr.someterOperacion(escribir)

			//poner todos los next index iguales al commit mio
			if nr.CommitIndex != 0 {
				for index, _ := range nr.NextIndex {
					nr.NextIndex[index] = nr.CommitIndex + 1
				}
			} else {
				for index, _ := range nr.NextIndex {
					nr.NextIndex[index] = 0
				}
			}

			for nr.Estado == LIDER {
				time.Sleep(T_HEARTBEAT * time.Millisecond)

				//Commit de las entradas del registro
				nAvanzados := 0
				for index, elem := range nr.MatchIndex {
					nr.MatchIndex[index] = nr.NextIndex[index] - 1
					if elem > nr.CommitIndex {
						nAvanzados++
					}
				}
				if nAvanzados > len(nr.Nodos)/2 {
					nr.CommitIndex++
				}

				for i := 0; i < len(nr.Nodos); i++ {
					if i != nr.Yo && ((nr.NextIndex[i] >= nr.CommitIndex) || nr.CommitIndex == 0) { //no mandamos hearbeat ni a mi mismo ni a los nodos que estén recuperando la tabla

						nr.Logger.Println("Se va a mandar heartbeat a ", i)
						go nr.mandarHeartbeat(i)
					}
				}
			}

		} else if nr.Estado == SEGUIDOR {

			nr.Logger.Println("Seguidor")
			select {
			case <-nr.heartBeat:
				//Se recibe latido, reset timeout
				nr.Logger.Println("Heartbeat recibido y aceptado")
			case <-time.After((time.Duration)(rand.Intn(T_TIMEOUT_MAX-T_TIMEOUT_MIN)+T_TIMEOUT_MIN) * time.Millisecond):
				//Se termina el timeout, se pasa a candidato

				nr.Mux.Lock()
				nr.Estado = CANDIDATO
				nr.Mux.Unlock()

				nr.Logger.Println("Timeout seguidor")

			}
			//}
		} else {
			//Candidato
			//for nr.Estado == CANDIDATO {
			nr.Logger.Println("Candidato: Empezando votacion")
			//nr.Logger.Println("lock")
			nr.Mux.Lock()
			nr.CurrentTerm++
			nr.VotedFor = nr.Yo
			nr.Votos = 1

			nr.Mux.Unlock()
			//nr.Logger.Println("Unlock")
			for i := 0; i < len(nr.Nodos); i++ {
				if i != nr.Yo {
					go nr.MandarVotacion(i)
				}
			}
			select {
			case <-nr.Done:
				//Lider o seguidor
				nr.Logger.Println("Votacion teminada antes de que acabe el tiempo")
			case <-time.After((time.Duration)(rand.Intn(2500-2000)+2000) * time.Millisecond):
				//Se acaba el tiempo de candidato, empezar nueva votacion
				nr.Logger.Println("Tiempo de votacion terminado")

			}
			//}

		}
	}
}

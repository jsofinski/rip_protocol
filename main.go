package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

type myDataPackage struct {
	fromR   int
	fromH   int
	toR     int
	toH     int
	visited []int
}

type myOffer struct {
	from int
	to   int
	cost int
}

type myPackage struct {
	offers []myOffer
}

type readWriteData struct {
	key     int
	cost    int
	nextHop int
	changed bool
}

type readOp struct {
	key  int
	resp chan readWriteData
}
type writeOp struct {
	data readWriteData
	resp chan bool
}
type writeFalseOp struct {
	key  int
	resp chan bool
}

type MyRoutingTable struct {
	nodeId  int
	nextHop map[int]int
	cost    map[int]int
	changed map[int]bool
}

type Node struct {
	id           int
	neighboursId []int
	routingTable MyRoutingTable
	receiver     chan myPackage
	forwarder    chan myDataPackage
	wg           sync.WaitGroup
}

type Host struct {
	rId      int
	hId      int
	receiver chan myDataPackage
}

var g Graph
var hosts []Host

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (n *Node) String() string {
	return fmt.Sprintf("%v", n.id)
}

type Graph struct {
	nodes []*Node
	edges map[int][]int
	lock  sync.RWMutex
}

func (g *Graph) AddNode(n *Node) {
	g.lock.Lock()
	g.nodes = append(g.nodes, n)
	g.lock.Unlock()
}

func (g *Graph) AddEdge(n1, n2 *Node) {
	g.lock.Lock()
	if g.edges == nil {
		g.edges = make(map[int][]int)
	}
	g.edges[n1.id] = append(g.edges[n1.id], n2.id)
	g.lock.Unlock()
}

// prints whole graph with edges
func (g *Graph) String() {
	g.lock.RLock()
	s := ""
	for i := 0; i < len(g.nodes); i++ {
		s += g.nodes[i].String() + " -> "
		for j := 0; j < len(g.edges[i]); j++ {
			s += g.nodes[g.edges[i][j]].String() + " "
		}
		s += "\n"
	}
	fmt.Println(s)
	g.lock.RUnlock()
}

func fillGraph(size int, randomAmount int) {
	// create all nodes
	for i := 0; i < size; i++ {
		newRoutingTable := MyRoutingTable{i, make(map[int]int), make(map[int]int), make(map[int]bool)}
		node := Node{i, []int{}, newRoutingTable, make(chan myPackage), make(chan myDataPackage), sync.WaitGroup{}}
		g.AddNode(&node)
	}

	// add every edge (i,i+1) up to n-2
	for i := 0; i < size-1; i++ {
		g.AddEdge(g.nodes[i], g.nodes[i+1])
	}

	for i := 0; i < randomAmount; i++ {

		var firstRandom = 0
		var secondRandom = 0

		for {
			rand.Seed(time.Now().UTC().UnixNano() + 1 + int64(i))
			firstRandom = rand.Intn(size)
			rand.Seed(time.Now().UTC().UnixNano() + 1 + int64(i))
			secondRandom = rand.Intn(size)

			if firstRandom != secondRandom {
				if firstRandom != (secondRandom - 1) { // that is not a shortcut
					if secondRandom != (firstRandom - 1) { // that is not a shortcut too
						if !contains(g.edges[firstRandom], secondRandom) { // this shortcut already exists
							if !contains(g.edges[secondRandom], firstRandom) { // this shortcut already exists too
								break // all is fine
							}
						}
					}
				}
			}
		}

		// edge(j,k) only if j < k
		if firstRandom > secondRandom {
			g.AddEdge(g.nodes[secondRandom], g.nodes[firstRandom])
		} else if firstRandom < secondRandom {
			g.AddEdge(g.nodes[firstRandom], g.nodes[secondRandom])
		}
	}
}

func fillNeighbours() {
	for i := 0; i < len(g.edges); i++ {
		for j := 0; j < len(g.edges[i]); j++ {
			var first = i
			var second = g.edges[i][j]
			g.nodes[first].neighboursId = append(g.nodes[first].neighboursId, second)
			g.nodes[second].neighboursId = append(g.nodes[second].neighboursId, first)
		}
	}
}

func fillHosts(numberOfHosts int) {
	for i := 0; i < numberOfHosts; i++ {
		var idOfNewHostInReceiver = 0
		var idOfReceiver = rand.Intn(len(g.nodes))
		for _, element := range hosts {
			if element.rId == idOfReceiver {
				idOfNewHostInReceiver++
			}
		}
		newHost := Host{idOfReceiver, idOfNewHostInReceiver, make(chan myDataPackage)}
		hosts = append(hosts, newHost)
	}
}

func fillRoutingTables() {
	for i := 0; i < len(g.nodes); i++ {
		for j := 0; j < len(g.nodes); j++ {
			if i == j {
				continue
			}
			g.nodes[i].routingTable.changed[j] = true
			if contains(g.nodes[i].neighboursId, j) {
				g.nodes[i].routingTable.nextHop[j] = j
				g.nodes[i].routingTable.cost[j] = 1
			} else {
				if i < j {
					g.nodes[i].routingTable.cost[j] = j - i
					g.nodes[i].routingTable.nextHop[j] = i + 1
				} else {
					g.nodes[i].routingTable.cost[j] = i - j
					g.nodes[i].routingTable.nextHop[j] = i - 1
				}
			}
		}
	}
}

func printNeighbours() {
	fmt.Println("sasiedzi")
	g.lock.RLock()
	s := ""
	for i := 0; i < len(g.nodes); i++ {
		s += g.nodes[i].String() + " -> "
		for j := 0; j < len(g.nodes[i].neighboursId); j++ {
			s += strconv.Itoa(g.nodes[i].neighboursId[j]) + " "
		}

		numberOfHosts := 0
		for _, element := range hosts {
			if element.rId == i {
				numberOfHosts++
			}
		}

		s += " hosts: " + strconv.Itoa(numberOfHosts)
		s += "\n"
	}
	fmt.Println(s)
	g.lock.RUnlock()
}

func nodeRoutine(m *sync.Mutex, messages chan string, id int, minTime float64, maxTime float64) {
	reads := make(chan readOp)
	writes := make(chan writeOp)
	writeChanges := make(chan writeFalseOp)
	//stateful goroutine
	go func() {
		var routingTable MyRoutingTable
		routingTable = g.nodes[id].routingTable

		for {
			select {
			case read := <-reads:
				var response = readWriteData{read.key,
					routingTable.cost[read.key],
					routingTable.nextHop[read.key],
					routingTable.changed[read.key],
				}
				read.resp <- response
			case write := <-writes:
				routingTable.cost[write.data.key] = write.data.cost
				routingTable.nextHop[write.data.key] = write.data.nextHop
				routingTable.changed[write.data.key] = write.data.changed
				write.resp <- true
			case setFalse := <-writeChanges:
				routingTable.changed[setFalse.key] = false
				setFalse.resp <- true
			}
		}
	}()

	//sender
	go func() {
		for {
			var newPackage myPackage
			for i := 0; i < len(g.nodes); i++ {
				if i == id {
					continue
				}
				read := readOp{
					key:  i,
					resp: make(chan readWriteData)}
				reads <- read
				newResp := <-read.resp
				if newResp.changed {
					var newOffer = myOffer{id, i, newResp.cost}
					newPackage.offers = append(newPackage.offers, newOffer)
					writeChange := writeFalseOp{
						key:  i,
						resp: make(chan bool),
					}
					writeChanges <- writeChange
					<-writeChange.resp
				}
			}
			if len(newPackage.offers) > 0 {
				for i := 0; i < len(g.nodes[id].neighboursId)-1; i++ {
					g.nodes[g.nodes[id].neighboursId[i]].receiver <- newPackage
					// messages <- ("Wysylam paczke ofert do " + strconv.Itoa(g.nodes[id].neighboursId[i]) + " do " + strconv.Itoa(id))
				}
			}

			time.Sleep(time.Duration((minTime + rand.Float64()*(maxTime-minTime)) * float64(time.Second)))
			// messages <- "sender from " + strconv.Itoa(id)
		}
	}()

	// var newOffer myOffer
	//receiver
	go func() {
		for {
			// messages <- "receiver from " + strconv.Itoa(id)
			newPackage := <-g.nodes[id].receiver
			// messages <- ("Otrzymalem paczke ofert w " + strconv.Itoa(id) + " od " + strconv.Itoa(newPackage.offers[0].from))
			for i := 0; i < len(newPackage.offers); i++ {
				var newCost = newPackage.offers[i].cost + 1
				read := readOp{
					key:  newPackage.offers[i].to,
					resp: make(chan readWriteData)}
				reads <- read
				newResp := <-read.resp
				if newCost < newResp.cost {
					newData := readWriteData{
						newPackage.offers[i].to,
						newCost,
						newPackage.offers[i].from,
						true,
					}
					write := writeOp{
						data: newData,
						resp: make(chan bool)}
					writes <- write
					<-write.resp
					messages <- ("Zmieniam sciezke w routing table w " + strconv.Itoa(id) + " do " + strconv.Itoa(newPackage.offers[i].to))
				}
			}
			time.Sleep(2 * time.Second)
		}
	}()

	forwarderArray := []myDataPackage{}
	// forwarder receiver
	go func() {
		for {
			newDataPackage := <-g.nodes[id].forwarder
			if len(newDataPackage.visited) == 0 {
				messages <- ("Otrzymalem paczke danych w " + strconv.Itoa(id) + " od mojego hosta " + strconv.Itoa(newDataPackage.fromH))
			} else {
				messages <- ("Otrzymalem paczke danych w " + strconv.Itoa(id) + " od " + strconv.Itoa(newDataPackage.visited[len(newDataPackage.visited)-1]))
			}
			forwarderArray = append(forwarderArray, newDataPackage)
		}
	}()

	// forwarder sender
	go func() {
		for {
			// messages <- "receiver from " + strconv.Itoa(id)
			if len(forwarderArray) > 0 {
				newDataPackage := forwarderArray[0]
				forwarderArray = append(forwarderArray[:0], forwarderArray[1:]...)
				newDataPackage.visited = append(newDataPackage.visited, id)
				if newDataPackage.toR == id {
					for _, element := range hosts {
						if element.rId == newDataPackage.toR && element.hId == newDataPackage.toH {
							element.receiver <- newDataPackage
						}
					}
					messages <- ("Przekazalem swojemu hostowi paczke danych w " + strconv.Itoa(id))
				} else {
					read := readOp{
						key:  newDataPackage.toR,
						resp: make(chan readWriteData)}
					reads <- read
					newResp := <-read.resp
					g.nodes[newResp.nextHop].forwarder <- newDataPackage

					messages <- ("Przekazalem dalej paczke danych w " + strconv.Itoa(id) + " do " + strconv.Itoa(newResp.nextHop))
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
}
func host(hostInHostsId int, rId int, hId int, messages chan string, minTime float64, maxTime float64) {
	var randomHost = rand.Intn(len(hosts))
	for hosts[randomHost].rId == rId {
		randomHost = rand.Intn(len(hosts))
	}
	newDataPackage := myDataPackage{rId, hId, hosts[randomHost].rId, hosts[randomHost].hId, []int{}}
	messages <- ("Host " + strconv.Itoa(hostInHostsId) + ": wysylam paczke od (" + strconv.Itoa(rId) + "," + strconv.Itoa(hId) + ") do (" + strconv.Itoa(hosts[randomHost].rId) + "," + strconv.Itoa(hosts[randomHost].hId) + ") ")
	g.nodes[rId].forwarder <- newDataPackage

	go func() {
		for {
			newDataPackage := <-hosts[hostInHostsId].receiver
			messages <- ("Host " + strconv.Itoa(hostInHostsId) + ": odebrałem paczke w (" + strconv.Itoa(rId) + ", " + strconv.Itoa(hId) + ") od (" + strconv.Itoa(newDataPackage.fromR) + "," + strconv.Itoa(newDataPackage.fromH) + ") ")

			nextDataPackage := myDataPackage{rId, hId, hosts[randomHost].rId, hosts[randomHost].hId, []int{}}
			messages <- ("Host " + strconv.Itoa(hostInHostsId) + ": odsylam paczke od (" + strconv.Itoa(rId) + "," + strconv.Itoa(hId) + ") do (" + strconv.Itoa(hosts[randomHost].rId) + "," + strconv.Itoa(hosts[randomHost].hId) + ") ")
			g.nodes[rId].forwarder <- nextDataPackage
		}
	}()
}

// It prints messages from sender and all routines in order
func channel(messages chan string) {
	for {
		time.Sleep(time.Duration(0.01 * float64(time.Second)))
		msg := <-messages
		fmt.Println(msg)
	}
}

func main() {

	var size = 5
	var randomPaths = 2
	var minTime = 2.0
	var maxTime = 6.0
	var numberOfHosts = 5
	var err error
	commandLineArgs := os.Args[1:]
	if len(commandLineArgs) == 3 {
		if commandLineArgs[0] != "" {
			size, err = strconv.Atoi(commandLineArgs[0])
		}
		if commandLineArgs[1] != "" {
			randomPaths, err = strconv.Atoi(commandLineArgs[1])
		}
		if commandLineArgs[2] != "" {
			numberOfHosts, err = strconv.Atoi(commandLineArgs[2])
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
	}

	var m sync.Mutex
	var w sync.WaitGroup

	fillGraph(size, randomPaths)
	// fmt.Println("Krawawedzie grafu: ")
	// g.String()
	fillNeighbours()
	fillRoutingTables()
	fillHosts(numberOfHosts)
	printNeighbours()

	myChannel := make(chan string)
	go channel(myChannel)

	w.Add(1)
	for i := 0; i < size; i++ {
		go nodeRoutine(&m, myChannel, i, minTime, maxTime)
	}

	for i := 0; i < numberOfHosts; i++ {
		go host(i, hosts[i].rId, hosts[i].hId, myChannel, minTime, maxTime)
	}
	w.Wait()

}

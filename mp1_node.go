package main

import (
	"bufio"
	"container/heap"
	"fmt"
	f "fmt"
	"reflect"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// tracks account balances --> account_number:balance
var accounts = map[string]int{}
// tracks proposed sequence nums from other nodes for txs I sent out  --> tx:array of proposals
var proposals = map[string][]int{} 
// tracks received messages -- used in reliable multicast implementation to only 
// remulticast messages we see for first time. 
var received = map[string]bool{}
// tracks TCP write connections by identifier --> identifier:net.Conn connection 
var other_connections = map[string]net.Conn{}
// tracks largest priority proposals made by self 
var proposal_counter = 0
// tracks largest observed agreed priorities
var largest_agreed_priority = 0
// tracks if messages are deliverable or not --> transaction:true for deliverable, false otherwise
var deliverable = map[string]bool{}
// priority queue of Message Structs 
var pq = make(PriorityQueue, 0)

// An Message is something we manage in a priority queue.
type Message struct {
	transaction string // The transaction; arbitrary.
	priority int    // The priority of the message in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the message in the heap.
}

// A PriorityQueue implements heap.Interface and holds Messages.
type PriorityQueue []*Message

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the lowest, not highest, priority so we use less than here.
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	message := x.(*Message)
	message.index = n
	*pq = append(*pq, message)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	message := old[n-1]
	old[n-1] = nil  // avoid memory leak
	message.index = -1 // for safety
	*pq = old[0 : n-1]
	return message
}

func (pq *PriorityQueue) Peek() (message *Message) {
	return (*pq)[0]
}

// update modifies the priority and value of an Message in the queue.
func (pq *PriorityQueue) update(message *Message, priority int) {
	message.priority = priority
	heap.Fix(pq, message.index)
}

func (pq *PriorityQueue) getQueue() ([]*Message) {
	return *pq
}

// ______________________________________________________________________________

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Prints out all account balances in alphabetic order
func printAccountBalances() {
	keys := make([]string, 0, len(accounts))
 
    for k := range accounts{
        keys = append(keys, k)
    }
    sort.Strings(keys)
 
	f.Print("BALANCES ")
    for _, k := range keys {
        f.Print(k, ":", accounts[k], " ")
    }
	f.Println()
}

// Handles DEPOSIT and TRANSFER transactions 
// Returns true if this was a valid transaction, else returns false
func handleTransaction(transaction string) (bool) {
	split := strings.Split(transaction, " ")
	action := split[0]
	
	if (action == "DEPOSIT") {
		account_number := split[1]
		amount, _ := strconv.Atoi(split[2])

		// we can use this if statement to check to see if 
		// a given key "account_number" exists within a map in Go
		if _, ok := accounts[account_number]; ok {
			accounts[account_number] = accounts[account_number] + amount
		} else {
			accounts[account_number] = amount
		}

	} else if (action == "TRANSFER") {
		source_account := split[1]
		destination_account := split[3]
		amount, _ := strconv.Atoi(split[4])

		if _, ok := accounts[source_account]; ok {
			if(accounts[source_account] >= amount) {
				if _, ok := accounts[destination_account]; ok {
					accounts[destination_account] = accounts[destination_account] + amount
				} else {
					accounts[destination_account] = amount
				}
				accounts[source_account] = accounts[source_account] - amount
			} else {
				printAccountBalances()
				return false
			}
		} else { // source_account doesn't exist, reject
			printAccountBalances()
			return false
		}
	}

	// print all balances in alphabetic order !! 
	printAccountBalances()
	return true 
}

// Parse config file and return all lines in array
func parseConfigFile(identifier string, config_file string) ([]string) {
	// first, parse config_file
	// https://golangdocs.com/golang-read-file-line-by-line
	readFile, err := os.Open(config_file)
  
    if err != nil {
        f.Println(err)
    }

    fileScanner := bufio.NewScanner(readFile)
    fileScanner.Split(bufio.ScanLines)
    var fileLines []string // all the lines in config file as array 
  
    for fileScanner.Scan() {
        fileLines = append(fileLines, fileScanner.Text())
    }
  
    readFile.Close()
  
	return fileLines
}

// node_name refers to the identifier of THIS node 
func sendTransactionData(node_name string, connections []net.Conn) {
	for {
		reader := bufio.NewReader(os.Stdin)
		transaction, err := reader.ReadString('\n')

		if err != nil {
			f.Fprintln(os.Stderr, "fatal err: %s", err.Error())
			os.Exit(1)
		}
		
		valid_tx := handleTransaction(transaction)
		if (!valid_tx) { // not valid tx, don't need to send to other nodes 
			continue
		}

		// ISIS ALG IMPLEMENTATION
		num_nodes := len(connections)
		proposals[transaction] = []int{}
		
		// MULTICAST MESSAGE TO ALL NODES 
		for _, conn := range connections {
			conn.Write([]byte(f.Sprintf("%s %s %s\r\n", "REQUEST_FOR_PROPOSALS", node_name, transaction)))
		} 

		// WAIT FOR ALL PROPOSED SEQUENCE NUMS TO ARRIVE
		// we can assume maximum message delay between any two nodes is 4-5 seconds
		// accordingly, after max 10s all proposed sequence nums must arrive
		// if they don't arrive, that node may be dead
		current_time := time.Now()
		timeout_time := current_time.Add(time.Second * 10) //adds 10 seconds to current time
		for (time.Now() != timeout_time && len(proposals[transaction]) != num_nodes) {
			// waits until timeout or all proposals from other nodes have arrived
		}

		// PICK HIGHEST PROPOSED VALUE AS AGREED_SEQ
		sort.Ints(proposals[transaction]) // sorts in place 
		proposed_seqs := proposals[transaction]
		agreed_seq := proposed_seqs[len(proposed_seqs)-1]

		// MULTICAST AGREED SEQ_NUM TO ALL NODES 
		for _, conn := range connections {
			conn.Write([]byte(f.Sprintf("%s %s %s %d\r\n", "AGREED_SEQ_FOR_PROPOSAL", node_name, transaction, agreed_seq)))
		}
	}
}

func handleIncomingConnections(conn net.Conn, connections []net.Conn){
	// TODO: handle all types of incoming connections and their requests 
	// REQUEST_FOR_PROPOSALS <sender node_name> <transaction string>
	// AGREED_SEQ_FOR_PROPOSAL <sender node_name> <transaction string> <agreed seq number>
	// REPLY_FOR_PROPOSAL <sender node_name> <transaction> <proposed priority> 
	// each incoming message must be remulticast (for reliable multicast) if this is first time seeing message

	for {
		//Read incoming data into string
		reader := bufio.NewReader(conn)
		message, err := reader.ReadString('\n')

		if err != nil {
			f.Fprintln(os.Stderr, "Error reading: ", err.Error())
			os.Exit(1)
		} 

		// RELIABLE MULTICAST IMPLEMENTATION
		if _, exists := received[message]; exists {
			// we have already seen this message and processed it, disregard
			continue 
		}

		// else, this is first time seeing message, process it and re-multicast to other processes

		split := strings.Split(message, " ")
		request_type := split[0]
		sender_node := split[1]

		if (request_type == "REQUEST_FOR_PROPOSALS") {
			priority_proposal := Max(proposal_counter, largest_agreed_priority) + 1
			proposal_counter = priority_proposal
			transaction := split[2] 
			for i := 3; i < len(split)-2; i++ {
				transaction += split[i]
			}
			// REPLY_FOR_PROPOSAL <sender node_name> <transaction> <proposed priority> 
			other_connections[sender_node].Write([]byte(f.Sprintf("%s %s %s %d\r\n", "REPLY_FOR_PROPOSAL", sender_node, transaction, priority_proposal)))

			// store message in priority queue ordered by priority (proposed or agreed)
			tx := &Message{
				transaction: transaction,
				priority: priority_proposal,
			}
			heap.Push(&pq, tx)

			// mark message as undeliverable
			deliverable[transaction] = false

			// REMULTICAST IT
			for _, conn := range connections {
				conn.Write([]byte(f.Sprintf("%s\r\n", message)))
			}  
		} else if (request_type == "AGREED_SEQ_FOR_PROPOSAL") {

			// AGREED_SEQ_FOR_PROPOSAL <sender node_name> <transaction string> <agreed seq number>
			agreed_priority, _ := strconv.Atoi(split[len(split)-1])
			transaction := split[2] 
			for i := 3; i < len(split)-2; i++ {
				transaction += split[i]
			}

			// update largest_agreed_priority
			largest_agreed_priority = Max(largest_agreed_priority, agreed_priority)

			// TODO 
			// Upon receiving agreed (final) priority for a message ‘m’
			// Update m’s priority to final, and accordingly reorder messages in queue.
			all_open_txs := pq.getQueue()
			for _, tx := range all_open_txs {
				if (tx.transaction == transaction) {
					pq.update(tx, agreed_priority)
				}
			}

			// • mark the message m as deliverable.
			deliverable[transaction] = true

			// • deliver any deliverable messages at front of priority queue
			tx := pq.Peek() // heap.Peek(&pq).(*Message)
			for deliverable[tx.transaction] {
				item := heap.Pop(&pq).(*Message)
				handleTransaction(item.transaction)
				tx = pq.Peek() // heap.Peek(&pq).(*Message)
			}

			// REMULTICAST IT
			for _, conn := range connections {
				conn.Write([]byte(f.Sprintf("%s\r\n", message)))
			} 
		} else if (request_type == "REPLY_FOR_PROPOSAL") {
			// appends proposed priority for message to list of proposals in map 
			proposed_priority, _ := strconv.Atoi(split[len(split)-1])
			transaction := split[2] 
			for i := 3; i < len(split)-2; i++ {
				transaction += split[i]
			}

			proposals[transaction] = append(proposals[transaction], proposed_priority)
		}
	}
}

func main() {

	// ./mp1_node <identifier> <configuration file>
	if len(os.Args) < 2 {
		f.Fprintln(os.Stderr, "too few arguments")
		os.Exit(1)
	}

	// initialize priority queue 
	heap.Init(&pq)

	identifier := os.Args[1]
	config_file := os.Args[2]

	fileLines := parseConfigFile(identifier, config_file)

	num_nodes, _ := strconv.Atoi(fileLines[0])
	num_nodes = num_nodes - 1 // don't count self node
	fmt.Println(reflect.TypeOf(num_nodes))
	var listener net.Listener
	var connections []net.Conn = make([]net.Conn, num_nodes)
 
	// Sets up all TCP connections for node. 
	// Each node must listen for TCP connections from other nodes, 
	// as well as initiate a TCP connection to each of the other nodes
	counter := 0
    for i, line := range fileLines {
		// contains num files in file
		if(i == 0) {continue} 
		
		split := strings.Split(line, " ")
		node_name := split[0]
		host_name := split[1]
		port_number := split[2]

		if(identifier == node_name) { // set up listen for incoming connections
			listener, error := net.Listen("tcp", host_name+":"+port_number)
			if error != nil {
				f.Fprintln(os.Stderr, "Error listening:", error.Error())
				os.Exit(1)
			}
			defer listener.Close()
		} else { // set up connections with all other nodes in process 
			conn, err := net.Dial("tcp", host_name+":"+port_number)
			for err != nil { // retry connection until it's successful
				conn, err = net.Dial("tcp", host_name+":"+port_number)
			}
			connections[counter] = conn
			other_connections[node_name] = conn 
			counter = counter + 1
		}
    }

	for {
		//Listen for a new connection
		conn, error := listener.Accept()
		if error != nil {
			f.Fprintln(os.Stderr, "Error accepting: ", error.Error())
			os.Exit(1)
		}

		//Send new connections to handler
		go handleIncomingConnections(conn, connections) // handle connections/transaction messages from incoming nodes 
		go sendTransactionData(identifier, connections) // send transaction messages to other nodes 
	}

	handleTransaction("DEPOSIT wqkby 10")
	handleTransaction("DEPOSIT yxpqg 75")
	handleTransaction("TRANSFER yxpqg -> wqkby 13")

}
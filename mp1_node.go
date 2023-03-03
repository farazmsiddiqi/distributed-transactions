package main

import (
	"fmt"
	"strings"
	"sort"
	"strconv"
	"os"
	"net"
)

// tracks account balances --> account_number:balance
var accounts = map[string]int{}
// tracks proposed sequence nums from other nodes for txs I sent out  --> tx:array of proposals
var proposals = map[string][]int{} 

// Prints out all account balances in alphabetic order
func printAccountBalances() {
	keys := make([]string, 0, len(accounts))
 
    for k := range accounts{
        keys = append(keys, k)
    }
    sort.Strings(keys)
 
	fmt.Print("BALANCES ")
    for _, k := range keys {
        fmt.Print(k, ":", accounts[k], " ")
    }
	fmt.Println()
}

// Handles DEPOSIT and TRANSFER transactions 
// Returns true if this was a valid transaction, else returns false
func bool handleTransaction(transaction string) {
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
func parseConfigFile(identifier string, config_file string) {
	// first, parse config_file
	// https://golangdocs.com/golang-read-file-line-by-line
	readFile, err := os.Open(config_file)
  
    if err != nil {
        fmt.Println(err)
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

func sendTransactionData(connections []net.Conn) {
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
		proposals[transaction] = {}
		
		// MULTICAST MESSAGE TO ALL NODES 
		// TODO: need to add message-type encoding: like this is a REQUEST_FOR_PROPOSALS
		// TODO: do we need message ids ???
		for i, conn := range connections {
			conn.Write([]byte(f.Sprintf("%s %s\r\n", "REQUEST_FOR_PROPOSALS", transaction)))
		} 

		// WAIT FOR ALL PROPOSED SEQUENCE NUMS TO ARRIVE
		// we can assume maximum message delay between any two nodes is 4-5 seconds
		// accordingly, after max 10s all proposed sequence nums must arrive
		// if they don't arrive, that node may be dead
		current_time := time.Now()
		timeout_time := current_time.Add(time.Second * 10) //adds 10 seconds to current time
		while (time.Now() != timeout_time && len(proposals[transaction]) != num_nodes) {
			// waits until timeout or all proposals from other nodes have arrived
		}

		// PICK HIGHEST PROPOSED VALUE AS AGREED_SEQ
		sort.Ints(values)  // sort the values finding min and max element
		proposed_seqs = sort.Ints(proposals[transaction])
		agreed_seq = proposed_seqs[len(proposed_seqs)-1]

		// MULTICAST AGREED SEQ_NUM TO ALL NODES 
		// TODO: need to add message-type encoding: like this is a AGREED_SEQ_FOR_PROPOSAL 
		for i, conn := range connections {
			conn.Write([]byte(f.Sprintf("%s %s %d\r\n", "AGREED_SEQ_FOR_PROPOSAL", transaction, agreed_seq)))
		}
	}
}

func handleIncomingConnections(conn net.Conn){
	// TODO: handle all types of incoming connections and their requests 
	// REQUEST_FOR_PROPOSALS <transaction string>
	// AGREED_SEQ_FOR_PROPOSAL <transaction string> <agreed seq number>
	// each incoming message must be remulticast (for reliable multicast) if this is first time seeing message
}

func main() {

	// ./mp1_node <identifier> <configuration file>
	if len(os.Args) < 2 {
		f.Fprintln(os.Stderr, "too few arguments")
		os.Exit(1)
	}

	identifier := os.Args[1]
	config_file := os.Args[2]

	fileLines := parseConfigFile(identifier, config_file)

	var num_nodes int = strconv.Atoi(fileLines[0]) - 1
	var listener net.Conn
	var connections [num_nodes]net.Conn 
 
	// Sets up all TCP connections for node. 
	// Each node must listen for TCP connections from other nodes, 
	// as well as initiate a TCP connection to each of the other nodes
	counter = 0
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
				fmt.Fprintln(os.Stderr, "Error listening:", error.Error())
				os.Exit(1)
			}
			defer listener.Close()
		} else { // set up connections with all other nodes in process 
			conn, err := net.Dial("tcp", host_name+":"+port_number)
			for err != nil { // retry connection until it's successful
				conn, err := net.Dial("tcp", host_name+":"+port_number)
			}
			connections[counter] = conn
			counter = counter + 1
		}
    }

	for {
		//Listen for a new connection
		conn, error := listener.Accept()
		if error != nil {
			fmt.Fprintln(os.Stderr, "Error accepting: ", error.Error())
			os.Exit(1)
		}

		//Send new connections to handler
		go handleIncomingConnections(conn) // handle connections/transaction messages from incoming nodes 
		go sendTransactionData(connections) // send transaction messages to other nodes 
	}

	handleTransaction("DEPOSIT wqkby 10")
	handleTransaction("DEPOSIT yxpqg 75")
	handleTransaction("TRANSFER yxpqg -> wqkby 13")

}
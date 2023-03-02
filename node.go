package main

import (
	"fmt"
	"strings"
	"sort"
	"strconv"
)

var accounts = map[string]int{}

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

func handleTransaction(transaction string) {
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
			}
		} else { // source_account doesn't exist, reject
		}
	}

	// print all balances in alphabetic order !! 
	printAccountBalances()
}

func main() {
	handleTransaction("DEPOSIT wqkby 10")
	handleTransaction("DEPOSIT yxpqg 75")
	handleTransaction("TRANSFER yxpqg -> wqkby 13")


}
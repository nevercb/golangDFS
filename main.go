package main

import "dfs/network"

func main() {
	port := "8080"
	network.StartServer(port)
}

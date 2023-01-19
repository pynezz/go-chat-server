package main

import (
	"fmt"
	"net"
	"os"

	"github.com/go-redis/redis"
)

func initRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("DB_ADDR"),
		Password: os.Getenv("DB_PASS"),
		DB:       0, // = default DB
	})

	// Check connection
	_, err := client.Ping().Result()
	if err != nil {
		fmt.Println("Error connecting to Redis")
		os.Exit(1)
	}

	return client
}

func handleConnection(conn net.Conn, client *redis.Client) {

	defer conn.Close()

	for {
		// Read data from connection
		buffer := make([]byte, 1024) // Max message length is 1024 byte. Characters are 1 byte long in Go, max message length is 1024 characters.

		n, err := conn.Read(buffer) // Read data from connection and store it in buffer
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

		message := string(buffer[:n]) // Convert buffer to string
		fmt.Println("Message received: ", message)

		// Store message in Redis
		err := client.Set("message", message, 0).Err()
	}
}

func main() {

	// Initialize Redis
	client := initRedis()

	// Listen for incoming connections
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Println("Error listening: ", err)
		os.Exit(1)
	}

	defer listener.Close()
	fmt.Println("Listening on port 8081")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			os.Exit(1)
		}

		go handleConnection(conn, client)
	}
}

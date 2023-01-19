package main

import (
	"fmt"
	"net"
	"os"

	"github.com/gomodule/redigo/redis"
)

// func initRedis() *redis.Client {
// 	client := redis.NewClient(&redis.Options{
// 		Addr:     os.Getenv("DB_ADDR"),
// 		Password: os.Getenv("DB_PASS"),
// 		DB:       0, // = default DB
// 	})

// 	// Check connection
// 	_, err := client.Ping().Result()
// 	if err != nil {
// 		fmt.Println("Error connecting to Redis")
// 		os.Exit(1)
// 	}

// 	return client
// }

func handleConnection(nconn net.Conn) {

	defer nconn.Close()

	for {
		// Read data from connection
		buffer := make([]byte, 1024) // Max message length is 1024 byte. Characters are 1 byte long in Go, max message length is 1024 characters.

		n, err := nconn.Read(buffer) // Read data from connection and store it in buffer
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

		message := string(buffer[:n]) // Convert buffer to string
		fmt.Println("Message received: ", message)

		c, err := redis.DialURL(os.Getenv("REDIS_URL"), redis.DialTLSSkipVerify(true))
		if err != nil {
			// Handle error
		}
		defer c.Close()

		// Save message to cache
		_, err = c.Do("SET", message, message)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Send message back to client
		_, err = nconn.Write(buffer[:n])
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

	}
}

func main() {

	// Listen for incoming connections
	listener, err := net.Listen("tcp", os.Getenv("PORT"))
	if err != nil {
		fmt.Println("Error listening: ", err)
		os.Exit(1)
	}

	defer listener.Close()
	fmt.Println("Listening on ", listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			continue // Continue to next iteration of loop, even if there is an error
		}

		go handleConnection(conn)
	}
}

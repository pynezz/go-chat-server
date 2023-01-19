package main

import (
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
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

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "The current time is: %s\n", time.Now())

	if r.Method == "POST" {
		fmt.Fprintf(w, "POST request\n")

		r.GetBody()
		fmt.Fprintf(w, "Message: %s\n", r.Form.Get("message"))
	}
}

var upgrader = websocket.Upgrader{} // use default options

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer ws.Close()

	for {
		mt, message, err := ws.ReadMessage()

		if err != nil {
			fmt.Fprintf(w, "%+v", err)
		}
		fmt.Printf("Received: %s", message)
		err = ws.WriteMessage(mt, message)
		if err != nil {
			fmt.Fprintf(w, "%+v", err)
		}
	}
}

func main() {
	http.HandleFunc("/", home)
	http.HandleFunc("/ws", websocketHandler)

	http.HandleFunc("/chat", handler)

	port := os.Getenv("PORT")
	http.ListenAndServe(":"+port, nil)

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

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/ws")
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>
const PORT = process.env.PORT;

window.addEventListener("load", function(evt) {
    var pport = document.getElementById("port");
	var pstatus = document.getElementById("status");
	

	pport.innerHTML = PORT;
	pstatus.innerHTML = "Connected";

});

const testConnectionBtn = document.getElementById("test-conn");
testConnectionBtn.addEventListener("click", function(evt) {
	evt.preventDefault();
	testConnection();
});


function testConnection() {
	var pconnectionStatus = document.getElementById("connection-status");
	var poutput = document.getElementById("output");
	var input = document.getElementById("input").value;

	fetch("http://https://termichat.herokuapp.com/chat", {
		method: "POST",
		headers: {
			"Content-Type": "application/json"
		},
		body: JSON.stringify({
			message: input
		})
	})
	.then(function(response) {
		return response.json();
	})
	.then(function(data) {
		console.log(data);
		pconnectionStatus.innerHTML = "Sent: " + input;
		poutput.innerHTML += "Sent: " + input + " " + new Date().toLocaleString();
	})
	.catch(function(err) {
		console.log(err);
		pconnectionStatus.innerHTML = "Sent: " + input;
		poutput.innerHTML += "Sent: " + input + " " + new Date().toLocaleString();
	});
}
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<h2>This is a simple dashboard displaying server status and logs.</h2>

<form>
<p id="port">---</p>
<p id="status">---</p>
<p><input id="input" type="text" value="Hello world!">
<button id="test-conn">Test Connection</button>
<p id="connection-status">---</p>
</form>
</td><td valign="top" width="50%">
<div id="output" style="max-height: 70vh;overflow-y: scroll;"></div>
</td></tr></table>
</body>
</html>
`))

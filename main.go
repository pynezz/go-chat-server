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

type Client struct {
	id   string
	conn *websocket.Conn
	room *Room
	send chan *Message
}

func (c *Client) read() {
	defer func() {
		c.room.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Set read deadline to 60 secs from now if no message is received

	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Add more time(60s) to the read deadline if a pong is received
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage() // Read JSON message from client
		if err != nil {
			log.Println("Error reading message: ", err)
			break
		}

		c.room.broadcast <- &Message{
			Message:  string(message),
			Type:     "message",
			ClientId: c.id,
		}
	}
}

func (c *Client) write() {
	ticker := time.NewTicker(50 * time.Second) // Send ping every 50 seconds
	defer func() {
		c.conn.Close()
		ticker.Stop()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(60 * time.Second)) // Set write deadline to 60 seconds from now

			if !ok { // Check if channel is closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte("Connection is closed"))
				return
			}

			c.conn.WriteJSON(message)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(60 * time.Second)) // Set write deadline to 60 seconds from now
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func newClient(id string, room *Room, w http.ResponseWriter, r *http.Request) *Client {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection: ", err)
		return nil
	}

	c := &Client{
		id:   id,
		conn: conn,
		room: room,
		send: make(chan *Message),
	}

	go c.read()
	go c.write()

	return c
}

// 			w, err := c.conn.NextWriter(websocket.TextMessage)
// 			if err != nil {
// 				return
// 			}

// 			w.Write([]byte(message.Message))

// 			if err := w.Close(); err != nil {
// 				return
// 			}
// 		}
// 	}
// }

func handleConnection(nconn net.Conn) {
	fmt.Println("handleConnection called")

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
	fmt.Println("Handler called")

	fmt.Println("Print: GET request")
	fmt.Fprintf(w, "GET request\n")

	fmt.Fprintf(w, "The current time is: %s\n", time.Now())

	if r.Method == "POST" {
		r.ParseForm()

		fmt.Println("Print: POST request")
		fmt.Fprintf(w, "POST request\n")

		r.GetBody()
		fmt.Fprintf(w, "Message: %s\n", r.Form.Get("message"))
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "https://"+r.Host+"/")

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

var upgrader = websocket.Upgrader{} // use default options

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Websockethandler called") // Works without SSL/TLS
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	room := newRoom()
	id := r.URL.Query().Get("id")

	if id == "" {
		fmt.Println("Error: ID is not set")
		return
	}

	client := newClient(id, room, w, r)
	room.register <- client

	fmt.Println("Client registered in room ", client.room)

	defer ws.Close()

	// for {
	// 	mt, message, err := ws.ReadMessage()

	// 	if err != nil {
	// 		fmt.Fprintf(w, "%+v", err)
	// 	}
	// 	fmt.Printf("Received: %s", message)
	// 	err = ws.WriteMessage(mt, message)
	// 	if err != nil {
	// 		fmt.Fprintf(w, "%+v", err)
	// 	}
	// 	home(w, r)
	// }

}

func main() {
	http.HandleFunc("/", home)
	http.HandleFunc("/test", websocketHandler)

	http.HandleFunc("/chat", handler)

	fmt.Println("Starting server on port 3333")
	port := os.Getenv("PORT")
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	fmt.Println("Server started on port ", port)

}

func test(w http.ResponseWriter, r *http.Request) {
	testTemplate.Execute(w, "ws://"+r.Host+"/ws")

	// if (r.FormValue("message")) != "" {
	// 	fmt.Println("Message received: ", r.FormValue("message")) // Dette virker
	// }
}

var testTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">

<style>
    body {
        font-family: Arial, Helvetica, sans-serif;
    }
</style>

<script>
window.addEventListener("load", function(evt) {
    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;
    var print = function(message) {
        var d = document.createElement("div");
        d.textContent = message;
        output.appendChild(d);
        output.scroll(0, output.scrollHeight);
    };
    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("RESPONSE: " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };
    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
        ws.send(input.value);
        return false;
    };
    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server,
"Send" to send a message to the server and "Close" to close the connection.
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><input id="input" type="text" value="Hello world!">
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output" style="max-height: 70vh;overflow-y: scroll;"></div>
</td></tr></table>
</body>
</html>
`))

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
body {
	font-family: Arial, Helvetica, sans-serif;
}
</style>

<script>
const PORT = process.env.PORT || 3333;

const socket = new WebSocket("ws://localhost:" + PORT);

window.onload = () => {

	window.addEventListener("onload", function(evt) {
		var pport = document.getElementById("port");
		var pstatus = document.getElementById("status");
		

		pport.innerHTML = 3333;
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

	socket.addEventListener("open", function(evt) {
		console.log("Connection open ...");
		socket.send("Hello Server!");
	});

	socket.addEventListener("message", function(evt) {
		console.log("Received Message: " + evt.data);
		socket.close();
	});

	socket.addEventListener("close", function(evt) {
		console.log("Connection closed.");
	});

	socket.addEventListener("error", function(evt) {
		console.log("Error: " + evt.data);
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

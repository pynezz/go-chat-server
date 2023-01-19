const PORT = process.env.PORT || 3333;

const socket = new WebSocket("ws://localhost:" + PORT);

window.onload = () => {

	window.addEventListener("onload", function(e) {
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


var elPrice = document.getElementById("price")
var elVolume = document.getElementById("volume")
// Crea una nueva conexión.
const socket = new WebSocket('ws://localhost:8080/price');

// Abre la conexión
socket.addEventListener('open', function (event) {
    socket.send('Hello Server!');
});

// Escucha por mensajes
socket.addEventListener('message', function (event) {
    var data = JSON.parse(event.data)
    console.log('Message from server', data);
    elPrice.innerHTML = `USD ${data.avgPrice.toFixed(8)}`;
    elVolume.innerHTML = `USD ${data.avgVolumen.toFixed(10)}`;
});
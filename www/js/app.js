
var client;
var active = false;
var connectTimeout;

var deviceId = ""

fetch("http://"+location.hostname+"/devices")
  .then(resp => resp.json())
  .then(devices => {
    device = devices.find(device => device.name == "MEKOB-P")
    if (!device) {
      status("MEKOB-P device not found.");
      return
    }
    deviceId = device.id;
    connect();
  })

function connect() {
  status("Connecting...")
  var id = "dashboard-"+(Math.random()+"").substr(2, 8);
  client = new Messaging.Client(location.hostname, 80, id);
  client.onConnectionLost = onConnectionLost;
  client.onMessageArrived = onMessageArrived;
  client.connect({onSuccess:onConnect, onFailure: onFailure});
}

function onConnect() {
  status("Connected.");
  client.subscribe("devices/"+deviceId+"/sensors/gps/value");
  client.subscribe("devices/"+deviceId+"/sensors/position/value");
  // message = new Messaging.Message("Hello");
  // message.destinationName = "/World";
  // client.send(message);
  // client.disconnect();
}

function onConnectionLost(responseObject) {
  if (!active) return
  if (responseObject.errorCode !== 0)
    console.log("lost:", responseObject.errorMessage);
}

function onFailure(err) {
  if (!active) return
  status("Error: ["+err.errorCode+"] "+err.errorMessage)
  connectTimeout = setTimeout(connect, 5000);
  console.log(err);
}

function onMessageArrived(message) {
  var data = JSON.parse(message.payloadString);
  var topic = message.destinationName;
  if(topic.endsWith("/.acc0/value"))
    onData(data)
  else if(topic.endsWith("/gps/value"))
    onGPSData(data)
  else if(topic.endsWith("/position/value"))
    onPositionData(data)
}

function status(text) {
  $("#status").text(text);
}



function start() {
  $("#start").hide()
  $("#stop, #chart-box").show()
  active = true;
  fetch("http://"+location.hostname+"/devices/"+deviceId+"/actuators/event/value", {
    method: "POST",
    body: JSON.stringify("start")
  });
  // document.documentElement.requestFullscreen()
  client.subscribe("devices/"+deviceId+"/sensors/.acc0/value");
}

function stop() {
  $("#stop").hide()
  $("#start").show()
  active = false;
  fetch("http://"+location.hostname+"/devices/"+deviceId+"/actuators/event/value", {
    method: "POST",
    body: JSON.stringify("stop")
  });
  // clearTimeout(connectTimeout);
  // document.webkitCancelFullScreen()
  // $("#status").text("Press 'Start' to begin.");
  client.unsubscribe("devices/"+deviceId+"/sensors/.acc0/value");
}

////////////////////////////////////////////////////////////////////////////////

$("#views").on("change", (event) => {

  const value = event.target.value;
  switch(value) {
    case "w":
      $("#chartA, #chartP").hide();
      $("#tracker").show();
      break;

    default:
      $("#tracker").hide();
      $("#chartA, #chartP").show();
      break;
  }
});

const track = $("#track")[0];

var trackWidth = window.innerWidth - 80 - 17
track.width = trackWidth;
var trackHeight = (window.innerWidth - 80 - 17)/4*3
track.height = trackHeight;
var trackings = [];
const ctx = track.getContext("2d");
var lastTime = 0;

function onPositionData(msg) {
  trackings.push(msg);
  ctx.clearRect(0, 0, track.width, track.height);

  if (trackings.length > 20) {
    trackings.unshift();
  }

  for (var i=0; i<trackings.length-1; i++) {
    ctx.beginPath();
    ctx.moveTo(trackings[i].x/4*trackWidth, trackings[i].y/3*trackHeight);
    ctx.lineTo(trackings[i+1].x/4*trackWidth, trackings[i+1].y/3*trackHeight);
    ctx.stroke();

    ctx.beginPath();
    ctx.rect(0, 0, track.width, track.height);
    ctx.fillStyle = "rgba(255, 255, 255, 0.1)";
    ctx.fill();
  }

  ctx.beginPath();
  ctx.arc(msg.x/4*trackWidth, msg.y/3*trackHeight, msg.accuracyRadius/4*trackWidth, 0, 2 * Math.PI);
  ctx.strokeStyle = 'blue';
  ctx.stroke();
  ctx.fillStyle = "rgba(80, 80, 255, 0.1)";
  ctx.fill();

  ctx.beginPath();
  ctx.arc(msg.x/4*trackWidth, msg.y/3*trackHeight, 3, 0, 2 * Math.PI);
  ctx.fillStyle = "blue";
  ctx.fill();

  var time = new Date(parseInt(msg.timestamp));
  if (lastTime != 0) {
    var dx = msg.x-trackings[trackings.length-2].x
    var dy = msg.y-trackings[trackings.length-2].y
    var d = Math.sqrt(Math.pow(dx, 2)+Math.pow(dy, 2))
    var dt = time-lastTime;
    if (dt != 0) {
      var speed = d/dt*3600;
      if (speed < 0.3) {
        $("#speed").text("Walking Speed: Standing");
      } else {
        $("#speed").text("Walking Speed: "+speed.toFixed(2)+"m/s");
      }
    }
  }
  lastTime = time;
}

////////////////////////////////////////////////////////////////////////////////

var chartA = new Chart('chartA', {
	type: 'line',
	data: {
  	labels: [],
  	datasets: [{
  		borderColor: "rgba(0, 0, 0, 0)",
  		data: [],
  		hidden: false,
      fill: false,
  		label: "Min-Z",
      pointRadius: 0,
      borderWidth: 0,
  	}, {
  		backgroundColor: "rgba(202, 223, 255, .5)", // grey
  		borderColor: "rgb(162, 179, 204)",  // orange
      borderWidth: 2,
  		data: [],
  		label: "Z",
  		fill: "-1"
  	}, {
  		backgroundColor: "rgba(202, 223, 255, .5)",
  		borderColor: "rgba(0, 0, 0, 0)",
  		data: [],
  		hidden: false,
  		label: "Max-Z",
  		fill: "-1",
      pointRadius: 0,
      borderWidth: 0,
  	}]
  },
	options: {
  	maintainAspectRatio: false,
  	spanGaps: false,
  	elements: {
  		line: {
  			tension: 0.000001
  		}
  	},
  	scales: {
      yAxes: [{
        ticks: {
          padding: -80,
          beginAtZero: true,
        },
      }]
  		//yAxes: [{
  		//	stacked: true
  		//}]
  	},
  	plugins: {
  		filler: {
  			propagate: false
  		}
  	}
  }
});

////////////////////////////////////////////////////////////////////////////////


var chartP = new Chart('chartP', {
	type: 'line',
	data: {
  	labels: [],
  	datasets: [{
  		borderColor: "rgb(162, 179, 204)",
      borderWidth: 2,
  		data: [],
  		label: "Force"
  	}]
  },
	options: {
  	maintainAspectRatio: false,
  	spanGaps: false,
  	elements: {
  		line: {
  			tension: 0.000001
  		}
  	},
  	scales: {
      yAxes: [{
        ticks: {
          padding: -80,
          beginAtZero: true,
        },
      }]
  		//yAxes: [{
  		//	stacked: true
  		//}]
  	},
  	plugins: {
  		filler: {
  			propagate: false
  		}
  	}
  }
});

function onData(msg) {
  if (!active) return

  if (msg.AValues != 0) {
    data = chartA.data;
    data.labels.push("");
    data.datasets[0].data.push(msg.Min.Z);
    data.datasets[1].data.push(msg.Mean.Z);
    data.datasets[2].data.push(msg.Max.Z);
    while(data.labels.length > 20) {
      data.labels.shift();
      data.datasets[0].data.shift();
      data.datasets[1].data.shift();
      data.datasets[2].data.shift();
    }
    chartA.update();
  }

  //

  if (msg.PValues != 0) {
    data = chartP.data;
    data.labels.push("");
    data.datasets[0].data.push(msg.Force[0]);
    while(data.labels.length > 20) {
      data.labels.shift();
      data.datasets[0].data.shift();
    }
    chartP.update();
  }
}

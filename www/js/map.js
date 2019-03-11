function toggleMap() {
  var map = document.getElementById("map");
  map.classList.toggle("opened");
}

var view = new ol.View({
  center: [0, 0],
  zoom: 2
});

var map = new ol.Map({
  layers: [
    new ol.layer.Tile({
      source: new ol.source.OSM()
    })
  ],
  target: 'map',
  controls: ol.control.defaults({
    attributionOptions: /** @type {olx.control.AttributionOptions} */ ({
      collapsible: false
    })
  }),
  view: view
});

var positionFeature = new ol.Feature();
positionFeature.setStyle(new ol.style.Style({
  image: new ol.style.Circle({
    radius: 6,
    fill: new ol.style.Fill({
      color: '#3399CC'
    }),
    stroke: new ol.style.Stroke({
      color: '#fff',
      width: 2
    })
  })
}));

function onGPSData(data) {
  var point = ol.proj.transform([data.Long, data.Lat], 'EPSG:4326', 'EPSG:3857');
  if (!positionFeature.getGeometry()) {
    map.getView().setZoom(19);
    map.getView().setCenter(point);
  }
  positionFeature.setGeometry(new ol.geom.Point(point));
  // console.log(data);
}

new ol.layer.Vector({
  map: map,
  source: new ol.source.Vector({
    features: [positionFeature]
  })
});

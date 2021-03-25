window.addEventListener("load", function load(event){

    var api_key = document.body.getAttribute("data-nextzen-api-key");
    var style_url = document.body.getAttribute("data-nextzen-style-url");
    var tile_url = document.body.getAttribute("data-nextzen-tile-url");

    /*
    if (! api_key){
	console.log("Missing API key");
	return;
    }
    
    if (! style_url){
	console.log("Missing style URL");
	return;
    }
    
    if (! tile_url){
	console.log("Missing tile URL");
	return;
    }
     */
    
    var pip_wrapper = document.getElementById("point-in-polygon");

    if (! pip_wrapper){
	console.log("Missing 'point-in-polygon' element.");
	return;
    }
    
    var init_lat = pip_wrapper.getAttribute("data-initial-latitude");

    if (! init_lat){
	console.log("Missing initial latitude");
	return;
    }
    
    var init_lon = pip_wrapper.getAttribute("data-initial-longitude");

    if (! init_lon){
	console.log("Missing initial longitude");
	return;
    }
    
    var init_zoom = pip_wrapper.getAttribute("data-initial-zoom");    

    if (! init_zoom){
	console.log("Missing initial zoom");
	return;
    }
    
    var map_el = document.getElementById("map");

    if (! map_el){
	console.log("Missing map element");	
	return;
    }
    
    var map_args = {
	"api_key": api_key,
	"style_url": style_url,
	"tile_url": tile_url,
    };

    // we need to do this _before_ Tangram starts trying to draw things
    // map_el.style.display = "block";
    
    var map = whosonfirst.spatial.maps.getMap(map_el, map_args);

    if (! map){
	console.log("Unable to instantiate map");
	return;
    }

    var hash = new L.Hash(map);
    
    var layers = L.layerGroup();
    layers.addTo(map);

    var spinner = new L.Control.Spinner();
    // map.addControl(spinner);
    
    var update_map = function(e){

	var pos = map.getCenter();	

	var args = {
	    'latitude': pos['lat'],
	    'longitude': pos['lng'],
	};
	
	var properties = [];

	var extra_properties = document.getElementById("extras");

	if (extra_properties){

	    var extras = extra_properties.value;

	    if (extras){
		properties = extras.split(",");
		args['properties'] = properties;
	    }
	}
	
	var existential_filters = document.getElementsByClassName("point-in-polygon-filter-existential");
	var count_existential = existential_filters.length;

	for (var i=0; i < count_existential; i++){

	    var el = existential_filters[i];

	    if (! el.checked){
		continue;
	    }
	    
	    var fl = el.value;
	    args[fl] = [ 1 ];
	}

	var placetypes = [];
	
	var placetype_filters = document.getElementsByClassName("point-in-polygon-filter-placetype");	
	var count_placetypes = placetype_filters.length;

	for (var i=0; i < count_placetypes; i++){

	    var el = placetype_filters[i];

	    if (! el.checked){
		continue;
	    }

	    var pt = el.value;
	    placetypes.push(pt);
	}

	if (placetypes.length > 0){
	    args['placetypes'] = placetypes;
	}

	var edtf_filters = document.getElementsByClassName("point-in-polygon-filter-edtf");
	var count_edtf = edtf_filters.length;

	for (var i=0; i < count_edtf; i++){

	    var el = edtf_filters[i];

	    var id = el.getAttribute("id");

	    if (! id.match("^(inception|cessation)$")){
		continue
	    }

	    var value = el.value;

	    if (value == ""){
		continue;
	    }
	    
	    // TO DO: VALIDATE EDTF HERE WITH WASM
	    // https://millsfield.sfomuseum.org/blog/2021/01/14/edtf/

	    var key = id + "_date";
	    args[key] = value;
	};

	var show_feature = function(id){

	    var data_root = document.body.getAttribute("data-root");

	    if (!data_root.endsWith("/")){
		data_root = data_root + "/";
	    }
	    
	    var url = data_root + id;

	    var on_success = function(data){

		var l = L.geoJSON(data, {
		    style: function(feature){
			return whosonfirst.spatial.pip.named_style("match");
		    },
		});
		
		layers.addLayer(l);
		l.bringToFront();
	    };

	    var on_fail= function(err){
		console.log("SAD", id, err);
	    }
	    
	    whosonfirst.net.fetch(url, on_success, on_fail);
	};
	
	var on_success = function(rsp){

	    map.removeControl(spinner);
	    
	    var places = rsp["places"];
	    var count = places.length;

	    var matches = document.getElementById("pip-matches");
	    matches.innerHTML = "";
	    
	    if (! count){
		return;
	    }
	    
	    for (var i=0; i < count; i++){
		var pl = places[i];
		show_feature(pl["wof:id"]);
	    }
	    
	    var table_props = whosonfirst.spatial.pip.default_properties();

	    // START OF something something something
	    
	    var extras_el = document.getElementById("extras");

	    if (extras_el){
		
		var str_extras = extras_el.value;
		var extras = null;
		
		if (str_extras){
		    extras = str_extras.split(",");  		    
		}

		if (extras){

		    var first = places[0];
		    
		    var count_extras = extras.length;		    
		    var extra_props = [];
		    
		    for (var i=0; i < count_extras; i++){

			var ex = extras[i];
			
			if ((ex.endsWith(":")) || (ex.endsWith(":*"))){
			    
			    var prefix = ex.replace("*", "");
			    
			    for (k in first){
				if (k.startsWith(prefix)){
				    extra_props.push(k);
				}
			    }
			    
			} else {

			    if (first[ex]) {
				extra_props.push(ex);
			    }
			}
		    }

		    for (idx in extra_props){
			var ex = extra_props[idx];
			table_props[ex] = "";
		    }
		}

	    }

	    // END OF something something something
	    
	    var table = whosonfirst.spatial.pip.render_properties_table(places, table_props);
	    matches.appendChild(table);
	    
	};

	var on_error = function(err){

	    var matches = document.getElementById("pip-matches");
	    matches.innerHTML = "";
	    
	    map.removeControl(spinner);	    
	    console.log("SAD", err);
	}

	whosonfirst.spatial.api.point_in_polygon(args, on_success, on_error);

	map.addControl(spinner);	
	layers.clearLayers();	
    };
    
    map.on("moveend", update_map);

    var filters = document.getElementsByClassName("point-in-polygon-filter");
    var count_filters = filters.length;
    
    for (var i=0; i < count_filters; i++){	    
	var el = filters[i];
	el.onchange = update_map;
    }

    var extras = document.getElementsByClassName("point-in-polygon-extra");
    var count_extras = extras.length;
    
    for (var i=0; i < count_extras; i++){	    
	var el = extras[i];
	el.onchange = update_map;
    }
    
    var hash_str = location.hash;

    if (hash_str){

	var parsed = whosonfirst.spatial.maps.parseHash(hash_str);

	if (parsed){
	    init_lat = parsed['latitude'];
	    init_lon = parsed['longitude'];
	    init_zoom = parsed['zoom'];
	}
    }
    
    map.setView([init_lat, init_lon], init_zoom);    

    slippymap.crosshairs.init(map);    
});

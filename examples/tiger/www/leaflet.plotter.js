L.Polyline.plotter = L.Polyline.extend({
    _lineMarkers: [],
    _editIcon: L.divIcon({className: 'leaflet-div-icon leaflet-editing-icon'}),
    _halfwayPointMarkers: [],
    _existingLatLngs: [],
    options: {
        weight: 2,
        color: '#000',
        unchangable: false
    },
    initialize: function (latlngs, options){
        this._setExistingLatLngs(latlngs);
        L.Polyline.prototype.initialize.call(this, [], options);
    },
    onAdd: function (map) {
        L.Polyline.prototype.onAdd.call(this, map);
        this._map = map;
        this._plotExisting();

        if (this.options.unchangable === false) {
            this._bindMapClick();
        }
    },
    clear: function() {
        this.setLatLngs([]);
        for (index in this._halfwayPointMarkers) {
            index = parseInt(index, 10);
            this._map.removeLayer(this._halfwayPointMarkers[index]);
        }
        this._halfwayPointMarkers = [];

        for (index in this._lineMarkers) {
            index = parseInt(index, 10);
            this._map.removeLayer(this._lineMarkers[index]);
        }
        this._lineMarkers = [];

    },
    changeCallback: function(cb) {
        callback = cb;
    },
    setLatLngs: function(latlngs){
        L.Polyline.prototype.setLatLngs.call(this, latlngs);
    },
    _bindMapClick: function(){
        if (this.options.unchangable === false) {
            this._map.on('click', this._addNewMarker, this);
        }
    },
    _setExistingLatLngs: function(latlngs){
        this._existingLatLngs = latlngs;
    },
    _replot: function(){
        this._redraw();
        this._redrawHalfwayPoints();
    },
    _getNewMarker: function(latlng, options){
        options.draggable = !this.options.unchangable;
        return new L.marker(latlng, options);
    },
    _canClick: true,
    _addToMapAndBindMarker: function(newMarker){
        newMarker.addTo(this._map);
        
        if (this.options.unchangable === true) {
            return;
        }

        newMarker.on('click', this._removePoint, this);
        newMarker.on('drag', function (e) {
            this._canClick = false;
            
            // One weird hack to prevent very short drags from being considered clicks
            var that = this;
            setTimeout(function() {
                that._canClick = true;
            }, 500);

            this._replot(e);
        }, this);
    },
    _removePoint: function(e){
        var index = this._lineMarkers.indexOf(e.target);
        if (index >= 0) {
            this._map.removeLayer(this._lineMarkers[index]);
            this._lineMarkers.splice(index, 1);
        }
        this._replot();
        this.fireEvent('remove-node');
    },
    _addNewMarker: function(e){
        if (this._canClick === false) {
            return;
        }
        var newMarker = this._getNewMarker(e.latlng, { icon: this._editIcon });
        this._addToMapAndBindMarker(newMarker);
        this._lineMarkers.push(newMarker);
        this._replot();

        this.fireEvent('add-node');
    },
    _redrawHalfwayPoints: function(){
        var i, that = this;

        for (i = 0 ; i < this._halfwayPointMarkers.length; i++) {
            this._map.removeLayer(this._halfwayPointMarkers[i]);
        }
        this._halfwayPointMarkers = [];
        for (i = 0; i < this._lineMarkers.length; i++) {
            if (typeof this._lineMarkers[i + 1] === 'undefined') {
                return;
            }
            var halfwayMarker = new L.Marker([
                (this._lineMarkers[i].getLatLng().lat + this._lineMarkers[i + 1].getLatLng().lat) / 2,
                (this._lineMarkers[i].getLatLng().lng + this._lineMarkers[i + 1].getLatLng().lng) / 2
            ], { icon: this._editIcon, opacity: 0.5, draggable: !this.options.unchangable }).addTo(this._map);
            halfwayMarker.index = i;
            
            if (this.options.unchangable === false) {
                halfwayMarker.on('mousedown', function(marker) {
                    return function (e) {
                        that._addHalfwayPoint(e, marker, that);
                    };
                }(halfwayMarker), this);
            }

            this._halfwayPointMarkers.push(halfwayMarker);
        }
    },
    _addHalfwayPoint: function(e, marker, self){
        marker.setOpacity(1.0);
        self._halfwayPointMarkers.splice(marker.index, 1);
        self._addToMapAndBindMarker(marker);
        self._lineMarkers.splice(e.target.index + 1, 0, marker);
        self._replot();

        marker.on('click', self._removePoint, self);
        marker.off('mousedown');
        self.fireEvent('add-node');
    },
    _plotExisting: function(){
        for(index in this._existingLatLngs){
            this._addNewMarker({
                latlng: new L.LatLng(this._existingLatLngs[index][0], this._existingLatLngs[index][1])
            });
        }
    },
    _redraw: function(){
        this.setLatLngs([]);
        this.redraw();
        for (index in this._lineMarkers) {
            this.addLatLng(this._lineMarkers[index].getLatLng());
        }
        this.redraw();
    }
});

L.Polyline.Plotter = function(latlngs, options){
	return new L.Polyline.plotter(latlngs, options);
};
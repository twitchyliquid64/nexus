var app = angular.module('recon', ['ui.materialize', 'angularMoment']);

var crosshairsIcon = {
  url: "/static/img/cross-hairs.gif",
  size: new google.maps.Size(30, 30),
  origin: new google.maps.Point(0, 0),
  anchor: new google.maps.Point(7, 7),
  scaledSize: new google.maps.Size(14, 14)
};

var spotIcon = {
  url: "/static/img/bluecircle.png",
  size: new google.maps.Size(30, 30),
  origin: new google.maps.Point(0, 0),
  anchor: new google.maps.Point(8, 8),
  scaledSize: new google.maps.Size(16, 16)
};

app.controller('BodyController', ["$scope", "$rootScope", "$location", "$http", "$window", function ($scope, $rootScope, $location, $http, $window) {
  var self = this;
  $scope.loading = false;
  $scope.uid = parseInt(window.location.href.substr(window.location.href.lastIndexOf('/') + 1));
  $scope.lastUpdated = moment();
  $scope.from = null;
  $scope.to = null;

  self.initMap = function(){
    var centerlatlng = new google.maps.LatLng(-33.915803, 151.195242);
    var myOptions = {
        zoom: 15,
        center: centerlatlng,
        mapTypeId: google.maps.MapTypeId.ROADMAP
    };
    $scope.map = new google.maps.Map(document.getElementById("map_canvas"), myOptions);
    $scope.getInfo();
  }

  $scope.refreshMap = function(){
    if (self.line)self.line.setMap(null);
    var points = [];
    for (var i = 0; i < $scope.records.length; i++) {
      points[points.length] = {lat: $scope.records[i].Lat, lng: $scope.records[i].Lon}
    }
    var line = new google.maps.Polyline({
      path: points,
      strokeColor: '#FF0000',
      strokeOpacity: 1.0,
      strokeWeight: 2
    })
    line.setMap($scope.map);
    self.line = line;

    var bounds = new google.maps.LatLngBounds();
    var points = line.getPath().getArray();
    for (var n = 0; n < points.length ; n++){
        bounds.extend(points[n]);
    }
    $scope.map.fitBounds(bounds);
  }

  $scope.getInfo = function(){
    $scope.from = moment($('#startDate').val(), 'DD MMM, YYYY').unix() + (864 * $('#timeSliderLeft').val());
    $scope.to = moment($('#endDate').val(), 'DD MMM, YYYY').unix() + (864 * $('#timeSliderRight').val());
    $scope.loading = true;
    $http({
      method: 'POST',
      url: '/app/recon/api/location',
      data: {
        UID: $scope.uid,
        Start: $scope.from,
        End: $scope.to,
      }
    }).then(function successCallback(response) {
      $scope.records = response.data;
      $scope.loading = false;
      $scope.lastUpdated = moment();
      $scope.refreshMap();
    }, function errorCallback(response) {
      $scope.loading = false;
    });
  }

  self.initMap();

  var dateTimeChangeHandler = function(){
    $scope.$apply(function(){
      $scope.getInfo();
    });
  }
  $scope.endDate = $("#endDate").pickadate({
    selectMonths: true,
    selectYears: 5,
    today: 'Today',
    clear: 'Clear',
    close: 'Ok',
    closeOnSelect: true,
    onSet: dateTimeChangeHandler,
  });
  $scope.toDate = $("#startDate").pickadate({
    selectMonths: true,
    selectYears: 5,
    today: 'Today',
    clear: 'Clear',
    close: 'Ok',
    closeOnSelect: true,
    onSet: dateTimeChangeHandler,
  });
  $('#timeSliderLeft').change(dateTimeChangeHandler);
  $('#timeSliderRight').change(dateTimeChangeHandler);

}]);

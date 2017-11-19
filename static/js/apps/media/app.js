$.urlParam = function (name) {
  var results = new RegExp('[\?&]' + name + '=([^&#]*)').exec(window.location.href);
  if (results) {
    return results[1] || 0;
  }
  return null;
}

var app = angular.module('media', ['ui.materialize', 'angularMoment']);


app.directive('loader', function($rootScope){
  return {
    scope: {
      loading: '=',
      error: '=',
    },
    //restrict E means its can only be used as an element.
    restrict: 'E',
    template: '<div class="progress" ng-show="loading"><div class="indeterminate"></div></div>  <blockquote ng-show="error"><h5>Error</h5>' +
        '<ul class="collection">' +
        '<li class="collection-item"><b>Error code</b>: {{ec()}}</li>' +
        '<li class="collection-item"><b>Explanation</b>: {{exp()}}</li>' +
        '<li class="collection-item"><b>The server said</b>: {{error.data}}{{error.reason}}</li>' +
        '</ul></blockquote>',
    link: function($scope, elem, attrs) {
      // scope = either parent scope or its own child scope if scope set.
      // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
    },
    controller: function($scope) {
      $scope.ec = function(){
        if (!$scope.error)return null;
        if ($scope.error.success === false)
          return 'N/A';
        return $scope.error.status;
      }
      $scope.exp = function(){
        if (!$scope.error)return null;
        if ($scope.error.status === -1)
          return "Network Error or server offline";
        if ($scope.error.success === false)
          return 'The server encountered a problem handling the request';
        return $scope.error.statusText;
      }
    },
  };
});

String.prototype.lpad = function(padString, length) {
    var str = this;
    while (str.length < length)
        str = padString + str;
    return str;
}

app.filter('musicSecs', [function() {
    return function(seconds) {
      var mins = Math.floor(seconds/60);
      var secs = Math.floor(seconds%60);
      if (secs < 10)
        secs = "0" + new String(secs);
      return mins + ":" + secs;
    };
}]);

app.controller('BodyController', ["$scope", "$rootScope", "$location", "$http", "$window", function ($scope, $rootScope, $location, $http, $window) {
  $scope.path = $.urlParam('path');
  $scope.path = $scope.path ? decodeURIComponent($scope.path) : $scope.path;
  $scope.isListMode = !$scope.path;

  $scope.dataSources = [];
  $scope.dataSourceFilter = {Kind: 2};

  $scope.files = [];
  $scope.filesFilter = {};
  $scope.msg = 'Please select a media file to begin playing.'
  $scope.audioElement = document.createElement('audio');
  $scope.audioElement.addEventListener("timeupdate",function(){
    $scope.$apply(function(){
      $scope.aTime = $scope.audioElement.currentTime;
      $scope.aDuration = $scope.audioElement.duration;
      $("#positionSlider").val($scope.aTime * 100 / $scope.aDuration);
    });
  });
  $scope.audioElement.addEventListener("loadedmetadata",function(){
    $scope.$apply(function(){
      $scope.aTime = $scope.audioElement.currentTime;
      $scope.aDuration = $scope.audioElement.duration;
      $("#positionSlider").val('0');
    });
  });
  $("#positionSlider").change(function(){
    $scope.audioElement.currentTime = $("#positionSlider").val() * $scope.aDuration / 100;
  });
  $scope.aTime = $scope.aDuration = $scope.playPercent = 0;

  $scope.getSources = function(){
    $scope.loading = true;
    $http({
      method: 'GET',
      url: '/app/media/sources',
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.dataSources = response.data;
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.getFiles = function(){
    $scope.loading = true;
    $http({
      method: 'POST',
      url: '/app/media/files',
      data: {path: $scope.path},
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.files = response.data;
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.icon = function(file){
    if (file.ItemKind == 3)
      return 'folder';

    if (file.Name.endsWith('.mp3'))
      return 'music_note';
    if (file.Name.endsWith('.mp4') || file.Name.endsWith('.mpg') || file.Name.endsWith('.mpeg') || file.Name.endsWith('.avi'))
      return 'movie';
  }

  $scope.playButton = function(){
    $scope.audioElement.play();
  }
  $scope.pauseButton = function(){
    $scope.audioElement.pause();
  }

  $scope.play = function(file){
    if ($scope.audioElement) {
      $scope.audioElement.pause();
    }
    var fname = file.Name.split('/');
    fname = fname[fname.length-1];

    if (file.Name.endsWith('.mp3')) {
      $scope.loading = true;
      $scope.msg = fname;
      $http({
        method: 'POST',
        url: '/app/media/getURL',
        data: {path: $scope.path + '/' + fname},
      }).then(function successCallback(response) {
        $scope.loading = false;
        $scope.audioElement.setAttribute('src', response.data.url);
        $scope.audioElement.canplay = $scope.audioElement.play();
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    } else if (file.Name.endsWith('.mp4')) {
      $scope.loading = true;
      $http({
        method: 'POST',
        url: '/app/media/getURL',
        data: {path: $scope.path + '/' + fname, video: true},
      }).then(function successCallback(response) {
        $scope.loading = false;
        $window.location.href = '/app/media/vid?path=' + encodeURIComponent(response.data.url) + '&backurl=' + encodeURIComponent($window.location.href);
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }
  }

  if ($scope.isListMode) {
    $scope.getSources();
  } else {
    $scope.getFiles();
  }
}]);





app.controller('VideoController', ["$scope", "$rootScope", "$location", "$http", "$window", function ($scope, $rootScope, $location, $http, $window) {
  $scope.path = decodeURIComponent($.urlParam('path'));
  $scope.backurl = decodeURIComponent($.urlParam('backurl'));
  $('#mvid').html('\
  <video class="responsive-video" controls>\
    <source src="' + $scope.path + '" type="video/mp4">\
  </video>\
  ')
}]);

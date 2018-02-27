var app = angular.module('terminal', ['ui.materialize', 'angularMoment']);

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

app.directive('commandDisplay', function($rootScope){
  return {
    scope: {
      result: '=',
      cmd: '=',
      sql: '=',
      when: '=',
    },
    //restrict E means its can only be used as an element.
    restrict: 'E',
    template: '<div class="message-section" style="margin-top: 14px;">' +
        '<h5 style="font-size: 1.2rem; display: inline-block; margin: 0.52rem 0 0.156rem 0;">{{cmd}}</h5><span am-time-ago="when" class="msg-time"></span>' +
        '<span class="command-marker" ng-class="{{badgeClasses()}}">{{badge()}}</span>' +
        '<div style="margin: 6px;">' +
        '<blockquote ng-if="!result.success"><b class="red-text">Error: </b> {{result.error}}</blockquote>' +
        '<div ng-if="result.success && sql">' +
          '<span ng-if="result.result.affected"><b>{{result.result.affected}}</b> rows affected.</span>' +
          '<div ng-if="result.result.rows">' +
            '<table class="highlight">' +
              '<thead ng-if="result.result.columns"><tr><th ng-repeat="col in result.result.columns">{{col}}</th></tr></thead>' +
              '<tbody><tr ng-repeat="row in result.result.rows">' +
                '<td ng-repeat="d in row">{{d}}</td>' +
              '</tr></tbody>' +
            '</table>' +
          '</div>' +
        '</div>' +
        '<div ng-if="result.success && !sql">' +
        '</div>' +
        '</div>' +
        '</div>',
    link: function($scope, elem, attrs) {
      // scope = either parent scope or its own child scope if scope set.
      // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
    },
    controller: function($scope) {
      $scope.badge = function(){
        return $scope.sql ? 'SQL' : 'raw';
      }

      $scope.badgeClasses = function(){
        return $scope.sql ? ["green", "white-text"] : ["blue", "white-text"];
      }
    },
  };
});

app.directive('shortcut', function() {
  return {
    restrict: 'E',
    replace: true,
    scope: true,
    link:    function postLink(scope, iElement, iAttrs){
      $(document).on('keydown', function(e){
         scope.$apply(scope.keyPressed(e));
       });
    }
  };
});

app.controller('BodyController', ["$scope", "$rootScope", "$location", "$http", "$window", function ($scope, $rootScope, $location, $http, $window) {
  var self = this;
  $scope.loading = false;
  $scope.lastCommand = null;
  $scope.command = '';
  $scope.sqlEnabled = true;
  $scope.commands = [];

  $scope.doScroll = function(){
    $scope.$$postDigest(function(){
      var objDiv = document.getElementById("messages-container");
      objDiv.scrollTop = objDiv.scrollHeight;
    });
  }

  function intercept(cmd) {
    if (cmd.toUpperCase() == 'SET SQL') {
      $scope.sqlEnabled = true;
      return true;
    }
    if (cmd.toUpperCase() == 'SET RAW') {
      $scope.sqlEnabled = false;
      return true;
    }
    if (cmd.toUpperCase() == 'TGM') {
      $scope.sqlEnabled = !$scope.sqlEnabled;
      return true;
    }
    if (cmd.toUpperCase() == 'CLEAR') {
      $scope.commands = [];
      return true;
    }
    return false;
  }

  $scope.keyPressed = function(e) {
    if (e.key != "End")return;
    $scope.sqlEnabled = !$scope.sqlEnabled;
  }

  $scope.rawCmd = function(cmd, sqlEnabled) {
    $scope.loading = false;
    $scope.commands.push({
      result: {success:false, error: "Raw commands are not yet supported."},
      cmd: cmd,
      sql: sqlEnabled,
      when: moment(),
    });
    $scope.doScroll();
  }

  $scope.run = function(){
    var cmd = $scope.command;
    var sqlEnabled = $scope.sqlEnabled;

    if (intercept(cmd)) {
      $scope.command = '';
      return;
    }

    $scope.lastCommand = cmd;
    $scope.command = '';
    $scope.loading = true;

    var req = null;

    if (sqlEnabled) {
      req = $http({
        method: 'POST',
        url: '/app/terminal/query',
        data: {
          query: cmd,
        }
      })
    } else {
      return $scope.rawCmd(cmd, sqlEnabled);
    }

    req.then(function successCallback(response) {
      $scope.commands.push({
        result: response.data,
        cmd: cmd,
        sql: sqlEnabled,
        when: moment(),
      });
      $scope.loading = false;
      $scope.doScroll();
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.commands.push({
        result: {success:false, error: JSON.stringify(response)},
        cmd: cmd,
        sql: sqlEnabled,
        when: moment(),
      });
      $scope.doScroll();
    });
  };
}]);
